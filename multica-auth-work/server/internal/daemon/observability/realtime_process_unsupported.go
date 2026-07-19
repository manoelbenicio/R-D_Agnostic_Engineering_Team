//go:build !linux

package observability

func newHostProcessSampler() (hostProcessSampler, error) {
	return nil, &UnsupportedHostMetricError{
		Metric: "process-runtime-rss-memory-fds-sockets",
		Reason: "only Linux procfs/rusage collection is implemented",
	}
}
