package brain

import (
	"context"
	"errors"
	"fmt"
)

// Coordinator owns neutral admission and lifecycle delegation only. It has no
// provider-account, credential-home, retry-router, or gateway implementation.
type Coordinator struct {
	Admission AdmissionController
	Executor  CoordinatorTaskExecutor
	Results   ResultSink
}

func NewCoordinator(admission AdmissionController, executor CoordinatorTaskExecutor, results ResultSink) (*Coordinator, error) {
	if admission == nil {
		return nil, fmt.Errorf("admission controller is required")
	}
	if executor == nil {
		return nil, fmt.Errorf("task executor is required")
	}
	if results == nil {
		return nil, fmt.Errorf("result sink is required")
	}
	return &Coordinator{Admission: admission, Executor: executor, Results: results}, nil
}

func (c *Coordinator) Run(ctx context.Context, task Task) (TaskResult, error) {
	if c == nil || c.Admission == nil || c.Executor == nil || c.Results == nil {
		return TaskResult{}, fmt.Errorf("coordinator is not configured")
	}
	decision, err := c.Admission.Admit(ctx, task)
	if err != nil {
		return TaskResult{}, err
	}
	if !decision.Admitted() {
		result := TaskResult{
			Correlation: task.Request.Correlation,
			Status:      decision.TaskStatus,
			Retryable:   decision.Retryable,
			ErrorClass:  decision.ErrorClass,
		}
		return c.publish(ctx, result, nil)
	}

	result, executeErr := c.Executor.ExecuteTask(ctx, task)
	if result.Correlation.TaskID == "" {
		result.Correlation = task.Request.Correlation
	}
	if result.Status == "" {
		if errors.Is(executeErr, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			result.Status = TaskStatusCancelled
			result.ErrorClass = "cancelled"
		} else {
			result.Status = TaskStatusFailed
			result.ErrorClass = "execution_failed"
		}
	}
	return c.publish(ctx, result, executeErr)
}

func (c *Coordinator) publish(ctx context.Context, result TaskResult, prior error) (TaskResult, error) {
	// Terminal recording must survive task cancellation while preserving
	// correlation values carried by the task context.
	publishErr := c.Results.PublishResult(context.WithoutCancel(ctx), result)
	return result, errors.Join(prior, publishErr)
}
