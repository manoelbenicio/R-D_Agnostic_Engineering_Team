// Command wsprobe verifies the realtime /ws authentication contract without a
// third-party WebSocket library and without ever logging the token.
//
// ORQ-42 Q-I. The rotation runbook needs a gate proving that (a) a valid token
// is accepted and, on the first-frame path, acknowledged with auth_ack, and
// (b) the pre-rotation token is rejected. No WebSocket client exists on the
// target host, and installing one is out of scope, so this probe speaks the
// handshake and the minimum framing it needs using only the standard library.
//
// Pinned server contract (multica-auth-work/server/internal/realtime/hub.go,
// sha256 d5e2dbc654b2316aa79435c4a833827039a9323fb48500817b0cbb2e2c5a7a87):
//
//	:770-775  a multica_auth cookie is verified BEFORE the upgrade, so an invalid
//	          cookie yields HTTP 401 and never reaches the WebSocket layer.
//	:719-728  without a cookie, the first frame must be
//	          {"type":"auth","payload":{"token":"..."}}.
//	:809-816  on success the server writes {"type":"auth_ack"} as the FIRST frame
//	          after authentication. auth_ack therefore arrives before any other
//	          traffic, which is why a small frame cap is sufficient.
//	:678,:682,:694  a failed verification emits exactly {"error":"invalid token"}.
//	:716      {"error":"auth timeout or read error"} and :726 {"error":"expected
//	          auth message as first frame"} are NOT credential verdicts.
//
// Consequence for the exit contract: exhausting the frame cap, a close frame, an
// EOF, a malformed frame, a timeout, or any error text other than the exact
// `invalid token` are all exit 1 ("could not measure"), never exit 3. Only a
// cookie-mode 401 and that exact error text are rejections.
//
// Two modes mirror the two server paths in internal/realtime/hub.go:
//
//	-mode cookie        send the token as the multica_auth cookie. The server
//	                    validates it BEFORE the upgrade, so the verdict is the
//	                    HTTP status: 101 accepted, 401 rejected.
//	-mode first-frame   upgrade unauthenticated, then send
//	                    {"type":"auth","payload":{"token":"..."}} and require a
//	                    {"type":"auth_ack"} text frame back.
//
// The token is read from stdin only. It never appears in argv, in any log line,
// or in the emitted verdict. Output is one fixed label.
package main

