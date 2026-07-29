package brain

import "errors"

type RecoveryState string

const (
	RecoveryNormal   RecoveryState = "NORMAL"
	RecoveryDegraded RecoveryState = "DEGRADED"
)

type RecoveryRouterOwner string

const (
	RecoveryRouterNone      RecoveryRouterOwner = "none"
	RecoveryRouterOmniRoute RecoveryRouterOwner = "omniroute"
)

type RecoveryTransition string

const (
	RecoveryGatewayUnavailable RecoveryTransition = "GATEWAY_UNAVAILABLE"
	RecoveryGatewayRestored    RecoveryTransition = "GATEWAY_RESTORED"
)

type RecoveryGates struct {
	SessionBoundary bool
	GatewayReady    bool
}

type RecoveryMode struct {
	state       RecoveryState
	routerOwner RecoveryRouterOwner
}

func NewRecoveryMode() RecoveryMode {
	return RecoveryMode{state: RecoveryNormal, routerOwner: RecoveryRouterOmniRoute}
}

func (m RecoveryMode) State() RecoveryState { return m.state }

func (m RecoveryMode) RouterOwner() RecoveryRouterOwner {
	if m.validate() != nil {
		return RecoveryRouterNone
	}
	return m.routerOwner
}

// Transition is fail-closed and OmniRoute-only. An outage removes admission
// authority; only a ready OmniRoute can restore it. There is no alternate
// router, operator bypass, or automatic fallback state.
func (m RecoveryMode) Transition(event RecoveryTransition, gates RecoveryGates) (RecoveryMode, error) {
	if err := m.validate(); err != nil {
		return m, err
	}
	if !gates.SessionBoundary {
		return m, errors.New("recovery transition requires a session boundary")
	}
	switch {
	case m.state == RecoveryNormal && event == RecoveryGatewayUnavailable:
		return RecoveryMode{state: RecoveryDegraded, routerOwner: RecoveryRouterNone}, nil
	case m.state == RecoveryDegraded && event == RecoveryGatewayRestored:
		if !gates.GatewayReady {
			return m, errors.New("gateway restore requires a ready OmniRoute")
		}
		return NewRecoveryMode(), nil
	default:
		return m, errors.New("recovery transition is not allowed from the current state")
	}
}

func (m RecoveryMode) validate() error {
	switch m.state {
	case RecoveryNormal:
		if m.routerOwner != RecoveryRouterOmniRoute {
			return errors.New("NORMAL requires OmniRoute as the single router owner")
		}
	case RecoveryDegraded:
		if m.routerOwner != RecoveryRouterNone {
			return errors.New("DEGRADED must fail closed without a router owner")
		}
	default:
		return errors.New("unknown recovery state")
	}
	return nil
}
