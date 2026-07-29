package observability

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"slices"
)

const (
	DevelopmentProfileVersion = "g4-development-20.v1"
	SyntheticClockEpochMillis = int64(1784342400000)
)

type SyntheticTask struct {
	TaskID          string         `json:"task_id"`
	Protocol        ProtocolFamily `json:"protocol"`
	PayloadClass    string         `json:"payload_class"`
	Streaming       bool           `json:"streaming"`
	ToolRequest     bool           `json:"tool_request"`
	ParallelTools   bool           `json:"parallel_tools"`
	Continuation    bool           `json:"continuation"`
	ConnectionSlot  string         `json:"connection_slot"`
	FailureCaseID   string         `json:"failure_case_id,omitempty"`
	ArrivalMillis   int64          `json:"arrival_millis"`
	StartMillis     int64          `json:"start_millis,omitempty"`
	QueueMillis     int64          `json:"queue_millis"`
	SelectionMillis int64          `json:"selection_millis"`
	TTFTMillis      int64          `json:"ttft_millis,omitempty"`
	DurationMillis  int64          `json:"duration_millis"`
	Outcome         string         `json:"outcome"`
	RetryCount      int            `json:"retry_count"`
	FallbackCount   int            `json:"fallback_count"`
	Started         bool           `json:"started"`
}

type Percentiles struct {
	P50 int64 `json:"p50"`
	P95 int64 `json:"p95"`
	P99 int64 `json:"p99"`
}

type RunCounts struct {
	Offered   int `json:"offered"`
	Admitted  int `json:"admitted"`
	Queued    int `json:"queued"`
	Rejected  int `json:"rejected"`
	Started   int `json:"started"`
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
}

type SlotDistribution struct {
	Slot                string `json:"slot"`
	IndependentRequests int    `json:"independent_requests"`
	ExpectedRequests    int    `json:"expected_requests"`
	AbsoluteDeviation   int    `json:"absolute_deviation"`
}

type ResourceModel struct {
	Source          string `json:"source"`
	PeakActive      int    `json:"peak_active"`
	CPUMilliPeak    int    `json:"cpu_milli_peak"`
	MemoryBytesPeak int64  `json:"memory_bytes_peak"`
	SocketPeak      int    `json:"socket_peak"`
}

type DevelopmentRunResult struct {
	SchemaVersion        string                 `json:"schema_version"`
	ProfileVersion       string                 `json:"profile_version"`
	RunID                string                 `json:"run_id"`
	SyntheticOnly        bool                   `json:"synthetic_only"`
	LiveEndpointUsed     bool                   `json:"live_endpoint_used"`
	CapacityTierEnabled  bool                   `json:"capacity_tier_enabled"`
	AcceptanceClaim      bool                   `json:"acceptance_claim"`
	VirtualEpochMillis   int64                  `json:"virtual_epoch_millis"`
	ConcurrencyLimit     int                    `json:"concurrency_limit"`
	Counts               RunCounts              `json:"counts"`
	PeakQueue            int                    `json:"peak_queue"`
	SelectionLatency     Percentiles            `json:"selection_latency_millis"`
	QueueLatency         Percentiles            `json:"queue_latency_millis"`
	FirstOutputLatency   Percentiles            `json:"first_output_latency_millis"`
	RequestLatency       Percentiles            `json:"request_latency_millis"`
	Retries              int                    `json:"retries"`
	Fallbacks            int                    `json:"fallbacks"`
	ProtocolCounts       map[ProtocolFamily]int `json:"protocol_counts"`
	PayloadCounts        map[string]int         `json:"payload_counts"`
	StreamingCount       int                    `json:"streaming_count"`
	ToolRequestCount     int                    `json:"tool_request_count"`
	ParallelToolCount    int                    `json:"parallel_tool_count"`
	IndependentCount     int                    `json:"independent_count"`
	ContinuationCount    int                    `json:"continuation_count"`
	SlotDistribution     []SlotDistribution     `json:"slot_distribution"`
	FairnessDeviationPct int                    `json:"fairness_deviation_percent"`
	Resources            ResourceModel          `json:"resources"`
	FailureCasesCovered  []string               `json:"failure_cases_covered"`
	Tasks                []SyntheticTask        `json:"tasks"`
	Blockers             []string               `json:"blockers"`
}

