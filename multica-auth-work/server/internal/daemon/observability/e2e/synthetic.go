package e2e

import "fmt"

// SyntheticTraceSpans returns a complete, valid, metadata-only set of the seven
// emitting-hop spans for one synthetic task, correctly cross-joined per the
// correlation contract. It is used by OBS-9 acceptance and by the capacity
// harness (W4) to prove one continuous trace per synthetic task. All values are
// synthetic; no secrets or content are involved.
func SyntheticTraceSpans(taskID string) []Span {
	c := Correlation{
		RequestID:     "req-" + taskID,
		QueueMsgID:    "qmsg-" + taskID,
		TaskID:        taskID,
		SessionID:     "sess-" + taskID,
		LaunchID:      "launch-" + taskID,
		ProcID:        "proc-" + taskID,
		OmniRequestID: "omni-" + taskID,
		ResultID:      "result-" + taskID,
		DeliveryID:    "delivery-" + taskID,
	}
	mk := func(hop HopKind, corr Correlation, outcome string) *Span {
		return NewSpan(hop, corr).WithOutcome(outcome, "ok").Finish()
	}
	return []Span{
		*mk(HopIngress, Correlation{RequestID: c.RequestID, TaskID: c.TaskID}, "accepted").
			WithLabel("method", "POST").WithLabel("route_template", "/v1/tasks").
			WithCounter("latency_ms", 8).WithHTTPStatus(202),
		*mk(HopQueue, Correlation{QueueMsgID: c.QueueMsgID, TaskID: c.TaskID}, "dequeued"),
		*mk(HopAdmission, Correlation{TaskID: c.TaskID, SessionID: c.SessionID, LaunchID: c.LaunchID}, "admitted").
			WithLabel("admission_decision", "admit").WithLabel("readiness_result", "ready"),
		*mk(HopCLI, Correlation{LaunchID: c.LaunchID, ProcID: c.ProcID}, "exited").
			WithArgvShape([]string{"subcommand", "flag=<redacted>"}).WithLabel("exit_code_class", "code_0"),
		*mk(HopRoute, Correlation{RequestID: c.RequestID, OmniRequestID: c.OmniRequestID}, "ok").
			WithLabel("route_model", "model-a").WithLabel("protocol", "openai-responses"),
		*mk(HopPersist, Correlation{TaskID: c.TaskID, ResultID: c.ResultID}, "persisted"),
		*mk(HopDelivery, Correlation{SessionID: c.SessionID, DeliveryID: c.DeliveryID}, "delivered"),
	}
}

// EmitSyntheticTask records a full synthetic trace for taskID via the recorder,
// returning the first emit error (if any) for fail-closed classification.
func EmitSyntheticTask(rec *Recorder, taskID string) error {
	if rec == nil {
		return fmt.Errorf("nil recorder")
	}
	for _, s := range SyntheticTraceSpans(taskID) {
		sp := s
		if err := rec.Emit(&sp); err != nil {
			return err
		}
	}
	return nil
}
