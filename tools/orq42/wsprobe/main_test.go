package main

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

const (
	goodToken = "good-token-value-not-a-real-jwt"
	oldToken  = "old-token-value-not-a-real-jwt"
)

// fakeHub reproduces the two authentication paths of
// internal/realtime/hub.go closely enough to exercise the probe:
//   - workspace_id (or workspace_slug) required, else 400
//   - a multica_auth cookie is validated BEFORE the upgrade: 401 when invalid
//   - without a cookie, upgrade and expect {"type":"auth","payload":{"token"}}
//     as the first frame, answering {"type":"auth_ack"} or an error frame
func fakeHub(t *testing.T, accept string) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("workspace_id") == "" && r.URL.Query().Get("workspace_slug") == "" {
			http.Error(w, `{"error":"workspace_id or workspace_slug required"}`, http.StatusBadRequest)
			return
		}
		if c, err := r.Cookie("multica_auth"); err == nil && c.Value != "" {
			if c.Value != accept {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}
			upgradeAndHold(t, w, r)
			return
		}
		conn, br := upgradeRaw(t, w, r)
		if conn == nil {
			return
		}
		defer conn.Close()
		opcode, payload, err := readFrame(br)
		if err != nil || opcode != opText {
			return
		}
		var msg struct {
			Type    string `json:"type"`
			Payload struct {
				Token string `json:"token"`
			} `json:"payload"`
		}
		if json.Unmarshal(payload, &msg) != nil || msg.Type != "auth" || msg.Payload.Token == "" {
			writeServerText(conn, []byte(`{"error":"expected auth message as first frame"}`))
			return
		}
		if msg.Payload.Token != accept {
			writeServerText(conn, []byte(`{"error":"invalid token"}`))
			return
		}
		writeServerText(conn, []byte(`{"type":"auth_ack"}`))
		time.Sleep(20 * time.Millisecond)
	}))
}

// upgradeRaw completes the server side of the handshake and hands back the
// hijacked connection.
func upgradeRaw(t *testing.T, w http.ResponseWriter, r *http.Request) (net.Conn, *bufio.Reader) {
	t.Helper()
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" || !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "bad upgrade", http.StatusBadRequest)
		return nil, nil
	}
	hj, ok := w.(http.Hijacker)
	if !ok {
		t.Fatal("ResponseWriter does not support hijacking")
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		t.Fatalf("hijack: %v", err)
	}
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	resp := "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + base64.StdEncoding.EncodeToString(sum[:]) + "\r\n\r\n"
	if _, err := rw.WriteString(resp); err != nil || rw.Flush() != nil {
		conn.Close()
		return nil, nil
	}
	return conn, rw.Reader
}

func upgradeAndHold(t *testing.T, w http.ResponseWriter, r *http.Request) {
	conn, _ := upgradeRaw(t, w, r)
	if conn == nil {
		return
	}
	time.Sleep(20 * time.Millisecond)
	conn.Close()
}

// writeServerText writes an unmasked server text frame.
func writeServerText(conn net.Conn, payload []byte) {
	header := []byte{0x80 | opText, byte(len(payload))}
	conn.Write(append(header, payload...))
}

func probe(t *testing.T, token string, args ...string) (int, string, string) {
	t.Helper()
	var out, errBuf bytes.Buffer
	code := run(args, strings.NewReader(token), &out, &errBuf)
	return code, strings.TrimSpace(out.String()), strings.TrimSpace(errBuf.String())
}

func TestProbeCookie_AcceptsValidTokenAndRejectsOld(t *testing.T) {
	srv := fakeHub(t, goodToken)
	defer srv.Close()
	u := srv.URL + "/ws?workspace_id=11111111-1111-1111-1111-111111111111"

	code, out, errOut := probe(t, goodToken, "-url", u, "-mode", "cookie")
	if code != 0 || out != labelAccepted {
		t.Fatalf("valid cookie: expected %s exit 0, got code=%d out=%q err=%q",
			labelAccepted, code, out, errOut)
	}

	code, out, errOut = probe(t, oldToken, "-url", u, "-mode", "cookie")
	if code != 3 || out != labelRejected {
		t.Fatalf("old cookie: expected %s exit 3, got code=%d out=%q err=%q",
			labelRejected, code, out, errOut)
	}
}

func TestProbeFirstFrame_RequiresAuthAckAndDeniesOld(t *testing.T) {
	srv := fakeHub(t, goodToken)
	defer srv.Close()
	u := srv.URL + "/ws?workspace_id=11111111-1111-1111-1111-111111111111"

	code, out, errOut := probe(t, goodToken, "-url", u, "-mode", "first-frame")
	if code != 0 || out != labelAuthAck {
		t.Fatalf("valid first-frame: expected %s exit 0, got code=%d out=%q err=%q",
			labelAuthAck, code, out, errOut)
	}

	code, out, errOut = probe(t, oldToken, "-url", u, "-mode", "first-frame")
	if code != 3 || out != labelRejected {
		t.Fatalf("old first-frame: expected %s exit 3, got code=%d out=%q err=%q",
			labelRejected, code, out, errOut)
	}
}

