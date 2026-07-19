package deploy

// CapacityHarnessOBSConfig prepares the 20-task capacity/failure harness
// so it runs WITH observability instrumentation enabled and measures span overhead (R30).
type CapacityHarnessOBSConfig struct {
	EnableObservabilityInstrumentation bool
	MeasureSpanOverheadR30             bool
}

func GetCapacityHarnessOBSConfig() CapacityHarnessOBSConfig {
	return CapacityHarnessOBSConfig{
		EnableObservabilityInstrumentation: true,
		MeasureSpanOverheadR30:             true,
	}
}