func RunDevelopment20Profile() DevelopmentRunResult {
	const concurrency = 4
	availableAt := make([]int64, concurrency)
	tasks := make([]SyntheticTask, 0, 20)
	independentIndex := 0
	peakQueue := 0

	failureByTask := map[int]string{
		2: "FAIL-ACCOUNT-DISABLE", 3: "FAIL-ACCESS-EXPIRY", 4: "FAIL-REFRESH-REVOKE",
		5: "FAIL-QUOTA", 6: "FAIL-429-ACCOUNT", 7: "FAIL-429-GLOBAL",
		8: "FAIL-UPSTREAM-MATRIX", 9: "FAIL-SSE-PREPOST", 13: "FAIL-SSE-PREPOST",
		16: "FAIL-CANCEL", 17: "FAIL-HOT-ACCOUNT", 18: "FAIL-CONTINUATION",
		19: "FAIL-RESTART", 20: "FAIL-CANCEL",
	}

	for number := 1; number <= 20; number++ {
		arrival := int64(number-1) * 50
		continuation := number%5 == 0
		slot := ""
		if continuation {
			slot = tasks[len(tasks)-1].ConnectionSlot
		} else {
			slot = fmt.Sprintf("slot-%d", independentIndex%4+1)
			independentIndex++
		}

		worker := earliestWorker(availableAt)
		start := max64(arrival, availableAt[worker])
		queueMillis := start - arrival
		queuedAhead := 0
		for _, prior := range tasks {
			if prior.Started && prior.StartMillis-SyntheticClockEpochMillis > arrival {
				queuedAhead++
			}
		}
		if queueMillis > 0 {
			queuedAhead++
		}
		peakQueue = maxInt(peakQueue, queuedAhead)

		task := SyntheticTask{
			TaskID:          fmt.Sprintf("task-%02d", number),
			Protocol:        protocolForTask(number),
			PayloadClass:    payloadForTask(number),
			Streaming:       (number-1)%10 < 7,
			ToolRequest:     (number-1)%5 < 2,
			ParallelTools:   number == 4 || number == 11 || number == 18,
			Continuation:    continuation,
			ConnectionSlot:  slot,
			FailureCaseID:   failureByTask[number],
			ArrivalMillis:   SyntheticClockEpochMillis + arrival,
			QueueMillis:     queueMillis,
			SelectionMillis: int64(2 + (number*7)%9),
			DurationMillis:  int64(180 + (number*53)%220),
			Outcome:         "completed",
			Started:         true,
		}
		if task.Streaming {
			task.TTFTMillis = int64(40 + (number*13)%61)
		}
		applySyntheticFailure(number, &task)
		if number == 16 && queueMillis > 0 {
			task.Started = false
			task.StartMillis = 0
			task.DurationMillis = 0
		} else {
			task.StartMillis = SyntheticClockEpochMillis + start
			availableAt[worker] = start + task.DurationMillis
		}
		tasks = append(tasks, task)
	}

	result := summarizeDevelopmentRun(tasks, peakQueue, concurrency)
	digest := sha256.Sum256([]byte(DevelopmentProfileVersion + ":synthetic-only:20:4"))
	result.RunID = "synthetic-" + hex.EncodeToString(digest[:8])
	return result
}

func applySyntheticFailure(number int, task *SyntheticTask) {
	switch number {
	case 2, 4, 5, 7:
		task.FallbackCount = 1
	case 3, 6, 8, 9:
		task.RetryCount = 1
	case 13:
		task.Outcome = "failed-after-output"
		task.RetryCount = 0
		task.FallbackCount = 0
	case 16:
		task.Outcome = "cancelled-queued"
	case 20:
		task.Outcome = "cancelled-active"
		task.DurationMillis = 80
	}
}