// A 400 means the request was wrong, not the token: it must be exit 1, never a
// rejection verdict.
func TestProbe_MissingWorkspaceIsAStopNotAVerdict(t *testing.T) {
	srv := fakeHub(t, goodToken)
	defer srv.Close()
	for _, mode := range []string{"cookie", "first-frame"} {
		code, out, errOut := probe(t, goodToken, "-url", srv.URL+"/ws", "-mode", mode)
		if code != 1 || out != "" || errOut != labelBadRequest {
			t.Fatalf("%s: expected %s exit 1 on stderr, got code=%d out=%q err=%q",
				mode, labelBadRequest, code, out, errOut)
		}
	}
}

// Only a numeric loopback host is allowed, and the check happens BEFORE stdin is
// read, so a token is never consumed for a refused destination.
func TestProbe_RefusesNonLoopbackTargetsBeforeReadingStdin(t *testing.T) {
	for _, raw := range []string{
		"http://example.com/ws?workspace_id=x",
		"http://localhost:18080/ws?workspace_id=x", // a name, not an address
		"http://10.0.0.5:18080/ws?workspace_id=x",
		"http://100.118.244.61:18080/ws?workspace_id=x", // the real ORQ1 tailnet IP
		"http://[2001:db8::1]:18080/ws?workspace_id=x",
	} {
		// Empty stdin: if the target were validated after the token read, this
		// would fail with WS_STOP_TOKEN instead.
		code, out, errOut := probe(t, "", "-url", raw)
		if code != 2 || out != "" || errOut != labelTarget {
			t.Fatalf("%s: expected %s exit 2, got code=%d out=%q err=%q",
				raw, labelTarget, code, out, errOut)
		}
	}
	// Both loopback literals are accepted by the guard itself.
	for _, raw := range []string{"http://127.0.0.1:18080/ws", "http://[::1]:18080/ws"} {
		u, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("parse %s: %v", raw, err)
		}
		if err := requireNumericLoopback(u); err != nil {
			t.Fatalf("%s should be accepted, got %v", raw, err)
		}
	}
}

// A 101 that is not a real upgrade must not be reported as acceptance.
func TestProbe_RejectsFake101WithoutUpgradeHeaders(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hj, ok := w.(http.Hijacker)
		if !ok {
			t.Fatal("no hijack")
		}
		conn, rw, err := hj.Hijack()
		if err != nil {
			t.Fatalf("hijack: %v", err)
		}
		defer conn.Close()
		// 101 with no Upgrade/Connection/Accept headers at all.
		rw.WriteString("HTTP/1.1 101 Switching Protocols\r\nContent-Length: 0\r\n\r\n")
		rw.Flush()
	}))
	defer srv.Close()
	code, out, errOut := probe(t, goodToken, "-url", srv.URL+"/ws?workspace_id=x")
	if code != 1 || out != "" || errOut != labelHandshake {
		t.Fatalf("expected %s exit 1, got code=%d out=%q err=%q", labelHandshake, code, out, errOut)
	}
}

// An error frame that is not exactly `invalid token` is inconclusive, and a close
// or an EOF before any verdict likewise. None may be exit 3.
func TestProbeFirstFrame_InconclusiveOutcomesAreExitOne(t *testing.T) {
	cases := []struct {
		name  string
		reply func(conn net.Conn)
		want  string
	}{
		{"other_error", func(c net.Conn) {
			writeServerText(c, []byte(`{"error":"not a member of this workspace"}`))
		}, labelAuthError},
		{"auth_timeout_error", func(c net.Conn) {
			writeServerText(c, []byte(`{"error":"auth timeout or read error"}`))
		}, labelAuthError},
		{"close_frame", func(c net.Conn) {
			c.Write([]byte{0x88, 0x00})
		}, labelClosed},
		{"eof", func(c net.Conn) { c.Close() }, labelEOF},
		{"malformed_json", func(c net.Conn) { writeServerText(c, []byte("not json")) }, labelMalformed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				conn, br := upgradeRaw(t, w, r)
				if conn == nil {
					return
				}
				defer conn.Close()
				if _, _, err := readFrame(br); err != nil {
					return
				}
				tc.reply(conn)
				time.Sleep(20 * time.Millisecond)
			}))
			defer srv.Close()
			code, out, errOut := probe(t, goodToken,
				"-url", srv.URL+"/ws?workspace_id=x", "-mode", "first-frame", "-timeout", "3s")
			if code != 1 || out != "" || errOut != tc.want {
				t.Fatalf("expected %s exit 1, got code=%d out=%q err=%q", tc.want, code, out, errOut)
			}
		})
	}
}

