package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/multica-ai/multica/server/internal/auth"
)

func TestCreateDaemonTokenRejectsMalformedOrMissingDaemon(t *testing.T) {
	for name, body := range map[string]string{
		"malformed":      `{`,
		"missing daemon": `{ "expires_at": "2030-01-01T00:00:00Z" }`,
	} {
		t.Run(name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/workspaces/00000000-0000-0000-0000-000000000001/daemon-tokens", strings.NewReader(body))
			w := httptest.NewRecorder()
			(&Handler{}).CreateDaemonToken(w, r)
			if w.Code != 400 {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}

func TestGenerateDaemonTokenIsOpaqueAndHashable(t *testing.T) {
	raw, err := auth.GenerateDaemonToken()
	if err != nil || !strings.HasPrefix(raw, "mdt_") || len(raw) != 44 {
		t.Fatalf("token shape invalid: %q err=%v", raw, err)
	}
	if auth.HashToken(raw) == raw || len(auth.HashToken(raw)) != 64 {
		t.Fatalf("token hash is not a one-way 64-char digest")
	}
}
