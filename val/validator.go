package val

import (
	"fmt"
	"net/mail"
	"regexp"
)

var (
	isValidUsername = regexp.MustCompile(`^[a-z0-9_]+$`).MatchString
	isValidFullName = regexp.MustCompile(`^[a-z0-9\\s]+$`).MatchString
)

func ValidateString(value string, minLength int, maxLength int) error {
	n := len(value)
	if n < minLength || n > maxLength {
		return fmt.Errorf("string length must be between %d and %d", minLength, maxLength)
	}
	return nil
}

func ValidateUsername(value string) error {
	err := ValidateString(value, 3, 100)
	if err != nil {
		return err
	}

	if !isValidUsername(value) {
		return fmt.Errorf("must be digits, letters, or underscores")
	}

	return nil
}

func ValidateFullName(value string) error {
	err := ValidateString(value, 3, 100)
	if err != nil {
		return err
	}

	if !isValidFullName(value) {
		return fmt.Errorf("must be digits, letters, or underscores")
	}

	return nil
}

func ValidatePassword(value string) error {
	return ValidateString(value, 6, 100)
}

func ValidateEmail(value string) error {
	err := ValidateString(value, 3, 200)
	if err != nil {
		return err
	}

	if _, err := mail.ParseAddress(value); err != nil {
		return fmt.Errorf("is not a valid email address")
	}

	return nil
}
