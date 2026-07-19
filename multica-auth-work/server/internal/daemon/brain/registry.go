package brain

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrRuntimeNotFound = errors.New("runtime not found")

// StaticRuntimeRegistry is an immutable CLIKind-to-executable registry. It
// deliberately has no provider, account, credential-home, or fallback key.
type StaticRuntimeRegistry struct {
	runtimes map[CLIKind]RuntimeDescriptor
}

func NewStaticRuntimeRegistry(descriptors ...RuntimeDescriptor) (*StaticRuntimeRegistry, error) {
	runtimes := make(map[CLIKind]RuntimeDescriptor, len(descriptors))
	for _, descriptor := range descriptors {
		kind, err := ParseCLIKind(string(descriptor.CLIKind))
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(descriptor.Path) == "" {
			return nil, fmt.Errorf("runtime path is required for CLI kind %q", kind)
		}
		if _, exists := runtimes[kind]; exists {
			return nil, fmt.Errorf("duplicate runtime for CLI kind %q", kind)
		}
		descriptor.CLIKind = kind
		runtimes[kind] = descriptor
	}
	return &StaticRuntimeRegistry{runtimes: runtimes}, nil
}

func (r *StaticRuntimeRegistry) ResolveRuntime(ctx context.Context, kind CLIKind) (RuntimeDescriptor, error) {
	if err := ctx.Err(); err != nil {
		return RuntimeDescriptor{}, err
	}
	if r == nil {
		return RuntimeDescriptor{}, fmt.Errorf("%w: registry is nil", ErrRuntimeNotFound)
	}
	parsed, err := ParseCLIKind(string(kind))
	if err != nil {
		return RuntimeDescriptor{}, err
	}
	descriptor, ok := r.runtimes[parsed]
	if !ok {
		return RuntimeDescriptor{}, fmt.Errorf("%w for CLI kind %q", ErrRuntimeNotFound, parsed)
	}
	return descriptor, nil
}
