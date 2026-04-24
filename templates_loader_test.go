package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadCloudInitTemplatesEmptyDirectory(t *testing.T) {
	dir := t.TempDir()
	_, err := LoadCloudInitTemplates(dir)
	if err == nil {
		t.Fatal("expected error for empty template directory")
	}
	if !strings.Contains(err.Error(), "no .yaml templates") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadCloudInitTemplatesInvalidTemplate(t *testing.T) {
	dir := t.TempDir()
	bad := "#cloud-config\nhostname: {{.Hostname\n"
	if err := os.WriteFile(filepath.Join(dir, "bad.yaml"), []byte(bad), 0o644); err != nil {
		t.Fatalf("write bad template: %v", err)
	}

	_, err := LoadCloudInitTemplates(dir)
	if err == nil {
		t.Fatal("expected parse error for invalid template")
	}
	if !strings.Contains(err.Error(), "parse template") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadCloudInitTemplatesSuccess(t *testing.T) {
	dir := t.TempDir()
	good := "#cloud-config\nhostname: {{.Hostname}}\n"
	if err := os.WriteFile(filepath.Join(dir, "example.yaml"), []byte(good), 0o644); err != nil {
		t.Fatalf("write good template: %v", err)
	}

	templates, err := LoadCloudInitTemplates(dir)
	if err != nil {
		t.Fatalf("LoadCloudInitTemplates error: %v", err)
	}
	if len(templates) != 1 {
		t.Fatalf("len(templates)=%d want 1", len(templates))
	}
	tpl, ok := templates["example"]
	if !ok {
		t.Fatalf("expected template named example")
	}
	if tpl.Filename != "example.yaml" {
		t.Fatalf("filename=%q want example.yaml", tpl.Filename)
	}
}
