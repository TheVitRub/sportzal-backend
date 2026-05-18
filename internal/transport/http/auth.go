package httptransport

import (
	"net/http"

	"workout-app/backend/internal/domain/model"
)

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}

	token, user, err := h.authService.Register(r.Context(), req.Email, req.Name, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: user})
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}

	token, user, err := h.authService.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: user})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

type authResponse struct {
	Token string     `json:"token"`
	User  model.User `json:"user"`
}
