package e2e

import "sort"

// Trace is the assembled end-to-end view of a single task: one span per emitting
// hop, joined by the documented correlation relationships.
type Trace struct {
	TaskID      string                `json:"task_id"`
	Hops        map[HopKind]Span      `json:"-"`
	Present     []HopKind             `json:"present"`
	Missing     []HopKind             `json:"missing"`
	Continuous  bool                  `json:"continuous"`
	Anchor      Correlation           `json:"anchor"`
}

// OrphanSpan is a span that could not be joined into any complete trace.
type OrphanSpan struct {
	Hop         HopKind     `json:"hop"`
	Correlation Correlation `json:"correlation"`
	Reason      string      `json:"reason"`
}

// AssemblyReport is the OBS-9 result: one Trace per synthetic task plus any
// orphan/gap findings. AllContinuous is true only when every discovered task
// has all seven emitting hops joined with zero orphans.
type AssemblyReport struct {
	Traces        []Trace      `json:"traces"`
	Orphans       []OrphanSpan `json:"orphans"`
	AllContinuous bool         `json:"all_continuous"`
}

// Assemble joins spans into per-task traces following the documented join
// relationships (docs/observability/e2e-metadata-span.md §2):
//
//	ingress   : task_id (also yields request_id)
//	queue     : task_id
//	admission : task_id (also yields session_id, launch_id)
//	cli       : launch_id  -> resolved to task via admission
//	route     : request_id -> resolved to task via ingress
//	persist   : task_id
//	delivery  : session_id -> resolved to task via admission
//
// A trace is continuous when all seven emitting hops are present for the task.
// Spans that reference identifiers belonging to no known task are reported as
// orphans. Only structurally valid spans are considered; invalid spans are
// reported as orphans (fail-closed).
func Assemble(spans []Span) AssemblyReport {
	report := AssemblyReport{}

	// Partition valid vs. invalid; index by hop.
	byHop := map[HopKind][]Span{}
	for _, s := range spans {
		if err := s.Validate(); err != nil {
			report.Orphans = append(report.Orphans, OrphanSpan{
				Hop: s.Hop, Correlation: s.Correlation, Reason: "invalid span: " + err.Error(),
			})
			continue
		}
		byHop[s.Hop] = append(byHop[s.Hop], s)
	}

	// Build resolver indices from the task-anchored hops.
	requestToTask := map[string]string{} // request_id -> task_id (from ingress)
	launchToTask := map[string]string{}  // launch_id  -> task_id (from admission)
	sessionToTask := map[string]string{} // session_id -> task_id (from admission)
	taskAnchor := map[string]Correlation{}
	taskSet := map[string]struct{}{}

	registerTask := func(id string) {
		if id != "" {
			taskSet[id] = struct{}{}
		}
	}
	for _, s := range byHop[HopIngress] {
		registerTask(s.Correlation.TaskID)
		requestToTask[s.Correlation.RequestID] = s.Correlation.TaskID
	}
	for _, s := range byHop[HopQueue] {
		registerTask(s.Correlation.TaskID)
	}
	for _, s := range byHop[HopAdmission] {
		registerTask(s.Correlation.TaskID)
		launchToTask[s.Correlation.LaunchID] = s.Correlation.TaskID
		sessionToTask[s.Correlation.SessionID] = s.Correlation.TaskID
		taskAnchor[s.Correlation.TaskID] = s.Correlation
	}
	for _, s := range byHop[HopPersist] {
		registerTask(s.Correlation.TaskID)
	}

	// Assign each hop's spans to a task via the documented join key.
	traceHops := map[string]map[HopKind]Span{}
	ensure := func(task string) map[HopKind]Span {
		if traceHops[task] == nil {
			traceHops[task] = map[HopKind]Span{}
		}
		return traceHops[task]
	}
	assignDirect := func(hop HopKind) {
		for _, s := range byHop[hop] {
			task := s.Correlation.TaskID
			if _, ok := taskSet[task]; !ok {
				report.Orphans = append(report.Orphans, OrphanSpan{hop, s.Correlation, "task_id not anchored"})
				continue
			}
			ensure(task)[hop] = s
		}
	}
	assignVia := func(hop HopKind, resolve map[string]string, key func(Correlation) string, reason string) {
		for _, s := range byHop[hop] {
			task, ok := resolve[key(s.Correlation)]
			if !ok || task == "" {
				report.Orphans = append(report.Orphans, OrphanSpan{hop, s.Correlation, reason})
				continue
			}
			ensure(task)[hop] = s
		}
	}

	assignDirect(HopIngress)
	assignDirect(HopQueue)
	assignDirect(HopAdmission)
	assignDirect(HopPersist)
	assignVia(HopCLI, launchToTask, func(c Correlation) string { return c.LaunchID }, "launch_id resolves to no admitted task")
	assignVia(HopRoute, requestToTask, func(c Correlation) string { return c.RequestID }, "request_id resolves to no ingress task")
	assignVia(HopDelivery, sessionToTask, func(c Correlation) string { return c.SessionID }, "session_id resolves to no admitted task")

	// Materialize traces deterministically.
	tasks := make([]string, 0, len(taskSet))
	for t := range taskSet {
		tasks = append(tasks, t)
	}
	sort.Strings(tasks)

	allContinuous := len(tasks) > 0
	for _, task := range tasks {
		hops := traceHops[task]
		tr := Trace{TaskID: task, Hops: hops, Anchor: taskAnchor[task]}
		for _, h := range EmittingHops() {
			if _, ok := hops[h]; ok {
				tr.Present = append(tr.Present, h)
			} else {
				tr.Missing = append(tr.Missing, h)
			}
		}
		tr.Continuous = len(tr.Missing) == 0
		if !tr.Continuous {
			allContinuous = false
		}
		report.Traces = append(report.Traces, tr)
	}
	if len(report.Orphans) > 0 {
		allContinuous = false
	}
	report.AllContinuous = allContinuous
	return report
}

// AssembleFromSink is a convenience wrapper over a MemorySink.
func AssembleFromSink(sink *MemorySink) AssemblyReport {
	if sink == nil {
		return AssemblyReport{}
	}
	return Assemble(sink.Spans())
}
