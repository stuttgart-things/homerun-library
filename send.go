/*
Copyright © 2025 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"text/template"
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

// SendToHomerun sends a message to the Homerun service with optional insecure TLS settings.
func SendToHomerun(destination, token string, renderedBody []byte, insecure bool) ([]byte, *http.Response, error) {
	req, err := http.NewRequest(http.MethodPost, destination, bytes.NewBuffer(renderedBody))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-Auth-Token", token)

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: insecure}, //nolint:gosec // caller-controlled
	}
	client := &http.Client{Transport: tr}

	resp, err := client.Do(req)
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
