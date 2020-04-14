package gochrome

// Goto a url
func (t *Tab) Goto(url string) (PageNavigateReturns, error) {
	return t.PageNavigate(url, "", "", "")
}
