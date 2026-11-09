// Package httputil provides HTTP response serialization and error formatting utilities.
package httputil

import (
	"encoding/json"
	"net/http"
)

// Serializes the payload into a standard JSON response with the designated HTTP status code.
func WriteJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// Emits a standardized JSON error envelope containing the specified message.
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{
		"error": message,
	})
}
