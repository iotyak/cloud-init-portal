package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	txttmpl "text/template"
)

type TextTemplate = *txttmpl.Template

func LoadCloudInitTemplates(dir string) (map[string]CloudInitTemplate, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read template dir: %w", err)
	}

	out := make(map[string]CloudInitTemplate)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if filepath.Ext(e.Name()) != ".yaml" {
			continue
		}

		full := filepath.Join(dir, e.Name())
		b, err := os.ReadFile(full)
		if err != nil {
			return nil, fmt.Errorf("read template %s: %w", e.Name(), err)
		}

		name := strings.TrimSuffix(e.Name(), ".yaml")
		tmpl, err := txttmpl.New(e.Name()).Option("missingkey=error").Parse(string(b))
		if err != nil {
			return nil, fmt.Errorf("parse template %s: %w", e.Name(), err)
		}

		out[name] = CloudInitTemplate{
			Name:     name,
			Filename: e.Name(),
			Raw:      string(b),
			Compiled: tmpl,
		}
	}

	if len(out) == 0 {
		return nil, errors.New("no .yaml templates found in ./templates")
	}

	return out, nil
}

func TemplateNames(m map[string]CloudInitTemplate) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func DefaultBoxTypes() map[string]BoxType {
	return map[string]BoxType{
		"nuc-dual-nic": {
			Name:                "nuc-dual-nic",
			BootstrapInterface:  "enp1s0",
			ProductionInterface: "enp2s0",
		},
		"supermicro-dual-nic": {
			Name:                "supermicro-dual-nic",
			BootstrapInterface:  "eno1",
			ProductionInterface: "eno2",
		},
	}
}

func BoxTypeNames(m map[string]BoxType) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
