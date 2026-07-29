//go:build linux

package observability

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

type linuxProcessSampler struct {
	procRoot string
}

func newHostProcessSampler() (hostProcessSampler, error) {
	sampler := &linuxProcessSampler{procRoot: "/proc"}
	if _, err := sampler.Sample(os.Getpid()); err != nil {
		if errors.Is(err, ErrUnsupportedHostMetric) {
			return nil, err
		}
		return nil, &UnsupportedHostMetricError{Metric: "linux-proc", Reason: "self-sampling unavailable"}
	}
	return sampler, nil
}

func (s *linuxProcessSampler) Source() string { return "linux-proc-and-rusage-observed" }

func (s *linuxProcessSampler) Sample(pid int) (HostProcessSample, error) {
	if pid <= 0 {
		return HostProcessSample{}, fmt.Errorf("invalid process identifier")
	}
	processRoot := filepath.Join(s.procRoot, strconv.Itoa(pid))
	rss, peak, err := readLinuxMemory(filepath.Join(processRoot, "status"))
	if err != nil {
		return HostProcessSample{}, err
	}
	cpu, err := readLinuxCPUTime(filepath.Join(processRoot, "schedstat"))
	if err != nil {
		return HostProcessSample{}, err
	}
	fds, sockets, err := readLinuxDescriptors(filepath.Join(processRoot, "fd"))
	if err != nil {
		return HostProcessSample{}, err
	}
	return HostProcessSample{
		CPUTime: cpu, RSSBytes: rss, PeakMemoryBytes: peak,
		OpenFDs: fds, OpenSockets: sockets,
	}, nil
}

func (s *linuxProcessSampler) FinalPeakMemoryBytes(state stateProcessState) (int64, error) {
	usage, ok := state.SysUsage().(*syscall.Rusage)
	if !ok || usage == nil || usage.Maxrss < 0 || usage.Maxrss > math.MaxInt64/1024 {
		return 0, &UnsupportedHostMetricError{Metric: "peak-memory", Reason: "Linux rusage max RSS unavailable"}
	}
	return usage.Maxrss * 1024, nil
}

func readLinuxMemory(path string) (int64, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, errObservedProcessExited
		}
		return 0, 0, &UnsupportedHostMetricError{Metric: "rss-and-peak-memory", Reason: "Linux process status unavailable"}
	}
	defer file.Close()

	values := map[string]int64{}
	processState := ""
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "State:") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				processState = fields[1]
			}
		}
		if strings.HasPrefix(line, "VmRSS:") {
			value, err := parseLinuxKiB(line)
			if err != nil {
				return 0, 0, err
			}
			values["VmRSS"] = value
		}
		if strings.HasPrefix(line, "VmHWM:") {
			value, err := parseLinuxKiB(line)
			if err != nil {
				return 0, 0, err
			}
			values["VmHWM"] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return 0, 0, &UnsupportedHostMetricError{Metric: "rss-and-peak-memory", Reason: "Linux process status unreadable"}
	}
	rss, hasRSS := values["VmRSS"]
	peak, hasPeak := values["VmHWM"]
	if !hasRSS || !hasPeak {
		if processState == "Z" || processState == "X" {
			return 0, 0, errObservedProcessExited
		}
		return 0, 0, &UnsupportedHostMetricError{Metric: "rss-and-peak-memory", Reason: "VmRSS or VmHWM missing"}
	}
	if peak < rss {
		return 0, 0, &UnsupportedHostMetricError{Metric: "rss-and-peak-memory", Reason: "VmHWM is lower than VmRSS"}
	}
	return rss, peak, nil
}

func parseLinuxKiB(line string) (int64, error) {
	fields := strings.Fields(line)
	if len(fields) != 3 || fields[2] != "kB" {
		return 0, &UnsupportedHostMetricError{Metric: "rss-and-peak-memory", Reason: "unsupported Linux memory unit"}
	}
	value, err := strconv.ParseInt(fields[1], 10, 64)
	if err != nil || value < 0 || value > math.MaxInt64/1024 {
		return 0, &UnsupportedHostMetricError{Metric: "rss-and-peak-memory", Reason: "invalid Linux memory counter"}
	}
	return value * 1024, nil
}

func readLinuxCPUTime(path string) (time.Duration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, errObservedProcessExited
		}
		return 0, &UnsupportedHostMetricError{Metric: "cpu-time", Reason: "Linux schedstat unavailable"}
	}
	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, &UnsupportedHostMetricError{Metric: "cpu-time", Reason: "Linux schedstat runtime missing"}
	}
	nanos, err := strconv.ParseUint(fields[0], 10, 64)
	if err != nil || nanos > math.MaxInt64 {
		return 0, &UnsupportedHostMetricError{Metric: "cpu-time", Reason: "invalid Linux schedstat runtime"}
	}
	return time.Duration(nanos), nil
}

func readLinuxDescriptors(path string) (int, int, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, 0, errObservedProcessExited
		}
		return 0, 0, &UnsupportedHostMetricError{Metric: "file-descriptors-and-sockets", Reason: "Linux fd directory unavailable"}
	}
	fds, sockets := 0, 0
	for _, entry := range entries {
		target, err := os.Readlink(filepath.Join(path, entry.Name()))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return 0, 0, &UnsupportedHostMetricError{Metric: "file-descriptors-and-sockets", Reason: "Linux fd metadata unavailable"}
		}
		fds++
		// /proc/<pid>/fd symlinks identify sockets as socket:[inode]. Stat
		// follows the link and observes the target inode, which is racy and
		// does not reliably preserve the descriptor type.
		if strings.HasPrefix(target, "socket:[") && strings.HasSuffix(target, "]") {
			sockets++
		}
	}
	return fds, sockets, nil
}
