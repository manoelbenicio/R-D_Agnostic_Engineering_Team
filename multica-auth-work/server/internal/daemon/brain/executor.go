package brain

import (
	"context"
	"fmt"
)

// LifecycleRequest contains only the neutral task and selected executable.
type LifecycleRequest struct {
	Task    Task
	Runtime RuntimeDescriptor
}

// PreservedLifecycle is the G2 strangler boundary around existing behavior.
// Its eventual G3 adapter retains workspace/repository/worktree preparation,
// context and skills, recovery, cancellation, watchdogs, stream batching,
// process cleanup, and terminal-result semantics without moving that logic
// into the neutral package.
type PreservedLifecycle interface {
	ExecuteLifecycle(context.Context, LifecycleRequest) (TaskResult, error)
}

// CoordinatorTaskExecutor is the neutral task boundary used by Coordinator.
// It accepts the complete neutral task, including opaque lifecycle references.
// The frozen G1 TaskExecutor remains available to legacy callers during
// migration.
type CoordinatorTaskExecutor interface {
	ExecuteTask(context.Context, Task) (TaskResult, error)
}

// LifecycleTaskExecutor resolves by CLIKind and delegates the complete
// lifecycle. It never selects provider accounts or prepares credentials.
type LifecycleTaskExecutor struct {
	Registry  RuntimeRegistry
	Lifecycle PreservedLifecycle
}

func NewLifecycleTaskExecutor(registry RuntimeRegistry, lifecycle PreservedLifecycle) (*LifecycleTaskExecutor, error) {
	if registry == nil {
		return nil, fmt.Errorf("runtime registry is required")
	}
	if lifecycle == nil {
		return nil, fmt.Errorf("preserved lifecycle is required")
	}
	return &LifecycleTaskExecutor{Registry: registry, Lifecycle: lifecycle}, nil
}

func (e *LifecycleTaskExecutor) ExecuteTask(ctx context.Context, task Task) (TaskResult, error) {
	if e == nil || e.Registry == nil || e.Lifecycle == nil {
		return TaskResult{}, fmt.Errorf("lifecycle task executor is not configured")
	}
	if err := task.Validate(); err != nil {
		return TaskResult{}, err
	}
	descriptor, err := e.Registry.ResolveRuntime(ctx, task.Request.CLIKind)
	if err != nil {
		return TaskResult{}, err
	}
	return e.Lifecycle.ExecuteLifecycle(ctx, LifecycleRequest{Task: task, Runtime: descriptor})
}
