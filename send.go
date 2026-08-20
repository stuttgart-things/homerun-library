/*
Copyright © 2025 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"text/template"
	"time"
)

var (
	contentType = "application/json"

	// HomeRunBodyData is the default request body template for SendToHomerun.
	//
	// Every value goes through the esc function, which JSON-escapes it. Without
	// that, a field containing a quote, a newline or a backslash - a CI failure
	// message, a stack trace, a Windows path - produces a document the receiver
	// rejects, and a crafted field can inject or override sibling keys.
	HomeRunBodyData = `{
		"Title": "{{ esc .Title }}",
		"Message": "{{ esc .Message }}",
		"Severity": "{{ esc .Severity }}",
		"Author": "{{ esc .Author }}",
		"Timestamp": "{{ esc .Timestamp }}",
		"System": "{{ esc .System }}",
		"Tags": "{{ esc .Tags }}",
		"AssigneeAddress": "{{ esc .AssigneeAddress }}",
		"AssigneeName": "{{ esc .AssigneeName }}",
		"Artifacts": "{{ esc .Artifacts }}",
		"Url": "{{ esc .Url }}"
	}`

	// templateFuncs are available to every template rendered by RenderBody.
	templateFuncs = template.FuncMap{"esc": jsonEscape}
)

// jsonEscape renders v as the *contents* of a JSON string - escaped, without
// the surrounding quotes - so it can be interpolated into a JSON template that
// supplies the quotes itself.
func jsonEscape(v any) (string, error) {
	var s string
	switch t := v.(type) {
	case nil:
		return "", nil
	case string:
		s = t
	case fmt.Stringer:
		s = t.String()
	default:
		s = fmt.Sprint(v)
	}

	encoded, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("failed to escape value for JSON: %w", err)
	}

	// strip the quotes json.Marshal added; the template provides them
	return string(encoded[1 : len(encoded)-1]), nil
}

// DefaultHTTPTimeout bounds a SendToHomerun call that is not already bounded by
// its context. Without any deadline a homerun endpoint that accepts the
// connection and then stalls blocks the caller forever.
const DefaultHTTPTimeout = 30 * time.Second

var (
	// One client (and therefore one connection pool) per TLS configuration.
	// SendToHomerun used to build a fresh http.Transport per call, so every send
	// paid a new TCP and TLS handshake and no connection was ever reused - on a
	// service's hot path that is the dominant cost. There are only two
	// configurations, keyed by insecure.
	httpClientsOnce sync.Once
	httpClients     map[bool]*http.Client

	// customHTTPClient, when set, overrides both.
	customHTTPClient atomic.Pointer[http.Client]
)

func initHTTPClients() {
	httpClients = make(map[bool]*http.Client, 2)
	for _, insecure := range []bool{false, true} {
		// Clone rather than construct: the zero http.Transport has no dial
		// timeout, no proxy support and no HTTP/2, all of which the default
		// transport sets up. The previous implementation used the zero value.
		tr := http.DefaultTransport.(*http.Transport).Clone()
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: insecure} //nolint:gosec // caller-controlled
		httpClients[insecure] = &http.Client{
			Transport: tr,
			Timeout:   DefaultHTTPTimeout,
		}
	}
}

// SetHTTPClient installs the HTTP client SendToHomerun and
// SendToHomerunContext use, for callers that need their own transport,
// timeout, proxy, instrumentation or retry wrapper. Passing nil restores the
// built-in clients.
//
// Note that a custom client is used for both secure and insecure calls; the
// insecure argument is then the caller's responsibility.
//
// It is safe to call at any time and from any goroutine.
func SetHTTPClient(c *http.Client) {
	customHTTPClient.Store(c)
}

func httpClientFor(insecure bool) *http.Client {
	if c := customHTTPClient.Load(); c != nil {
		return c
	}
	httpClientsOnce.Do(initHTTPClients)
	return httpClients[insecure]
}

// SendToHomerun sends a message to the Homerun service with optional insecure
// TLS settings.
//
// This is the context-free form, equivalent to SendToHomerunContext with
// context.Background(). It is still bounded by DefaultHTTPTimeout. Callers that
// want to cancel, or that have a request-scoped context to propagate, should
// use SendToHomerunContext.
func SendToHomerun(destination, token string, renderedBody []byte, insecure bool) ([]byte, *http.Response, error) {
	return SendToHomerunContext(context.Background(), destination, token, renderedBody, insecure)
}

// SendToHomerunContext sends a message to the Homerun service, bounded by ctx.
//
// The returned *http.Response has already had its body read and closed; the
// body bytes are the first return value. See #117 - the response is returned
// for its status code and headers only.
func SendToHomerunContext(
	ctx context.Context,
	destination, token string,
	renderedBody []byte,
	insecure bool,
) ([]byte, *http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, destination, bytes.NewBuffer(renderedBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Auth-Token", token)

	resp, err := httpClientFor(insecure).Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	answer, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %w", err)
	}

	return answer, resp, nil
}

// RenderBody renders templateData with object.
//
// The esc function is registered for JSON escaping - any template that
// interpolates untrusted values into a JSON document must apply it to every
// field, as HomeRunBodyData does. A template that interpolates raw values can
// still produce malformed or injected JSON.
func RenderBody(templateData string, object interface{}) (string, error) {
	tmpl, err := template.New("template").Funcs(templateFuncs).Parse(templateData)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, object); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
