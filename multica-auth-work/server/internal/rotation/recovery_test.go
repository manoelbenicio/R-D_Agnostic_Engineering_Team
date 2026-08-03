package rotation

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestRotationRecoveryBackoffIsBounded(t *testing.T) {
	want := []time.Duration{
		0, 5 * time.Second, 30 * time.Second, 2 * time.Minute,
		10 * time.Minute, 30 * time.Minute,
	}
	if maxNativeRetirementAttempts != len(want) {
		t.Fatalf("max attempts = %d, want %d", maxNativeRetirementAttempts, len(want))
	}
	for i := range want {
		if nativeRetirementBackoff[i] != want[i] {
			t.Fatalf("backoff[%d] = %s, want %s", i, nativeRetirementBackoff[i], want[i])
		}
	}
}

func TestRotationRecoveryContractsArePathlessAndValueFree(t *testing.T) {
	for _, value := range []any{
		NativeHomeIdentityV1{}, NativeRetirementRequestV1{},
		NativeRetirementResultV1{},
	} {
		typ := reflect.TypeOf(value)
		for i := 0; i < typ.NumField(); i++ {
			name := strings.ToLower(typ.Field(i).Name)
			for _, forbidden := range []string{
				"path", "homedir", "config", "account", "credential",
				"token", "secret", "providerresponse",
			} {
				if strings.Contains(name, forbidden) {
					t.Fatalf("%s contains forbidden field %s", typ.Name(), typ.Field(i).Name)
				}
			}
		}
	}
}
