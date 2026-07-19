package gateway

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/multica-ai/multica/server/internal/daemon/brain"
)

const (
	defaultRequestTimeout = 30 * time.Second
	defaultMaxBodyBytes   = int64(2 << 20)
	operationLiveness     = "liveness"
	operationReadiness    = "readiness"
	operationModels       = "models"
)

const (
	HeaderTaskID         = "X-Task-Id"
	HeaderSessionID      = "X-Session-Id"
	HeaderRequestID      = "X-Request-Id"
	HeaderContinuationID = "X-Continuation-Id"
)

// CredentialSource owns any authorized secret-file access outside this
// package. Implementations expose a value only for the duration of use and
// must not include that value in returned errors.
type CredentialSource interface {
	WithCredential(context.Context, brain.SecretFileRef, func(string) error) error
}

type EndpointSet struct {
	Liveness  string
	Readiness string
}

func (e EndpointSet) Validate() error {
	if err := validateEndpointPath(e.Liveness); err != nil {
		return fmt.Errorf("liveness endpoint: %w", err)
	}
	if err := validateEndpointPath(e.Readiness); err != nil {
		return fmt.Errorf("readiness endpoint: %w", err)
	}
	return nil
}

type ClientOptions struct {
	Gateway         brain.GatewayConfig
	Endpoints       EndpointSet
	Credential      CredentialSource
	HTTPClient      *http.Client
	RequestTimeout  time.Duration
	MaxResponseBody int64
}

type Client struct {
	baseURL         *url.URL
	secretFile      brain.SecretFileRef
	endpoints       EndpointSet
	credential      CredentialSource
	httpClient      *http.Client
	requestTimeout  time.Duration
	maxResponseBody int64
}

func NewClient(options ClientOptions) (*Client, error) {
	if err := options.Gateway.Validate(); err != nil {
		return nil, &GatewayError{Operation: "client", Class: ErrorInvalidConfiguration}
	}
	if options.Gateway.SecretFile.Path == "" || options.Credential == nil {
		return nil, &GatewayError{Operation: "client", Class: ErrorInvalidConfiguration}
	}
	if err := options.Endpoints.Validate(); err != nil {
		return nil, &GatewayError{Operation: "client", Class: ErrorInvalidConfiguration}
	}
	baseURL, err := url.Parse(options.Gateway.BaseURL)
	if err != nil {
		return nil, &GatewayError{Operation: "client", Class: ErrorInvalidConfiguration}
	}
	timeout := options.RequestTimeout
	if timeout == 0 {
		timeout = defaultRequestTimeout
	}
	if timeout < time.Millisecond || timeout > 2*time.Minute {
		return nil, &GatewayError{Operation: "client", Class: ErrorInvalidConfiguration}
	}
	maxBody := options.MaxResponseBody
	if maxBody == 0 {
		maxBody = defaultMaxBodyBytes
	}
	if maxBody < 1024 || maxBody > 16<<20 {
		return nil, &GatewayError{Operation: "client", Class: ErrorInvalidConfiguration}
	}
	httpClient := options.HTTPClient
	if httpClient == nil {
		httpClient = defaultHTTPClient(timeout)
	}
	httpClientCopy := *httpClient
	httpClientCopy.CheckRedirect = func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &Client{
		baseURL:         baseURL,
		secretFile:      options.Gateway.SecretFile,
		endpoints:       options.Endpoints,
		credential:      options.Credential,
		httpClient:      &httpClientCopy,
		requestTimeout:  timeout,
		maxResponseBody: maxBody,
	}, nil
}

func defaultHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: min(timeout, 10*time.Second), KeepAlive: 30 * time.Second}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialer.DialContext
	transport.ResponseHeaderTimeout = min(timeout, 15*time.Second)
	transport.TLSHandshakeTimeout = min(timeout, 10*time.Second)
	transport.IdleConnTimeout = 90 * time.Second
	return &http.Client{Transport: transport, Timeout: timeout}
}

func (c *Client) String() string {
	if c == nil || c.baseURL == nil {
		return "gateway.Client{unconfigured}"
	}
	return fmt.Sprintf("gateway.Client{base_url:%q, authentication:[redacted]}", c.baseURL.Redacted())
}

func (c *Client) CheckLiveness(ctx context.Context, correlation brain.Correlation) (ProbeResult, error) {
	return c.probe(ctx, operationLiveness, c.endpoints.Liveness, correlation, false)
}

func (c *Client) CheckReadiness(ctx context.Context, correlation brain.Correlation) (ProbeResult, error) {
	return c.probe(ctx, operationReadiness, c.endpoints.Readiness, correlation, true)
}

func (c *Client) probe(ctx context.Context, operation, endpoint string, correlation brain.Correlation, authenticated bool) (ProbeResult, error) {
	response, err := c.do(ctx, operation, http.MethodGet, endpoint, correlation, authenticated)
	if err != nil {
		return ProbeResult{}, err
	}
	defer response.Body.Close()
	if err := drainBounded(response.Body, c.maxResponseBody); err != nil {
		return ProbeResult{}, &GatewayError{Operation: operation, Class: ErrorProtocol}
	}
	return ProbeResult{
		StatusCode: response.StatusCode,
		RequestID:  safeIdentifier(response.Header.Get(HeaderOmniRouteRequestID)),
	}, nil
}

