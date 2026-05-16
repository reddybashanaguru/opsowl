// Package response provides the standard {data, meta, error} API envelope.
// Every handler must use these helpers — never write raw JSON directly.
package response

import (
	"encoding/json"
	"net/http"
)

type envelope struct {
	Data  any    `json:"data,omitempty"`
	Meta  *Meta  `json:"meta,omitempty"`
	Error *Error `json:"error,omitempty"`
}

type Meta struct {
	Cursor  string `json:"cursor,omitempty"`
	HasMore bool   `json:"has_more"`
	Total   *int64 `json:"total,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	write(w, status, envelope{Data: data})
}

func Paged(w http.ResponseWriter, status int, data any, cursor string, hasMore bool) {
	write(w, status, envelope{
		Data: data,
		Meta: &Meta{Cursor: cursor, HasMore: hasMore},
	})
}

func Err(w http.ResponseWriter, status int, code, message string) {
	write(w, status, envelope{Error: &Error{Code: code, Message: message}})
}

func ErrDetail(w http.ResponseWriter, status int, code, message string, details any) {
	write(w, status, envelope{Error: &Error{Code: code, Message: message, Details: details}})
}

func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
