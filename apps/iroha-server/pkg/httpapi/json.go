package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"unicode"
)

const maxJSONBodyBytes = 1 << 20

var errTrailingJSON = errors.New("request contains multiple JSON values")

type errorResponse struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// boundsResponse is the shared shape for every domain's /bounds endpoint --
// the earliest and latest calendar date with a real record. Both fields are
// omitted when the domain has no data yet.
type boundsResponse struct {
	Min string `json:"min,omitempty"`
	Max string `json:"max,omitempty"`
}

func writeBounds(w http.ResponseWriter, minDate, maxDate string, ok bool) {
	if !ok {
		writeJSON(w, http.StatusOK, boundsResponse{})
		return
	}
	writeJSON(w, http.StatusOK, boundsResponse{Min: minDate, Max: maxDate})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeContractError(w, status, errorCode(message), message)
}

func writeContractError(w http.ResponseWriter, status int, code string, message string) {
	writeJSON(w, status, errorResponse{
		Code:      code,
		Message:   message,
		RequestID: w.Header().Get("X-Request-ID"),
	})
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, destination any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBodyBytes)
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return errTrailingJSON
		}
		return err
	}
	return nil
}

func errorCode(message string) string {
	var builder strings.Builder
	previousUnderscore := false
	for _, r := range strings.ToLower(message) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			previousUnderscore = false
			continue
		}
		if !previousUnderscore {
			builder.WriteByte('_')
			previousUnderscore = true
		}
	}
	return strings.Trim(builder.String(), "_")
}