func (c *Client) FetchModels(ctx context.Context, correlation brain.Correlation) (ModelsDocument, error) {
	response, err := c.do(ctx, operationModels, http.MethodGet, "/v1/models", correlation, true)
	if err != nil {
		return ModelsDocument{}, err
	}
	defer response.Body.Close()
	body, err := readBounded(response.Body, c.maxResponseBody)
	if err != nil {
		return ModelsDocument{}, &GatewayError{Operation: operationModels, Class: ErrorProtocol}
	}
	var document ModelsDocument
	if err := json.Unmarshal(body, &document); err != nil {
		return ModelsDocument{}, &GatewayError{Operation: operationModels, Class: ErrorProtocol}
	}
	headerVersion := strings.TrimSpace(response.Header.Get(HeaderRegistryVersion))
	if headerVersion != "" {
		if document.RegistryVersion != "" && document.RegistryVersion != headerVersion {
			return ModelsDocument{}, &GatewayError{Operation: operationModels, Class: ErrorProtocol}
		}
		document.RegistryVersion = headerVersion
	}
	return document, nil
}

func (c *Client) do(ctx context.Context, operation, method, endpoint string, correlation brain.Correlation, authenticated bool) (*http.Response, error) {
	if err := correlation.Validate(); err != nil || !validCorrelationHeaders(correlation) {
		return nil, &GatewayError{Operation: operation, Class: ErrorInvalidRequest}
	}
	requestURL, err := resolveEndpoint(c.baseURL, endpoint)
	if err != nil {
		return nil, &GatewayError{Operation: operation, Class: ErrorInvalidConfiguration}
	}
	requestCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestCtx, method, requestURL.String(), nil)
	if err != nil {
		return nil, &GatewayError{Operation: operation, Class: ErrorInvalidConfiguration}
	}
	applyCorrelation(request.Header, correlation)
	request.Header.Set("Accept", "application/json")
	var response *http.Response
	if authenticated {
		var transportErr error
		err = c.credential.WithCredential(requestCtx, c.secretFile, func(credential string) error {
			if !validCredential(credential) {
				return errInvalidCredential
			}
			request.Header.Set("Authorization", "Bearer "+credential)
			response, transportErr = c.httpClient.Do(request)
			request.Header.Del("Authorization")
			return nil
		})
		if err != nil {
			if errors.Is(err, context.Canceled) || errors.Is(requestCtx.Err(), context.Canceled) {
				return nil, &GatewayError{Operation: operation, Class: ErrorCancelled}
			}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(requestCtx.Err(), context.DeadlineExceeded) {
				return nil, &GatewayError{Operation: operation, Class: ErrorTimeout, Retryable: true}
			}
			if errors.Is(err, errInvalidCredential) {
				return nil, &GatewayError{Operation: operation, Class: ErrorAuthentication}
			}
			return nil, &GatewayError{Operation: operation, Class: ErrorAuthentication}
		}
		if transportErr != nil {
			return nil, classifyTransportError(operation, requestCtx, transportErr)
		}
	} else {
		response, err = c.httpClient.Do(request)
		if err != nil {
			return nil, classifyTransportError(operation, requestCtx, err)
		}
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_ = response.Body.Close()
		return nil, classifyStatus(operation, response)
	}
	return response, nil
}

var errInvalidCredential = errors.New("invalid credential")

func validCredential(value string) bool {
	return value != "" && len(value) <= 4096 && strings.TrimSpace(value) == value && !strings.ContainsAny(value, "\r\n\x00")
}

func classifyTransportError(operation string, ctx context.Context, err error) error {
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		return &GatewayError{Operation: operation, Class: ErrorCancelled}
	}
	var networkError net.Error
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) || errors.As(err, &networkError) && networkError.Timeout() {
		return &GatewayError{Operation: operation, Class: ErrorTimeout, Retryable: true}
	}
	return &GatewayError{Operation: operation, Class: ErrorTransport, Retryable: true}
}

func validateEndpointPath(value string) error {
	if value == "" || !strings.HasPrefix(value, "/") {
		return errors.New("must be an absolute path")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Host != "" || parsed.Scheme != "" {
		return errors.New("must not contain a host, query, or fragment")
	}
	if path.Clean(value) != value || strings.Contains(value, "..") {
		return errors.New("must be normalized")
	}
	return nil
}

func resolveEndpoint(baseURL *url.URL, endpoint string) (*url.URL, error) {
	if err := validateEndpointPath(endpoint); err != nil {
		return nil, err
	}
	copyURL := *baseURL
	copyURL.Path = strings.TrimRight(copyURL.Path, "/") + endpoint
	copyURL.RawPath = ""
	return &copyURL, nil
}

func validCorrelationHeaders(correlation brain.Correlation) bool {
	values := []string{correlation.TaskID, correlation.SessionID, correlation.RequestID, correlation.ContinuationID}
	for index, value := range values {
		if index == len(values)-1 && value == "" {
			continue
		}
		if len(value) > 256 || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\r\n\x00") {
			return false
		}
	}
	return true
}

func applyCorrelation(header http.Header, correlation brain.Correlation) {
	header.Set(HeaderTaskID, correlation.TaskID)
	header.Set(HeaderSessionID, correlation.SessionID)
	header.Set(HeaderRequestID, correlation.RequestID)
	if correlation.ContinuationID != "" {
		header.Set(HeaderContinuationID, correlation.ContinuationID)
	}
}

func readBounded(reader io.Reader, limit int64) ([]byte, error) {
	body, err := io.ReadAll(io.LimitReader(reader, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errors.New("response exceeds configured limit")
	}
	return body, nil
}

func drainBounded(reader io.Reader, limit int64) error {
	_, err := readBounded(reader, limit)
	return err
}
