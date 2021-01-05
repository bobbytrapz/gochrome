package gochrome

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
)

// Screenshot captures page as png
// uses Page.captureScreenshot
func (t *Tab) Screenshot(saveAs string) error {
	res, err := t.PageCaptureScreenshot("png", 0, nil, true)
	if err != nil {
		return fmt.Errorf("Tab.Screenshot: %w", err)
	}

	// decode
	img, err := base64.StdEncoding.DecodeString(res.Data)
	if err != nil {
		Log("Tab.Screenshot: %s", err)
		return err
	}

	// save
	if err := ioutil.WriteFile(saveAs, img, 0644); err != nil {
		Log("Tab.Screenshot: %s", err)
		return err
	}

	return nil
}

// Snapshot page in mhtml format
// uses Page.captureSnapshot
func (t *Tab) Snapshot(saveAs string) error {
	res, err := t.PageCaptureSnapshot("mhtml")
	if err != nil {
		return fmt.Errorf("Tab.Snapshot: %w", err)
	}

	// save
	if err := ioutil.WriteFile(saveAs, []byte(res.Data), 0644); err != nil {
		Log("Tab.Snapshot: %s", err)
	} else {
		Log("Tab.Snapshot: %s", saveAs)
	}

	return nil
}
