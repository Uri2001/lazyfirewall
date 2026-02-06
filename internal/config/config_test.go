package config

import "testing"

func TestStripComment(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "no comment", input: "key = value", want: "key = value"},
		{name: "with comment", input: "key = value # comment", want: "key = value "},
		{name: "quoted hash", input: `key = "value#1"`, want: `key = "value#1"`},
		{name: "escaped quote", input: `key = "val\"ue" # comment`, want: `key = "val\"ue" `},
		{name: "multiple hashes", input: "key = val # comment # more", want: "key = val "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripComment(tt.input)
			if got != tt.want {
				t.Fatalf("stripComment(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
