package validation

import (
	"errors"
	"strings"
	"unicode"
)

const maxZoneNameLength = 128

var (
	ErrZoneNameEmpty          = errors.New("zone name cannot be empty")
	ErrZoneNameTooLong        = errors.New("zone name too long (max 128 characters)")
	ErrZoneNameTraversal      = errors.New("zone name cannot contain '..'")
	ErrZoneNamePathSeparator  = errors.New("zone name cannot contain path separators")
	ErrZoneNameInvalidCharSet = errors.New("zone name contains invalid characters")
)

// IsValidZoneName validates that a zone name is safe to use in file paths.
func IsValidZoneName(name string) error {
	if name == "" {
		return ErrZoneNameEmpty
	}
	if len(name) > maxZoneNameLength {
		return ErrZoneNameTooLong
	}
	if strings.Contains(name, "..") {
		return ErrZoneNameTraversal
	}
	if strings.ContainsAny(name, `/\`) {
		return ErrZoneNamePathSeparator
	}

	for _, r := range name {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' || r == '_' {
			continue
		}
		return ErrZoneNameInvalidCharSet
	}

	return nil
}
