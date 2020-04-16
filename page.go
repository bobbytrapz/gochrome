package gochrome

// Goto a url
func (t *Tab) Goto(url string) (PageNavigateReturns, error) {
	return t.PageNavigate(url, "", "", "")
}

// Close a tab; just calls Tab.PageClose
func (t *Tab) Close() {
	t.PageClose()
}
