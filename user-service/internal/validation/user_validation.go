package validation

import (
	"errors"
	"regexp"
	"strings"
	"unicode"
)

var (
	emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
)

// ValidationError represents a validation failure
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (e ValidationError) Error() string {
	return e.Message
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return ValidationError{Field: "email", Message: "email is required"}
	}
	if len(email) > 254 {
		return ValidationError{Field: "email", Message: "email must be less than 254 characters"}
	}
	if !emailRegex.MatchString(email) {
		return ValidationError{Field: "email", Message: "invalid email format"}
	}
	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < 8 {
		return ValidationError{Field: "password", Message: "password must be at least 8 characters"}
	}
	if len(password) > 128 {
		return ValidationError{Field: "password", Message: "password must be less than 128 characters"}
	}

	var hasUpper, hasLower, hasNumber bool
	for _, c := range password {
		switch {
		case unicode.IsUpper(c):
			hasUpper = true
		case unicode.IsLower(c):
			hasLower = true
		case unicode.IsNumber(c):
			hasNumber = true
		}
	}

	if !hasUpper || !hasLower || !hasNumber {
		return ValidationError{Field: "password", Message: "password must contain at least one uppercase, one lowercase, and one number"}
	}
	return nil
}

// ValidateUsername validates username format
func ValidateUsername(username string) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return ValidationError{Field: "username", Message: "username is required"}
	}
	if len(username) < 3 {
		return ValidationError{Field: "username", Message: "username must be at least 3 characters"}
	}
	if len(username) > 30 {
		return ValidationError{Field: "username", Message: "username must be less than 30 characters"}
	}

	// Only alphanumeric and underscores
	for _, c := range username {
		if !unicode.IsLetter(c) && !unicode.IsNumber(c) && c != '_' {
			return ValidationError{Field: "username", Message: "username can only contain letters, numbers, and underscores"}
		}
	}
	return nil
}

// ValidateFullName validates full name
func ValidateFullName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ValidationError{Field: "full_name", Message: "full name is required"}
	}
	if len(name) > 100 {
		return ValidationError{Field: "full_name", Message: "full name must be less than 100 characters"}
	}
	return nil
}

// ValidateRegistration validates all registration fields
func ValidateRegistration(email, username, password, fullName string) error {
	if err := ValidateEmail(email); err != nil {
		return err
	}
	if err := ValidateUsername(username); err != nil {
		return err
	}
	if err := ValidatePassword(password); err != nil {
		return err
	}
	if err := ValidateFullName(fullName); err != nil {
		return err
	}
	return nil
}

// ValidatePasswordChange validates password change request
func ValidatePasswordChange(currentPassword, newPassword string) error {
	if currentPassword == "" {
		return errors.New("current password is required")
	}
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	if currentPassword == newPassword {
		return errors.New("new password must be different from current password")
	}
	return nil
}
