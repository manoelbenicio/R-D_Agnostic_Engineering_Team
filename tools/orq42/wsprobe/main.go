// Command wsprobe verifies the realtime /ws authentication contract without a
// third-party WebSocket library and without ever logging the token.
//
// ORQ-42 Q-I. The rotation runbook needs a gate proving that (a) a valid token
// is accepted and, on the first-frame path, acknowledged with auth_ack, and
// (b) the pre-rotation token is rejected. No WebSocket client exists on the
// target host, and installing one is out of scope, so this probe speaks the
// handshake and the minimum framing it needs using only the standard library.
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
const (
	labelAccepted   = "WS_ACCEPTED"       // cookie mode: 101
	labelAuthAck    = "WS_AUTH_ACK"       // first-frame mode: auth_ack received
	labelRejected   = "WS_REJECTED"       // cookie mode: 401/403
	labelAuthDenied = "WS_AUTH_DENIED"    // first-frame mode: error frame or close
	labelBadRequest = "WS_BAD_REQUEST"    // 400: workspace_id/workspace_slug missing
	labelUsage      = "WS_STOP_USAGE"     //
	labelTokenInput = "WS_STOP_TOKEN"     //
	labelDial       = "WS_STOP_DIAL"      //
	labelHandshake  = "WS_STOP_HANDSHAKE" //
	labelProtocol   = "WS_STOP_PROTOCOL"  //
	labelTimeout    = "WS_STOP_TIMEOUT"   //
	labelStatus     = "WS_STOP_STATUS"    // any other HTTP status
)

const (
	maxTokenBytes = 8192
	maxFrameBytes = 1 << 16
	opText        = 0x1
	opClose       = 0x8
)

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
	// Verify the server proved it understood the handshake.
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
	if resp.Header.Get("Sec-WebSocket-Accept") != base64.StdEncoding.EncodeToString(sum[:]) {
		conn.Close()
		return nil, nil, status, stop(labelHandshake)
	}
	return conn, br, status, nil
}

// writeTextFrame writes a masked client text frame, as RFC 6455 requires.
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
		if errors.Is(err, os.ErrDeadlineExceeded) {
			return 0, nil, stop(labelTimeout)
		}
		return 0, nil, stop(labelProtocol)
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
	case http.StatusUnauthorized, http.StatusForbidden:
		return labelRejected, nil
	case http.StatusBadRequest:
		return labelBadRequest, nil
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
		if status == http.StatusBadRequest {
			return labelBadRequest, nil
		}
		return "", stop(labelStatus)
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

	for i := 0; i < 4; i++ {
		opcode, payload, err := readFrame(br)
		if err != nil {
			var s *stopError
			if errors.As(err, &s) && s.label == labelProtocol {
				// A closed connection without auth_ack is a denial.
				return labelAuthDenied, nil
			}
			return "", err
		}
		switch opcode {
		case opClose:
			return labelAuthDenied, nil
		case opText:
			var msg struct {
				Type  string `json:"type"`
				Error string `json:"error"`
			}
			if err := json.Unmarshal(payload, &msg); err != nil {
				continue
			}
			if msg.Type == "auth_ack" {
				return labelAuthAck, nil
			}
			if msg.Error != "" {
				return labelAuthDenied, nil
			}
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
	return 3 // a definite rejection is not a tool failure
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}
