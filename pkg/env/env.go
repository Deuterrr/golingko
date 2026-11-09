// Package env provides type-safe environment variable parsing and configuration loading.
package env

import (
	"os"
	"strconv"
	"strings"
)

type EnvType interface {
	~string | ~int | ~int64 | ~bool | ~float64
}

// Retrieves the value of the environment variable named by key, or returns defaultVal if unset or empty.
func Get[T EnvType](key string, defaultVal T) T {
	val, ok := os.LookupEnv(key)
	if !ok || val == "" {
		return defaultVal
	}

	var anyVal any = defaultVal
	switch anyVal.(type) {
	case string:
		return any(val).(T)
	case int:
		if intVal, err := strconv.Atoi(val); err == nil {
			return any(intVal).(T)
		}
	case int64:
		if int64Val, err := strconv.ParseInt(val, 10, 64); err == nil {
			return any(int64Val).(T)
		}
	case bool:
		if boolVal, err := strconv.ParseBool(val); err == nil {
			return any(boolVal).(T)
		}
	case float64:
		if float64Val, err := strconv.ParseFloat(val, 64); err == nil {
			return any(float64Val).(T)
		}
	}

	return defaultVal
}

// Reads and applies key-value pairs from a dotenv file into the process environment without overwriting existing keys.
func Load(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)

		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
}
