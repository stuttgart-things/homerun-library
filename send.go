/*
Copyright © 2025 Patrick Hermann patrick.hermann@sva.de
*/

package homerun

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"text/template"
)

var (
	contentType     = "application/json"
	HomeRunBodyData = `{
		"Title": "{{ .Title }}",
		"Message": "{{ .Message }}",
		"Severity": "{{ .Severity }}",
		"Author": "{{ .Author }}",
		"Timestamp": "{{ .Timestamp }}",
		"System": "{{ .System }}",
		"Tags": "{{ .Tags }}",
		"AssigneeAddress": "{{ .AssigneeAddress }}",
		"AssigneeName": "{{ .AssigneeName }}",
		"Artifacts": "{{ .Artifacts }}",
		"Url": "{{ .Url }}"
	}`
)

// SendToHomerun sends a message to the Homerun service with optional insecure TLS settings.
func SendToHomerun(destination, token string, renderedBody []byte, insecure bool) ([]byte, *http.Response, error) {
	req, err := http.NewRequest("POST", destination, bytes.NewBuffer(renderedBody))
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
	defer resp.Body.Close()

	answer, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, resp, fmt.Errorf("failed to read response body: %w", err)
	}

	return answer, resp, nil
}

func RenderBody(templateData string, object interface{}) string {

	tmpl, err := template.New("template").Parse(templateData)
	if err != nil {
		panic(err)
	}

	var buf bytes.Buffer

	err = tmpl.Execute(&buf, object)

	if err != nil {
		fmt.Println(err)
	}

	return buf.String()

}
