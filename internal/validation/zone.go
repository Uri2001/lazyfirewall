package validation

import (
	"fmt"
	"strings"
	"unicode"
)

const maxZoneNameLength = 128

// IsValidZoneName validates that a zone name is safe to use in file paths.
func IsValidZoneName(name string) error {
	if name == "" {
		return fmt.Errorf("zone name cannot be empty")
	}
	if len(name) > maxZoneNameLength {
		return fmt.Errorf("zone name too long (max %d characters)", maxZoneNameLength)
	}
	if strings.Contains(name, "..") {
		return fmt.Errorf("zone name cannot contain '..'")
	}
	if strings.ContainsAny(name, `/\`) {
		return fmt.Errorf("zone name cannot contain path separators")
	}

	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return fmt.Errorf("zone name contains invalid character: %q", r)
	}

	return nil
}
