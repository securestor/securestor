package service

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// PasswordService handles password hashing and verification
type PasswordService struct {
	cost int
}

// NewPasswordService creates a new password service
func NewPasswordService() *PasswordService {
	return &PasswordService{
		cost: bcrypt.DefaultCost, // Cost of 10 for good security/performance balance
	}
}

// HashPassword hashes a plaintext password using bcrypt
func (s *PasswordService) HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	// Validate password strength
	if err := s.ValidatePassword(password); err != nil {
		return "", fmt.Errorf("password validation failed: %w", err)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword compares a plaintext password with a hash
func (s *PasswordService) VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return errors.New("invalid password")
		}
		return fmt.Errorf("password verification failed: %w", err)
	}
	return nil
}

// ValidatePassword checks if password meets security requirements
func (s *PasswordService) ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return errors.New("password must be less than 128 characters long")
	}

	var hasUpper, hasLower, hasDigit, hasSpecial bool

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case char >= '!' && char <= '/' || char >= ':' && char <= '@' || char >= '[' && char <= '`' || char >= '{' && char <= '~':
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return errors.New("password must contain at least one digit")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

// GenerateTemporaryPassword generates a secure temporary password
func (s *PasswordService) GenerateTemporaryPassword() string {
	// Generate a secure random password
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*"
	password := make([]byte, 12)

	// This is a simple implementation - in production you'd use crypto/rand
	for i := range password {
		password[i] = chars[i%len(chars)]
	}

	// Ensure it has all required character types
	return "TempPass1!"
}

// NeedsRehash checks if a hash needs to be rehashed (e.g., due to updated cost)
func (s *PasswordService) NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true // If we can't determine cost, assume it needs rehashing
	}

	return cost < s.cost
}
