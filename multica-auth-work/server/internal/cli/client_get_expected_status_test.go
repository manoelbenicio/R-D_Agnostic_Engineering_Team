package cli

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGetJSONExpectedStatus(t *testing.T) {
	type response struct {
		ID string `json:"id"`
	}

	t.Run("accepts exact status and sends workspace", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				t.Fatalf("method = %s, want GET", r.Method)
			}
			if got := r.Header.Get("X-Workspace-ID"); got != "ws-contract" {
				t.Fatalf("X-Workspace-ID = %q, want ws-contract", got)
			}
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(response{ID: "issue-1"})
		}))
		defer srv.Close()

		client := NewAPIClient(srv.URL, "ws-contract", "test-token")
		var out response
		if err := client.GetJSONExpectedStatus(context.Background(), "/issue", http.StatusOK, &out); err != nil {
			t.Fatalf("GetJSONExpectedStatus: %v", err)
		}
		if out.ID != "issue-1" {
			t.Fatalf("id = %q, want issue-1", out.ID)
		}
	})

	for _, status := range []int{http.StatusFound, http.StatusCreated, http.StatusAccepted, http.StatusNoContent} {
		status := status
		t.Run(http.StatusText(status), func(t *testing.T) {
			const privateBody = `{"id":"must-not-be-rendered"}`
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, privateBody)
			}))
			defer srv.Close()

			client := NewAPIClient(srv.URL, "ws-contract", "")
			var out response
			err := client.GetJSONExpectedStatus(context.Background(), "/issue", http.StatusOK, &out)
			var statusErr *UnexpectedStatusError
			if !errors.As(err, &statusErr) {
				t.Fatalf("error = %T %v, want *UnexpectedStatusError", err, err)
			}
			if statusErr.Expected != http.StatusOK || statusErr.Actual != status {
				t.Fatalf("status error = %#v", statusErr)
			}
			if strings.Contains(err.Error(), "must-not-be-rendered") {
				t.Fatalf("unexpected status error leaked response body: %v", err)
			}
			if out.ID != "" {
				t.Fatalf("response decoded on status mismatch: %#v", out)
			}
		})
	}

	t.Run("preserves HTTPError for error statuses", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = io.WriteString(w, `{"error":"issue not found"}`)
		}))
		defer srv.Close()

		client := NewAPIClient(srv.URL, "ws-contract", "")
		err := client.GetJSONExpectedStatus(context.Background(), "/issue", http.StatusOK, &response{})
		var httpErr *HTTPError
		if !errors.As(err, &httpErr) {
			t.Fatalf("error = %T %v, want *HTTPError", err, err)
		}
		if httpErr.StatusCode != http.StatusNotFound {
			t.Fatalf("status = %d, want 404", httpErr.StatusCode)
		}
	})

	t.Run("rejects redirect without reaching Location", func(t *testing.T) {
		destinationCalls := 0
		destination := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			destinationCalls++
			_, _ = io.WriteString(w, `{"id":"destination"}`)
		}))
		defer destination.Close()
		source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Location", destination.URL)
			w.WriteHeader(http.StatusFound)
		}))
		defer source.Close()

		client := NewAPIClient(source.URL, "ws-contract", "")
		var out response
		err := client.GetJSONExpectedStatus(context.Background(), "/issue", http.StatusOK, &out)
		var statusErr *UnexpectedStatusError
		if !errors.As(err, &statusErr) || statusErr.Actual != http.StatusFound {
			t.Fatalf("error = %T %v, want redirect UnexpectedStatusError", err, err)
		}
		if destinationCalls != 0 || out.ID != "" {
			t.Fatalf("destination calls=%d out=%#v, want 0/empty", destinationCalls, out)
		}
	})

	for name, body := range map[string]string{
		"invalid JSON":         `{`,
		"wrong top-level type": `[]`,
		"multiple JSON values": `{"id":"one"}\n{"id":"two"}`,
	} {
		name, body := name, body
		t.Run(name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_, _ = io.WriteString(w, body)
			}))
			defer srv.Close()

			client := NewAPIClient(srv.URL, "ws-contract", "")
			if err := client.GetJSONExpectedStatus(context.Background(), "/issue", http.StatusOK, &response{}); err == nil {
				t.Fatal("expected decode error")
			}
		})
	}

	t.Run("legacy GetJSON behavior remains compatible", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusCreated)
			_, _ = io.WriteString(w, `{"id":"legacy"}`)
		}))
		defer srv.Close()

		client := NewAPIClient(srv.URL, "", "")
		var out response
		if err := client.GetJSON(context.Background(), "/legacy", &out); err != nil {
			t.Fatalf("legacy GetJSON changed: %v", err)
		}
		if out.ID != "legacy" {
			t.Fatalf("legacy id = %q", out.ID)
		}
	})
}
