package gateway

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type ErrorClass string

const (
	ErrorInvalidConfiguration ErrorClass = "invalid_configuration"
	ErrorCancelled            ErrorClass = "cancelled"
	ErrorTimeout              ErrorClass = "timeout"
	ErrorTransport            ErrorClass = "transport"
	ErrorAuthentication       ErrorClass = "authentication"
	ErrorAuthorization        ErrorClass = "authorization"
	ErrorInvalidRequest       ErrorClass = "invalid_request"
	ErrorNotFound             ErrorClass = "not_found"
	ErrorUnknownModel         ErrorClass = "unknown_model"
	ErrorRateLimited          ErrorClass = "rate_limited"
	ErrorOverloaded           ErrorClass = "overloaded"
	ErrorUpstream             ErrorClass = "upstream"
	ErrorProtocol             ErrorClass = "protocol"
	ErrorCapability           ErrorClass = "capability"
)

// GatewayError deliberately omits response bodies, URLs, headers, credentials,
// and source errors so it is safe for structured diagnostics.
type GatewayError struct {
	Operation  string
	Class      ErrorClass
	StatusCode int
	Retryable  bool
	RetryAfter time.Duration
	RequestID  string
}

func (e *GatewayError) Error() string {
	if e == nil {
		return "gateway error"
	}
	if e.StatusCode != 0 {
		return fmt.Sprintf("gateway %s failed: class=%s status=%d", e.Operation, e.Class, e.StatusCode)
	}
	return fmt.Sprintf("gateway %s failed: class=%s", e.Operation, e.Class)
}

func IsErrorClass(err error, class ErrorClass) bool {
	var gatewayErr *GatewayError
	return errors.As(err, &gatewayErr) && gatewayErr.Class == class
}

func classifyStatus(operation string, response *http.Response) *GatewayError {
	result := &GatewayError{
		Operation:  operation,
		StatusCode: response.StatusCode,
		RequestID:  safeIdentifier(response.Header.Get(HeaderOmniRouteRequestID)),
	}
	switch response.StatusCode {
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		result.Class = ErrorInvalidRequest
	case http.StatusUnauthorized:
		result.Class = ErrorAuthentication
	case http.StatusForbidden:
		result.Class = ErrorAuthorization
	case http.StatusNotFound:
		result.Class = ErrorNotFound
	case http.StatusRequestTimeout:
		result.Class = ErrorTimeout
		result.Retryable = true
	case http.StatusTooManyRequests:
		result.Class = ErrorRateLimited
		result.Retryable = true
		result.RetryAfter = parseRetryAfter(response.Header.Get("Retry-After"), time.Now())
	case http.StatusServiceUnavailable:
		result.Class = ErrorOverloaded
		result.Retryable = true
		result.RetryAfter = parseRetryAfter(response.Header.Get("Retry-After"), time.Now())
	default:
		if response.StatusCode >= 500 {
			result.Class = ErrorUpstream
			result.Retryable = true
		} else {
			result.Class = ErrorProtocol
		}
	}
	return result
}

func parseRetryAfter(raw string, now time.Time) time.Duration {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	if seconds, err := strconv.ParseInt(raw, 10, 64); err == nil {
		if seconds <= 0 || seconds > int64((24*time.Hour)/time.Second) {
			return 0
		}
		return time.Duration(seconds) * time.Second
	}
	when, err := http.ParseTime(raw)
	if err != nil || !when.After(now) || when.Sub(now) > 24*time.Hour {
		return 0
	}
	return when.Sub(now)
}
