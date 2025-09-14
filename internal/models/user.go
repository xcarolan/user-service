package models

import (
	"fmt"
	"strconv"
	"strings"
)

// User represents a user in the system
type User struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Validate checks if the user data is valid
func (u *User) Validate() error {
	if u.Name == "" {
		return fmt.Errorf("name cannot be empty")
	}
	if u.Email == "" {
		return fmt.Errorf("email cannot be empty")
	}
	if !strings.Contains(u.Email, "@") {
		return fmt.Errorf("email must contain @")
	}
	return nil
}

// ParseUserID converts a string ID to an integer
func ParseUserID(idStr string) (int, error) {
	if idStr == "" {
		return 0, fmt.Errorf("id parameter is missing")
	}

	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("id parameter is invalid")
	}

	return id, nil
}