func summarizeDevelopmentRun(tasks []SyntheticTask, peakQueue, concurrency int) DevelopmentRunResult {
	result := DevelopmentRunResult{
		SchemaVersion:       EvidenceSchemaVersion,
		ProfileVersion:      DevelopmentProfileVersion,
		SyntheticOnly:       true,
		VirtualEpochMillis:  SyntheticClockEpochMillis,
		ConcurrencyLimit:    concurrency,
		Counts:              RunCounts{Offered: len(tasks), Admitted: len(tasks)},
		PeakQueue:           peakQueue,
		ProtocolCounts:      map[ProtocolFamily]int{},
		PayloadCounts:       map[string]int{},
		Tasks:               tasks,
		FailureCasesCovered: failureIDs(tasks),
		Blockers: []string{
			"g4-gateway-tests.md must exist and pass independent validation",
			"g4-runtime-isolation.md must exist and pass independent validation",
			"G3 security correction artifacts and independent pB re-review must be accepted",
			"synthetic modeled resources are not tier-20 host measurements",
			"task 9.2 tier enablement is Codex1-only and remains gated",
		},
	}

	var selection, queue, firstOutput, request []int64
	slotCounts := map[string]int{}
	var intervals [][2]int64
	for _, task := range tasks {
		result.ProtocolCounts[task.Protocol]++
		result.PayloadCounts[task.PayloadClass]++
		selection = append(selection, task.SelectionMillis)
		queue = append(queue, task.QueueMillis)
		if task.Streaming && task.Started {
			firstOutput = append(firstOutput, task.TTFTMillis)
		}
		if task.Started {
			result.Counts.Started++
			request = append(request, task.QueueMillis+task.DurationMillis)
			start := task.StartMillis - SyntheticClockEpochMillis
			intervals = append(intervals, [2]int64{start, start + task.DurationMillis})
		}
		if task.QueueMillis > 0 {
			result.Counts.Queued++
		}
		switch task.Outcome {
		case "completed":
			result.Counts.Completed++
		case "failed-after-output":
			result.Counts.Failed++
		default:
			result.Counts.Cancelled++
		}
		result.Retries += task.RetryCount
		result.Fallbacks += task.FallbackCount
		if task.Streaming {
			result.StreamingCount++
		}
		if task.ToolRequest {
			result.ToolRequestCount++
		}
		if task.ParallelTools {
			result.ParallelToolCount++
		}
		if task.Continuation {
			result.ContinuationCount++
		} else {
			result.IndependentCount++
			slotCounts[task.ConnectionSlot]++
		}
	}
	result.SelectionLatency = percentiles(selection)
	result.QueueLatency = percentiles(queue)
	result.FirstOutputLatency = percentiles(firstOutput)
	result.RequestLatency = percentiles(request)

	expected := result.IndependentCount / 4
	maxDeviation := 0
	for index := 1; index <= 4; index++ {
		slot := fmt.Sprintf("slot-%d", index)
		deviation := absInt(slotCounts[slot] - expected)
		maxDeviation = maxInt(maxDeviation, deviation)
		result.SlotDistribution = append(result.SlotDistribution, SlotDistribution{
			Slot: slot, IndependentRequests: slotCounts[slot], ExpectedRequests: expected, AbsoluteDeviation: deviation,
		})
	}
	if expected > 0 {
		result.FairnessDeviationPct = maxDeviation * 100 / expected
	}
	peakActive := peakConcurrent(intervals)
	result.Resources = ResourceModel{
		Source:          "deterministic-model-not-host-sampled",
		PeakActive:      peakActive,
		CPUMilliPeak:    150 + peakActive*110,
		MemoryBytesPeak: int64(48+peakActive*8) * 1024 * 1024,
		SocketPeak:      2 + peakActive*2,
	}
	return result
}

func (r DevelopmentRunResult) Validate() error {
	if r.SchemaVersion != EvidenceSchemaVersion || r.ProfileVersion != DevelopmentProfileVersion || !safeID(r.RunID, 128) {
		return fmt.Errorf("invalid synthetic result identity")
	}
	if !r.SyntheticOnly || r.LiveEndpointUsed || r.CapacityTierEnabled || r.AcceptanceClaim {
		return fmt.Errorf("development result crossed the synthetic-only acceptance boundary")
	}
	if r.Counts.Offered != 20 || r.Counts.Admitted != 20 || r.Counts.Rejected != 0 || len(r.Tasks) != 20 || r.ConcurrencyLimit != 4 {
		return fmt.Errorf("development profile must remain a 20-task bounded simulation")
	}
	if r.Counts.Completed+r.Counts.Failed+r.Counts.Cancelled != r.Counts.Admitted || r.Counts.Started+r.Counts.Cancelled-r.cancelledStarted() != r.Counts.Admitted {
		return fmt.Errorf("synthetic lifecycle counters do not reconcile")
	}
	if r.ProtocolCounts[ProtocolAnthropicMessages] != 6 || r.ProtocolCounts[ProtocolOpenAIResponses] != 6 ||
		r.ProtocolCounts[ProtocolOpenAIChat] != 6 || r.ProtocolCounts[ProtocolAntigravityDirect] != 2 {
		return fmt.Errorf("protocol distribution does not match the approved 30/30/30/10 shape")
	}
	if r.StreamingCount != 14 || r.ToolRequestCount != 8 || r.ParallelToolCount != 3 ||
		r.IndependentCount != 16 || r.ContinuationCount != 4 {
		return fmt.Errorf("stream, tool, and continuation counts do not match the profile")
	}
	if r.PayloadCounts["small"] != 8 || r.PayloadCounts["medium"] != 8 || r.PayloadCounts["large-context-relative"] != 4 {
		return fmt.Errorf("payload distribution does not match the approved 40/40/20 shape")
	}
	if len(r.SlotDistribution) != 4 || r.FairnessDeviationPct != 0 {
		return fmt.Errorf("development independent-request distribution is not deterministic and even")
	}
	if !slices.Equal(r.FailureCasesCovered, sortedFailureIDs()) || len(r.Blockers) == 0 {
		return fmt.Errorf("failure coverage or acceptance blockers are incomplete")
	}
	if r.Resources.Source != "deterministic-model-not-host-sampled" || r.Resources.PeakActive > r.ConcurrencyLimit {
		return fmt.Errorf("resource measurements must remain explicitly modeled and bounded")
	}
	return nil
}

