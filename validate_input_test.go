package main

import "testing"

func TestValidateInputFailures(t *testing.T) {
	templates := map[string]CloudInitTemplate{"example": {Name: "example"}}
	boxes := DefaultBoxTypes()

	tests := []struct {
		name    string
		args    []any
		wantErr bool
	}{
		{name: "unknown template", args: []any{"missing", "nuc-dual-nic", "edge-1", "192.168.1.10", "24", "192.168.1.1", []string{"1.1.1.1"}}, wantErr: true},
		{name: "unknown box", args: []any{"example", "missing", "edge-1", "192.168.1.10", "24", "192.168.1.1", []string{"1.1.1.1"}}, wantErr: true},
		{name: "invalid hostname", args: []any{"example", "nuc-dual-nic", "-bad", "192.168.1.10", "24", "192.168.1.1", []string{"1.1.1.1"}}, wantErr: true},
		{name: "invalid static ip", args: []any{"example", "nuc-dual-nic", "edge-1", "nope", "24", "192.168.1.1", []string{"1.1.1.1"}}, wantErr: true},
		{name: "invalid cidr", args: []any{"example", "nuc-dual-nic", "edge-1", "192.168.1.10", "64", "192.168.1.1", []string{"1.1.1.1"}}, wantErr: true},
		{name: "invalid gateway", args: []any{"example", "nuc-dual-nic", "edge-1", "192.168.1.10", "24", "bad", []string{"1.1.1.1"}}, wantErr: true},
		{name: "invalid dns", args: []any{"example", "nuc-dual-nic", "edge-1", "192.168.1.10", "24", "192.168.1.1", []string{"bad"}}, wantErr: true},
		{name: "valid minimal", args: []any{"example", "nuc-dual-nic", "edge-1", "192.168.1.10", "24", "", []string{"1.1.1.1", "8.8.8.8"}}, wantErr: false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateInput(
				tc.args[0].(string),
				tc.args[1].(string),
				tc.args[2].(string),
				tc.args[3].(string),
				tc.args[4].(string),
				tc.args[5].(string),
				tc.args[6].([]string),
				templates,
				boxes,
			)
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
