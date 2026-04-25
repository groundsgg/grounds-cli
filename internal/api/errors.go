package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Error struct {
	StatusCode int
	Code       string `json:"code,omitempty"`
	Message    string `json:"message,omitempty"`
	Hint       string `json:"hint,omitempty"`
	RetryAfter string
}

func (e *Error) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return fmt.Sprintf("HTTP %d", e.StatusCode)
}

// ExitCode maps an *Error to one of the documented CLI exit codes.
func ExitCode(err error) int {
	e, ok := err.(*Error)
	if !ok {
		return 1
	}
	switch e.StatusCode {
	case http.StatusUnauthorized:
		return 2
	case http.StatusNotFound:
		return 3
	case http.StatusForbidden:
		return 4
	case http.StatusTooManyRequests:
		return 5
	case http.StatusConflict:
		return 6
	case http.StatusBadRequest, http.StatusPreconditionFailed:
		return 11
	default:
		if e.StatusCode >= 500 {
			return 7
		}
		return 1
	}
}

func parseError(resp *http.Response) error {
	raw, _ := io.ReadAll(resp.Body)
	out := &Error{StatusCode: resp.StatusCode, RetryAfter: resp.Header.Get("Retry-After")}
	// Try to decode body; if it's not JSON or has no fields, fall back to status text
	var body struct {
		Error   string `json:"error"`
		Code    string `json:"code"`
		Message string `json:"message"`
		Hint    string `json:"hint"`
	}
	if err := json.Unmarshal(raw, &body); err == nil {
		out.Code = firstNonEmpty(body.Code, body.Error)
		out.Message = firstNonEmpty(body.Message, body.Error)
		out.Hint = body.Hint
	}
	if out.Message == "" {
		out.Message = http.StatusText(resp.StatusCode)
	}
	return out
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
