package gochrome

func (t *Tab) Evaluate(js string) (RuntimeEvaluateReturns, error) {
	r, err := t.RuntimeEvaluate(js, "", false, false, 0, false, false, true, true, false, 0, false, true)

	if err != nil {
		Log("Tab.Evaluate: error: %q", err)
	}

	if r.Result != nil {
		Log("Tab.Evaluate: type: %q value: %q", r.Result["type"], r.Result["value"])
	}

	return r, err
}
