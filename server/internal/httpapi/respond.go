package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"localtask/internal/project"
)

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type errorResponse struct {
	Error  string   `json:"error"`
	Issues []string `json:"issues,omitempty"`
}

func writeError(w http.ResponseWriter, status int, err error) {
	resp := errorResponse{Error: err.Error()}
	var verr *project.ValidationError
	if errors.As(err, &verr) {
		resp.Issues = verr.Issues
	}
	writeJSON(w, status, resp)
}

// writeDomainError maps an error from the project package to the
// appropriate HTTP status: not-found errors to 404, an external edit
// conflict to 409, a failed validation to 422, and anything else
// (malformed input) to 400.
func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, project.ErrTicketNotFound):
		writeError(w, http.StatusNotFound, err)
	case errors.Is(err, project.ErrStaleTaskboard):
		writeError(w, http.StatusConflict, err)
	default:
		var verr *project.ValidationError
		if errors.As(err, &verr) {
			writeError(w, http.StatusUnprocessableEntity, err)
			return
		}
		writeError(w, http.StatusBadRequest, err)
	}
}
