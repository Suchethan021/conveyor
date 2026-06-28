// Package httpx holds small shared helpers for writing JSON HTTP responses
// with a consistent envelope and error shape.
package httpx

import (
	"encoding/json"
	"net/http"
)

// JSON writes v as a JSON response with the given status code.
func JSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Error writes a structured error: {"error": {"code", "message"}}.
func Error(w http.ResponseWriter, code int, errCode, msg string) {
	JSON(w, code, errorBody{Error: errorDetail{Code: errCode, Message: msg}})
}
