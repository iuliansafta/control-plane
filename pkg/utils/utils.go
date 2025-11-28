package utils

import "os"

// IntPtr returns a pointer to the given integer
func IntPtr(i int) *int {
	return &i
}

// StringPtr returns a pointer to the given string
func StringPtr(s string) *string {
	return &s
}

// GetEnv returns the value of the environment variable with the given key,
// or the default value if the key is not set
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
