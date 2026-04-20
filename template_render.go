package main

import "bytes"

func renderTemplate(t CloudInitTemplate, data RenderData) (string, error) {
	var buf bytes.Buffer
	if err := t.Compiled.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
