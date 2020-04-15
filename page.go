package gochrome

import (
	"encoding/base64"
	"fmt"
)

// Goto a url
func (t *Tab) Goto(url string) (PageNavigateReturns, error) {
	return t.PageNavigate(url, "", "", "")
}

// GetResponseBody for a request
func (t *Tab) GetResponseBody(id NetworkRequestId) (string, error) {
	res, err := t.NetworkGetResponseBody(id)
	if err != nil {
		return "", fmt.Errorf("Tab.GetResponseBody: %v", err)
	}
	var body string
	if res.Base64Encoded {
		decoded, err := base64.StdEncoding.DecodeString(res.Body)
		if err != nil {
			return "", fmt.Errorf("Tab.GetResponseBody: %v", err)
		}
		body = string(decoded)
	} else {
		body = res.Body
	}
	return body, nil
}
