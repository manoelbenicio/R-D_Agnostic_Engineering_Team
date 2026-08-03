package rotation

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	dbgen "github.com/multica-ai/multica/server/pkg/db/generated"
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
		NativeRetirementResultV1{}, NativeRetirementTransitionV1{},
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

func TestRotationRecoveryTransitionBindsImmutableIdentity(t *testing.T) {
	attempt, transition := nativeRetirementTransitionFixture()
	if !retirementTransitionMatches(attempt, transition) {
		t.Fatal("exact immutable transition identity did not match")
	}
	for name, mutate := range map[string]func(*NativeRetirementTransitionV1){
		"attempt": func(v *NativeRetirementTransitionV1) { v.AttemptID = testUUID(9) },
		"request": func(v *NativeRetirementTransitionV1) { v.RetirementRequestID = testUUID(9) },
		"daemon":  func(v *NativeRetirementTransitionV1) { v.DaemonID = "other-daemon" },
		"boot":    func(v *NativeRetirementTransitionV1) { v.DaemonBootID = testUUID(9) },
		"session": func(v *NativeRetirementTransitionV1) { v.RuntimeSessionID = testUUID(9) },
		"binding": func(v *NativeRetirementTransitionV1) { v.RuntimeBindingID = testUUID(9) },
		"generation": func(v *NativeRetirementTransitionV1) {
			v.BindingGeneration++
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := transition
			mutate(&changed)
			if retirementTransitionMatches(attempt, changed) {
				t.Fatal("cross-linked transition identity matched")
			}
		})
	}
}

func TestRotationRecoveryTerminalPredecessorAndReplayRules(t *testing.T) {
	attempt, transition := nativeRetirementTransitionFixture()
	attempt.ChannelBindingDigest = pgtype.Text{
		String: transition.ChannelBindingDigest, Valid: true,
	}
	attempt.RequestBodyDigest = pgtype.Text{
		String: transition.RequestBodyDigest, Valid: true,
	}
	result := NativeRetirementResultV1{
		AttemptID:           transition.AttemptID,
		RetirementRequestID: transition.RetirementRequestID,
		State:               "succeeded", ResultCode: "retired",
		ChannelBindingDigest: transition.ChannelBindingDigest,
		RequestBodyDigest:    transition.RequestBodyDigest,
	}
	attempt.State = "executing"
	if err := validateNativeRetirementResult(attempt, result); err != nil {
		t.Fatalf("executing -> succeeded rejected: %v", err)
	}
	attempt.State = "succeeded"
	if err := validateNativeRetirementResult(attempt, result); !errors.Is(err, ErrNativeCASConflict) {
		t.Fatalf("terminal replay error = %v, want CAS conflict", err)
	}
	attempt.State = "accepted"
	result.State = "quarantined"
	if err := validateNativeRetirementResult(attempt, result); err != nil {
		t.Fatalf("explicit accepted quarantine rejected: %v", err)
	}
	attempt.State = "scheduled"
	if err := validateNativeRetirementResult(attempt, result); !errors.Is(err, ErrNativeCASConflict) {
		t.Fatalf("unbound scheduled quarantine error = %v, want CAS conflict", err)
	}
	result.State = "succeeded"
	if err := validateNativeRetirementResult(attempt, result); !errors.Is(err, ErrNativeCASConflict) {
		t.Fatalf("scheduled success error = %v, want CAS conflict", err)
	}
}

func nativeRetirementTransitionFixture() (
	dbgen.NativeRotationRetirementAttempt,
	NativeRetirementTransitionV1,
) {
	attempt := dbgen.NativeRotationRetirementAttempt{
		ID:                  mustNativeUUID(testUUID(1)),
		RetirementRequestID: mustNativeUUID(testUUID(2)),
		DaemonID:            "daemon-a", DaemonBootID: mustNativeUUID(testUUID(3)),
		RuntimeSessionID: mustNativeUUID(testUUID(4)),
		RuntimeBindingID: mustNativeUUID(testUUID(5)), BindingGeneration: 7,
	}
	transition := NativeRetirementTransitionV1{
		AttemptID: testUUID(1), RetirementRequestID: testUUID(2),
		DaemonID: "daemon-a", DaemonBootID: testUUID(3),
		RuntimeSessionID: testUUID(4), RuntimeBindingID: testUUID(5),
		BindingGeneration: 7, ChannelBindingDigest: strings.Repeat("a", 64),
		RequestBodyDigest: strings.Repeat("b", 64),
	}
	return attempt, transition
}

func testUUID(last int) string {
	return "00000000-0000-4000-8000-00000000000" + string(rune('0'+last))
}
