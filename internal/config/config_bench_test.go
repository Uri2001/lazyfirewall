package config

import "testing"

func BenchmarkStripComment(b *testing.B) {
	lines := []string{
		`theme = "default"`,
		`theme = "default#value" # comment`,
		`default_permanent = true # keep`,
		`log_level = "info\"quoted\"" # tail`,
		`auto_refresh_interval = 5`,
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = stripComment(lines[i%len(lines)])
	}
}

func BenchmarkParse(b *testing.B) {
	raw := `
[ui]
theme = "default"

[behavior]
default_permanent = true
auto_refresh_interval = 0

[advanced]
log_level = "info"
`

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		cfg := Default()
		if _, err := parse(raw, &cfg); err != nil {
			b.Fatalf("parse() error = %v", err)
		}
	}
}
