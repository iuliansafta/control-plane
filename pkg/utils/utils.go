package utils

import "os"

func IntPtr(i int) *int {
	return &i
}

func StringPtr(s string) *string {
	return &s
}

func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
