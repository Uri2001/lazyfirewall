package validation

import "testing"

func BenchmarkIsValidZoneNameValid(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := IsValidZoneName("public_zone-1"); err != nil {
			b.Fatalf("IsValidZoneName() error = %v", err)
		}
	}
}

func BenchmarkIsValidZoneNameInvalid(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := IsValidZoneName("../etc/passwd"); err == nil {
			b.Fatalf("IsValidZoneName() expected error for invalid zone")
		}
	}
}