import (
	"bufio"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Fixed labels. The token is never part of any of them.
// Only two labels mean "the server gave a definite verdict" (exit 0), and only
// two mean "the server definitely rejected this credential" (exit 3). Everything
// else is "could not measure" (exit 1), because a gate must never read an
// inconclusive result as a rejection.
const (
	labelAccepted = "WS_ACCEPTED" // exit 0: cookie mode, 101
	labelAuthAck  = "WS_AUTH_ACK" // exit 0: first-frame mode, auth_ack received

	labelRejected = "WS_REJECTED" // exit 3: cookie 401, or first-frame `invalid token`

	labelUsage      = "WS_STOP_USAGE"       // exit 2
	labelTarget     = "WS_STOP_TARGET"      // exit 2: not a numeric loopback host
	labelTokenInput = "WS_STOP_TOKEN"       // exit 2
	labelDial       = "WS_STOP_DIAL"        // exit 1
	labelHandshake  = "WS_STOP_HANDSHAKE"   // exit 1: 101 without a valid upgrade
	labelBadRequest = "WS_STOP_BAD_REQUEST" // exit 1: 400, the request was wrong, not the token
	labelForbidden  = "WS_STOP_FORBIDDEN"   // exit 1: 403, membership, not the token
	labelStatus     = "WS_STOP_STATUS"      // exit 1: any other HTTP status
	labelClosed     = "WS_STOP_CLOSED"      // exit 1: close frame before any verdict
	labelEOF        = "WS_STOP_EOF"         // exit 1: connection ended before any verdict
	labelMalformed  = "WS_STOP_MALFORMED"   // exit 1: unparseable frame
	labelAuthError  = "WS_STOP_AUTH_ERROR"  // exit 1: an error that is NOT `invalid token`
	labelProtocol   = "WS_STOP_PROTOCOL"    // exit 1: framing violation or frame cap
	labelTimeout    = "WS_STOP_TIMEOUT"     // exit 1
)

// errInvalidToken is the exact body internal/realtime/hub.go emits for a token
// that failed verification: `{"error":"invalid token"}` at hub.go:678, :682 and
// :694 (hub.go sha256 d5e2dbc654b2316aa79435c4a833827039a9323fb48500817b0cbb2e2c5a7a87).
// Anything else - `auth timeout or read error` (:716), `expected auth message as
// first frame` (:726), `not a member of this workspace` - is NOT a credential
// verdict and must not be reported as one.
const errInvalidToken = "invalid token"

const (
	maxTokenBytes = 8192
	maxFrameBytes = 1 << 16
	// maxVerdictFrames caps how many frames may precede the verdict.
	maxVerdictFrames = 4
	opText           = 0x1
	opClose          = 0x8
)

// requireNumericLoopback refuses anything that is not a numeric loopback
// address. The probe speaks plaintext HTTP only, so a hostname (which could
// resolve anywhere, now or later) or any routable address would put the token on
// a network. `localhost` is refused too: it is a name, not an address.
func requireNumericLoopback(u *url.URL) error {
	host := u.Hostname()
	ip := net.ParseIP(host)
	if ip == nil || !ip.IsLoopback() {
		return stop(labelTarget)
	}
	return nil
}

type stopError struct{ label string }

func (e *stopError) Error() string { return e.label }

func stop(label string) error { return &stopError{label: label} }

// readToken takes the token from stdin only, so it never enters argv.
func readToken(r io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(r, maxTokenBytes+1))
	if err != nil || len(raw) > maxTokenBytes {
		return "", stop(labelTokenInput)
	}
	token := strings.TrimRight(string(raw), "\r\n")
	if token == "" || strings.ContainsAny(token, "\r\n") {
		return "", stop(labelTokenInput)
	}
	return token, nil
}

// handshake performs the HTTP/1.1 upgrade by hand and returns the live
// connection plus the buffered reader positioned after the response headers.
func handshake(target *url.URL, cookie string, timeout time.Duration) (net.Conn, *bufio.Reader, int, error) {
	host := target.Host
	if target.Port() == "" {
		host = net.JoinHostPort(target.Hostname(), "80")
	}
	conn, err := net.DialTimeout("tcp", host, timeout)
	if err != nil {
		return nil, nil, 0, stop(labelDial)
	}
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		conn.Close()
		return nil, nil, 0, stop(labelDial)
	}

	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		conn.Close()
		return nil, nil, 0, stop(labelHandshake)
	}
	key := base64.StdEncoding.EncodeToString(nonce)

	path := target.RequestURI()
	var req bytes.Buffer
	fmt.Fprintf(&req, "GET %s HTTP/1.1\r\n", path)
	fmt.Fprintf(&req, "Host: %s\r\n", target.Host)
	req.WriteString("Upgrade: websocket\r\n")
	req.WriteString("Connection: Upgrade\r\n")
	fmt.Fprintf(&req, "Sec-WebSocket-Key: %s\r\n", key)
	req.WriteString("Sec-WebSocket-Version: 13\r\n")
	if cookie != "" {
		// Written straight to the wire; never logged and never in argv.
		fmt.Fprintf(&req, "Cookie: multica_auth=%s\r\n", cookie)
	}
	req.WriteString("\r\n")
	if _, err := conn.Write(req.Bytes()); err != nil {
		conn.Close()
		return nil, nil, 0, stop(labelHandshake)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		conn.Close()
		return nil, nil, 0, stop(labelHandshake)
	}
	status := resp.StatusCode
	if status != http.StatusSwitchingProtocols {
		io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		conn.Close()
		return nil, nil, status, nil
	}
	// A 101 is only an upgrade if the server said so on all three headers.
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	if resp.Header.Get("Sec-WebSocket-Accept") != base64.StdEncoding.EncodeToString(sum[:]) ||
		!strings.EqualFold(strings.TrimSpace(resp.Header.Get("Upgrade")), "websocket") ||
		!headerHasToken(resp.Header.Get("Connection"), "upgrade") {
		conn.Close()
		return nil, nil, status, stop(labelHandshake)
	}
	return conn, br, status, nil
}

