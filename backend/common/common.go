package common

import (
	"os"
	"strconv"
)

func GetString(key, fallback string) string {
	val := os.Getenv(key); if val == "" {
		return fallback
	}
	return val
}

func GetInt(key string, fallback int) int {
	val := os.Getenv(key); if val == "" {
		return fallback
	} 
	parsedVal, err := strconv.Atoi(val);
	if err != nil {
		return fallback
	}
	return parsedVal
}
