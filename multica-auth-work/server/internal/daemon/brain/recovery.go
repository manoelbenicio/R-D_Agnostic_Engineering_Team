package brain

import "errors"

// RecoveryState is the platform-wide routing state. It is deliberately not a
// per-task choice: a state applies only after an accepted session-boundary
// transition.
type RecoveryState string

const (
	RecoveryNormal   RecoveryState = "NORMAL"
	RecoveryDegraded RecoveryState = "DEGRADED"
	RecoveryActive   RecoveryState = "RECOVERY"
)

// RecoveryRouterOwner identifies the only router allowed to own new sessions
// in a recovery state. DEGRADED intentionally has no owner and fails closed.
type RecoveryRouterOwner string

const (
	RecoveryRouterNone      RecoveryRouterOwner = "none"
	RecoveryRouterOmniRoute RecoveryRouterOwner = "omniroute"
	RecoveryRouterRustL2    RecoveryRouterOwner = "rust_l2"
)

// RecoveryTransition is an explicit platform-level action. There is no
// automatic NORMAL-to-RECOVERY transition.
type RecoveryTransition string

const (
	RecoveryGatewayUnavailable RecoveryTransition = "GATEWAY_UNAVAILABLE"
	RecoveryGatewayRestored    RecoveryTransition = "GATEWAY_RESTORED"
	RecoveryEnable             RecoveryTransition = "ENABLE_RECOVERY"
	RecoveryRestore            RecoveryTransition = "RESTORE"
)

// RecoveryGates are metadata-only safety assertions supplied by the future
// platform operator path. They never carry credentials or request content.
type RecoveryGates struct {
	OperatorAuthorized bool
	SessionBoundary    bool
	OmniRouteQuiesced  bool
	ProdexDrained      bool
	GatewayReady       bool
}

// RecoveryMode is immutable. Call Transition to obtain the next accepted
// state. NewRecoveryMode always starts NORMAL with Prodex cold and OFF.
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

func (m RecoveryMode) ProdexEnabled() bool {
	return m.state == RecoveryActive && m.RouterOwner() == RecoveryRouterRustL2
}

// Transition enforces AB-REQ-41. Every transition occurs at a session
// boundary; recovery entry and restore are operator-gated. A gateway outage
// enters DEGRADED with no router owner and never promotes Prodex automatically.
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
			return m, errors.New("gateway restore requires a ready gateway")
		}
		return NewRecoveryMode(), nil
	case m.state == RecoveryDegraded && event == RecoveryEnable:
		if !gates.OperatorAuthorized {
			return m, errors.New("recovery entry requires operator authorization")
		}
		if !gates.OmniRouteQuiesced {
			return m, errors.New("recovery entry requires OmniRoute to be quiesced")
		}
		return RecoveryMode{state: RecoveryActive, routerOwner: RecoveryRouterRustL2}, nil
	case m.state == RecoveryActive && event == RecoveryRestore:
		if !gates.OperatorAuthorized {
			return m, errors.New("recovery restore requires operator authorization")
		}
		if !gates.ProdexDrained {
			return m, errors.New("recovery restore requires Prodex to be drained")
		}
		if !gates.GatewayReady {
			return m, errors.New("recovery restore requires a ready gateway")
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
	case RecoveryActive:
		if m.routerOwner != RecoveryRouterRustL2 {
			return errors.New("RECOVERY requires rust_l2 as the single router owner")
		}
	default:
		return errors.New("unknown recovery state")
	}
	return nil
}
