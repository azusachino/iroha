package httpapi

import (
	"net/http"
	"strconv"
)

const (
	defaultPageLimit = 50
	maxPageLimit     = 100
)

func parsePageLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	value := r.URL.Query().Get("limit")
	if value == "" {
		return defaultPageLimit, true
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 || limit > maxPageLimit {
		writeError(w, http.StatusBadRequest, "invalid limit")
		return 0, false
	}
	return limit, true
}