// The probe must never write the token anywhere. This is the property that
// makes it usable in an authorised window with a real owner session.
func TestProbe_NeverEmitsTheToken(t *testing.T) {
	srv := fakeHub(t, goodToken)
	defer srv.Close()
	u := srv.URL + "/ws?workspace_id=11111111-1111-1111-1111-111111111111"

	for _, tc := range []struct{ mode, token string }{
		{"cookie", goodToken}, {"cookie", oldToken},
		{"first-frame", goodToken}, {"first-frame", oldToken},
	} {
		_, out, errOut := probe(t, tc.token, "-url", u, "-mode", tc.mode)
		if strings.Contains(out, tc.token) || strings.Contains(errOut, tc.token) {
			t.Fatalf("%s leaked the token: out=%q err=%q", tc.mode, out, errOut)
		}
		for _, s := range []string{out, errOut} {
			if s != "" && !strings.HasPrefix(s, "WS_") {
				t.Fatalf("%s emitted a non-label line: %q", tc.mode, s)
			}
		}
	}
}

func TestProbe_TokenComesFromStdinOnly(t *testing.T) {
	srv := fakeHub(t, goodToken)
	defer srv.Close()
	u := srv.URL + "/ws?workspace_id=11111111-1111-1111-1111-111111111111"

	// Empty stdin must be refused before any connection is attempted.
	code, _, errOut := probe(t, "", "-url", u, "-mode", "cookie")
	if code != 2 || errOut != labelTokenInput {
		t.Fatalf("empty stdin: expected %s exit 2, got code=%d err=%q", labelTokenInput, code, errOut)
	}
	// A trailing newline from `printf '%s\n'` must be tolerated.
	code, out, _ := probe(t, goodToken+"\n", "-url", u, "-mode", "cookie")
	if code != 0 || out != labelAccepted {
		t.Fatalf("trailing newline: expected %s exit 0, got code=%d out=%q", labelAccepted, code, out)
	}
}

func TestProbe_UsageAndDialFailures(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"-url", "http://127.0.0.1:1/ws", "-mode", "bogus"},
		{"-url", "ftp://127.0.0.1/ws"},
		{"-url", "http://127.0.0.1:18080/ws", "-timeout", "0"},
	} {
		code, _, errOut := probe(t, goodToken, args...)
		if code != 2 || errOut != labelUsage {
			t.Fatalf("args %v: expected %s exit 2, got code=%d err=%q", args, labelUsage, code, errOut)
		}
	}
	// Closed port: a dial failure must be a tool stop, not a verdict.
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := l.Addr().String()
	l.Close()
	code, out, errOut := probe(t, goodToken,
		"-url", "http://"+addr+"/ws?workspace_id=x", "-timeout", "2s")
	if code != 1 || out != "" || errOut != labelDial {
		t.Fatalf("closed port: expected %s exit 1, got code=%d out=%q err=%q",
			labelDial, code, out, errOut)
	}
}

func TestReadFrame_RejectsOversizedAndHugeFrames(t *testing.T) {
	// 127-length frames are refused outright.
	br := bufio.NewReader(bytes.NewReader([]byte{0x81, 127}))
	if _, _, err := readFrame(br); err == nil {
		t.Fatal("expected refusal of a 64-bit length frame")
	}
	// A truncated frame is a protocol stop, not a panic.
	br = bufio.NewReader(bytes.NewReader([]byte{0x81, 10, 'a'}))
	if _, _, err := readFrame(br); err == nil {
		t.Fatal("expected refusal of a truncated frame")
	}
}

func TestWriteTextFrame_MasksPayload(t *testing.T) {
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()

	payload := []byte(`{"type":"auth"}`)
	done := make(chan error, 1)
	go func() { done <- writeTextFrame(client, payload) }()

	header := make([]byte, 2)
	if _, err := io.ReadFull(server, header); err != nil {
		t.Fatalf("read header: %v", err)
	}
	if header[1]&0x80 == 0 {
		t.Fatal("client frames must set the mask bit")
	}
	mask := make([]byte, 4)
	if _, err := io.ReadFull(server, mask); err != nil {
		t.Fatalf("read mask: %v", err)
	}
	masked := make([]byte, len(payload))
	if _, err := io.ReadFull(server, masked); err != nil {
		t.Fatalf("read payload: %v", err)
	}
	if bytes.Equal(masked, payload) {
		t.Fatal("payload reached the wire unmasked")
	}
	for i := range masked {
		masked[i] ^= mask[i%4]
	}
	if !bytes.Equal(masked, payload) {
		t.Fatalf("unmasking mismatch: %q", masked)
	}
	if err := <-done; err != nil {
		t.Fatalf("writeTextFrame: %v", err)
	}
}
