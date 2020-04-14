package gochrome

// SetUserAgent override
func (t *Tab) SetUserAgent(ua string) {
	t.NetworkSetUserAgentOverride(ua, "", "")
}

// SetRequestHeaders override
func (t *Tab) SetRequestHeaders(ua string, lang string, platform string) {
	t.NetworkSetUserAgentOverride(ua, lang, platform)
}