func MarshalDevelopmentRunResult(result DevelopmentRunResult) ([]byte, error) {
	if err := result.Validate(); err != nil {
		return nil, err
	}
	return json.MarshalIndent(result, "", "  ")
}

func DevelopmentRunResultDigest(result DevelopmentRunResult) (string, error) {
	data, err := MarshalDevelopmentRunResult(result)
	if err != nil {
		return "", err
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:]), nil
}

func (r DevelopmentRunResult) cancelledStarted() int {
	count := 0
	for _, task := range r.Tasks {
		if task.Started && task.Outcome == "cancelled-active" {
			count++
		}
	}
	return count
}

func protocolForTask(number int) ProtocolFamily {
	switch {
	case number <= 6:
		return ProtocolAnthropicMessages
	case number <= 12:
		return ProtocolOpenAIResponses
	case number <= 18:
		return ProtocolOpenAIChat
	default:
		return ProtocolAntigravityDirect
	}
}

func payloadForTask(number int) string {
	switch (number - 1) % 5 {
	case 0, 1:
		return "small"
	case 2, 3:
		return "medium"
	default:
		return "large-context-relative"
	}
}

func earliestWorker(available []int64) int {
	index := 0
	for candidate := 1; candidate < len(available); candidate++ {
		if available[candidate] < available[index] {
			index = candidate
		}
	}
	return index
}

func percentiles(values []int64) Percentiles {
	if len(values) == 0 {
		return Percentiles{}
	}
	ordered := slices.Clone(values)
	slices.Sort(ordered)
	return Percentiles{
		P50: ordered[nearestRank(len(ordered), 50)],
		P95: ordered[nearestRank(len(ordered), 95)],
		P99: ordered[nearestRank(len(ordered), 99)],
	}
}

func nearestRank(length, percentile int) int {
	rank := (length*percentile + 99) / 100
	if rank < 1 {
		rank = 1
	}
	return rank - 1
}

func peakConcurrent(intervals [][2]int64) int {
	type event struct {
		at    int64
		delta int
	}
	events := make([]event, 0, len(intervals)*2)
	for _, interval := range intervals {
		events = append(events, event{interval[0], 1}, event{interval[1], -1})
	}
	slices.SortFunc(events, func(a, b event) int {
		if a.at < b.at {
			return -1
		}
		if a.at > b.at {
			return 1
		}
		return a.delta - b.delta
	})
	active, peak := 0, 0
	for _, event := range events {
		active += event.delta
		peak = maxInt(peak, active)
	}
	return peak
}

func failureIDs(tasks []SyntheticTask) []string {
	seen := map[string]bool{}
	for _, task := range tasks {
		if task.FailureCaseID != "" {
			seen[task.FailureCaseID] = true
		}
	}
	result := make([]string, 0, len(seen))
	for id := range seen {
		result = append(result, id)
	}
	slices.Sort(result)
	return result
}

func sortedFailureIDs() []string {
	spec := DefaultAcceptanceHarnessSpec()
	result := make([]string, 0, len(spec.Failures))
	for _, failure := range spec.Failures {
		result = append(result, failure.ID)
	}
	slices.Sort(result)
	return result
}

func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