// writeTextFrame writes a masked client text frame, as RFC 6455 requires.
// headerHasToken reports whether a comma-separated header lists the token.
func headerHasToken(value, token string) bool {
	for _, part := range strings.Split(value, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}

func writeTextFrame(conn net.Conn, payload []byte) error {
	var mask [4]byte
	if _, err := rand.Read(mask[:]); err != nil {
		return stop(labelProtocol)
	}
	header := []byte{0x80 | opText}
	n := len(payload)
	switch {
	case n <= 125:
		header = append(header, byte(0x80|n))
	case n <= 0xFFFF:
		header = append(header, 0x80|126, byte(n>>8), byte(n))
	default:
		return stop(labelProtocol)
	}
	header = append(header, mask[:]...)
	masked := make([]byte, n)
	for i := 0; i < n; i++ {
		masked[i] = payload[i] ^ mask[i%4]
	}
	if _, err := conn.Write(append(header, masked...)); err != nil {
		return stop(labelProtocol)
	}
	return nil
}

// readFrame reads one unmasked server frame and returns its opcode and payload.
func readFrame(br *bufio.Reader) (byte, []byte, error) {
	h := make([]byte, 2)
	if _, err := io.ReadFull(br, h); err != nil {
		switch {
		case errors.Is(err, os.ErrDeadlineExceeded):
			return 0, nil, stop(labelTimeout)
		case errors.Is(err, io.EOF):
			return 0, nil, stop(labelEOF)
		default:
			return 0, nil, stop(labelMalformed)
		}
	}
	opcode := h[0] & 0x0F
	masked := h[1]&0x80 != 0
	length := int(h[1] & 0x7F)
	switch length {
	case 126:
		ext := make([]byte, 2)
		if _, err := io.ReadFull(br, ext); err != nil {
			return 0, nil, stop(labelProtocol)
		}
		length = int(ext[0])<<8 | int(ext[1])
	case 127:
		return 0, nil, stop(labelProtocol) // no frame this large is expected
	}
	if length > maxFrameBytes {
		return 0, nil, stop(labelProtocol)
	}
	var mask [4]byte
	if masked {
		if _, err := io.ReadFull(br, mask[:]); err != nil {
			return 0, nil, stop(labelProtocol)
		}
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(br, payload); err != nil {
		return 0, nil, stop(labelProtocol)
	}
	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}
	return opcode, payload, nil
}

// probeCookie mirrors hub.go's pre-upgrade cookie path: the verdict is the
// HTTP status, so no framing is needed.
func probeCookie(target *url.URL, token string, timeout time.Duration) (string, error) {
	conn, _, status, err := handshake(target, token, timeout)
	if conn != nil {
		defer conn.Close()
	}
	if err != nil {
		return "", err
	}
	switch status {
	case http.StatusSwitchingProtocols:
		return labelAccepted, nil
	case http.StatusUnauthorized:
		// The only status that means "this token was rejected".
		return labelRejected, nil
	case http.StatusForbidden:
		// Membership, not the credential.
		return "", stop(labelForbidden)
	case http.StatusBadRequest:
		return "", stop(labelBadRequest)
	default:
		return "", stop(labelStatus)
	}
}

// probeFirstFrame mirrors hub.go's firstMessageAuth path and requires the
// {"type":"auth_ack"} frame for a valid token.
func probeFirstFrame(target *url.URL, token string, timeout time.Duration) (string, error) {
	conn, br, status, err := handshake(target, "", timeout)
	if conn != nil {
		defer conn.Close()
	}
	if err != nil {
		return "", err
	}
	if status != http.StatusSwitchingProtocols {
		switch status {
		case http.StatusBadRequest:
			return "", stop(labelBadRequest)
		case http.StatusForbidden:
			return "", stop(labelForbidden)
		case http.StatusUnauthorized:
			// A cookie was not sent in this mode, so a 401 here is not a verdict
			// about the token carried in the first frame.
			return "", stop(labelStatus)
		default:
			return "", stop(labelStatus)
		}
	}

	body, err := json.Marshal(map[string]any{
		"type":    "auth",
		"payload": map[string]string{"token": token},
	})
	if err != nil {
		return "", stop(labelProtocol)
	}
	if err := writeTextFrame(conn, body); err != nil {
		return "", err
	}
	// Zero the marshalled token copy as soon as it is on the wire.
	for i := range body {
		body[i] = 0
	}

	// hub.go writes auth_ack as the FIRST frame after a successful first-frame
	// authentication (hub.go:809-816), so a small cap is enough. Exhausting it,
	// or a close/EOF before any verdict, is "could not measure" - never a
	// rejection.
	for i := 0; i < maxVerdictFrames; i++ {
		opcode, payload, err := readFrame(br)
		if err != nil {
			return "", err // already a fixed label: timeout, EOF or malformed
		}
		switch opcode {
		case opClose:
			return "", stop(labelClosed)
		case opText:
			var msg struct {
				Type  string `json:"type"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(payload, &msg); err != nil {
				return "", stop(labelMalformed)
			}
			if msg.Type == "auth_ack" {
				return labelAuthAck, nil
			}
			if msg.Error == errInvalidToken {
				return labelRejected, nil
			}
			if msg.Error != "" {
				return "", stop(labelAuthError)
			}
			return "", stop(labelMalformed)
		}
	}
	return "", stop(labelProtocol)
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("wsprobe", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	rawURL := fs.String("url", "", "ws endpoint, e.g. http://127.0.0.1:18080/ws?workspace_id=<uuid>")
	mode := fs.String("mode", "cookie", "cookie|first-frame")
	timeout := fs.Duration("timeout", 10*time.Second, "dial and read deadline")
	if err := fs.Parse(args); err != nil || *rawURL == "" ||
		(*mode != "cookie" && *mode != "first-frame") || *timeout <= 0 {
		fmt.Fprintln(stderr, labelUsage)
		return 2
	}
	target, err := url.Parse(*rawURL)
	if err != nil || target.Host == "" || (target.Scheme != "http" && target.Scheme != "ws") {
		fmt.Fprintln(stderr, labelUsage)
		return 2
	}
	// The target is validated BEFORE stdin is read: a token must never be
	// consumed for a destination this probe would refuse to talk to.
	if err := requireNumericLoopback(target); err != nil {
		fmt.Fprintln(stderr, labelTarget)
		return 2
	}
	token, err := readToken(stdin)
	if err != nil {
		fmt.Fprintln(stderr, labelTokenInput)
		return 2
	}
	defer func() { token = "" }()

	var verdict string
	if *mode == "cookie" {
		verdict, err = probeCookie(target, token, *timeout)
	} else {
		verdict, err = probeFirstFrame(target, token, *timeout)
	}
	if err != nil {
		var s *stopError
		if errors.As(err, &s) {
			fmt.Fprintln(stderr, s.label)
		} else {
			fmt.Fprintln(stderr, labelProtocol)
		}
		return 1
	}
	fmt.Fprintln(stdout, verdict)
	if verdict == labelAccepted || verdict == labelAuthAck {
		return 0
	}
	// Reached only for labelRejected: cookie 401 or the exact `invalid token`
	// error frame. Every inconclusive outcome already returned 1 above.
	return 3
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
