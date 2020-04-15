package gochrome

import (
	"encoding/base64"
	"fmt"
)

// SetUserAgent override
func (t *Tab) SetUserAgent(ua string) {
	t.NetworkSetUserAgentOverride(ua, "", "")
}

// SetRequestHeaders override
func (t *Tab) SetRequestHeaders(ua string, lang string, platform string) {
	t.NetworkSetUserAgentOverride(ua, lang, platform)
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
