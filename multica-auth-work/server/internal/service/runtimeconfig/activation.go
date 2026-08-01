package runtimeconfig

// Activation and rollback semantics for a resolved runtime configuration.
//
// The activation state machine is deliberately tiny and pure: it takes the
// current state plus a candidate, and returns the next state and the plan a
// caller must apply. It performs no I/O, holds no lock, and mutates nothing
// the caller passed in, so it is safe to call concurrently on independent
// state and cheap to test exhaustively.
//
// Frozen rules:
//
//  1. Fail closed on both sides. A candidate is rejected unless it passes the
//     same integrity gate as a resolved configuration, and a tampered current
//     state is rejected too — otherwise a corrupted active generation could be
//     used as the rollback target.
//  2. Activation is idempotent by digest. Re-activating the identical digest is
//     a no-op: the generation does not advance and the previous generation is
//     preserved, so a retried or duplicated activation cannot destroy the only
//     rollback target.
//  3. Exactly one previous generation is retained. That is what an immediate
//     revert needs; deeper history belongs to durable storage, not to this
//     in-memory contract.
//  4. Rollback swaps. After a rollback the configuration that was rolled back
//     from becomes the new rollback target, so a mistaken rollback can itself
//     be undone. Generation still advances, because a rollback is an applied
//     transition and the counter is an audit trail, not a config version.
//  5. A restart-requiring transition is reported, never performed. Activate
//     returns the plan; deciding when to restart is the caller's authority.

// InitialActivation validates config and returns the first activation
// generation. The plan lists every field that must be applied to reach the
// configuration from nothing, classified by reload class, with no rollback
// target because no earlier generation exists.
func InitialActivation(config Effective) (Activation, ChangePlan, error) {
	if err := ValidateEffective(config); err != nil {
		return Activation{}, ChangePlan{}, err
	}
	plan := initialPlan(config)
	active := cloneEffective(config)
	return Activation{
		Generation: 1,
		Active:     active,
		LastPlan:   cloneChangePlan(plan),
	}, plan, nil
}

// Activate transitions current to next and returns the plan the caller must
// apply. A nil current is the first activation and is equivalent to
// InitialActivation. Re-activating the digest already active is a no-op.
func Activate(current *Activation, next Effective) (Activation, ChangePlan, error) {
	if current == nil {
		return InitialActivation(next)
	}
	if err := ValidateEffective(current.Active); err != nil {
		return Activation{}, ChangePlan{}, err
	}
	if err := ValidateEffective(next); err != nil {
		return Activation{}, ChangePlan{}, err
	}
	if current.Active.Digest == next.Digest {
		return cloneActivation(*current), ChangePlan{}, nil
	}

	plan, err := ClassifyChanges(current.Active, next)
	if err != nil {
		return Activation{}, ChangePlan{}, err
	}
	previous := cloneEffective(current.Active)
	return Activation{
		Generation: current.Generation + 1,
		Active:     cloneEffective(next),
		Previous:   &previous,
		LastPlan:   cloneChangePlan(plan),
	}, plan, nil
}

// Rollback restores the retained previous generation and returns the plan to
// get there from the currently active configuration. It fails closed when
// there is no rollback target or when either side is tampered.
func Rollback(current Activation) (Activation, ChangePlan, error) {
	if err := ValidateEffective(current.Active); err != nil {
		return Activation{}, ChangePlan{}, err
	}
	if current.Previous == nil {
		return Activation{}, ChangePlan{}, ValidationErrors{{Code: ErrNoRollbackTarget, Field: Field("previous")}}
	}
	if err := ValidateEffective(*current.Previous); err != nil {
		return Activation{}, ChangePlan{}, err
	}

	plan, err := ClassifyChanges(current.Active, *current.Previous)
	if err != nil {
		return Activation{}, ChangePlan{}, err
	}
	rolledBackFrom := cloneEffective(current.Active)
	return Activation{
		Generation: current.Generation + 1,
		Active:     cloneEffective(*current.Previous),
		Previous:   &rolledBackFrom,
		LastPlan:   cloneChangePlan(plan),
	}, plan, nil
}

// initialPlan classifies every set field of a first activation. Delegability is
// not reported as changed because there is no prior policy to differ from.
func initialPlan(config Effective) ChangePlan {
	var plan ChangePlan
	for _, field := range allFields {
		if _, ok := fieldValue(config.Values, field); !ok {
			continue
		}
		if class, _ := ClassifyField(field); class == ReloadRestartRequired {
			plan.RestartRequired = append(plan.RestartRequired, field)
		} else {
			plan.Hot = append(plan.Hot, field)
		}
	}
	return plan
}

func cloneActivation(activation Activation) Activation {
	clone := Activation{
		Generation: activation.Generation,
		Active:     cloneEffective(activation.Active),
		LastPlan:   cloneChangePlan(activation.LastPlan),
	}
	if activation.Previous != nil {
		previous := cloneEffective(*activation.Previous)
		clone.Previous = &previous
	}
	return clone
}

func cloneChangePlan(plan ChangePlan) ChangePlan {
	clone := ChangePlan{DelegabilityChanged: plan.DelegabilityChanged}
	if plan.Hot != nil {
		clone.Hot = make([]Field, len(plan.Hot))
		copy(clone.Hot, plan.Hot)
	}
	if plan.RestartRequired != nil {
		clone.RestartRequired = make([]Field, len(plan.RestartRequired))
		copy(clone.RestartRequired, plan.RestartRequired)
	}
	return clone
}
