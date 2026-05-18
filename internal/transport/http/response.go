package httptransport

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"workout-app/backend/internal/domain/apperrors"
)

func decodeJSON(r *http.Request, dst interface{}) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	decoder.DisallowUnknownFields()
	return decoder.Decode(dst)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	_ = encoder.Encode(payload)
}

func writeError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	message := "внутренняя ошибка сервера"

	var appErr *apperrors.Error
	if errors.As(err, &appErr) {
		message = appErr.Message
		switch {
		case errors.Is(appErr.Kind, apperrors.ErrBadRequest):
			status = http.StatusBadRequest
		case errors.Is(appErr.Kind, apperrors.ErrUnauthorized):
			status = http.StatusUnauthorized
		case errors.Is(appErr.Kind, apperrors.ErrForbidden):
			status = http.StatusForbidden
		case errors.Is(appErr.Kind, apperrors.ErrNotFound):
			status = http.StatusNotFound
		case errors.Is(appErr.Kind, apperrors.ErrConflict):
			status = http.StatusConflict
		}
	}

	writeJSON(w, status, map[string]string{"error": message})
}

func writeBadRequest(w http.ResponseWriter, message string) {
	writeError(w, apperrors.BadRequest(message))
}

func pathID(r *http.Request, name string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(name))
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, apperrors.BadRequest("некорректный id")
	}
	return id, nil
}
