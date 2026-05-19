package httptransport

import (
	"net/http"
	"time"

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
	writeJSON(w, http.StatusCreated, authResponse{Token: token, User: newUserResponse(user)})
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
	writeJSON(w, http.StatusOK, authResponse{Token: token, User: newUserResponse(user)})
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, newUserResponse(currentUser(r)))
}

type authResponse struct {
	Token string       `json:"token"`
	User  userResponse `json:"user"`
}

type userResponse struct {
	ID        int64     `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

func newUserResponse(user model.User) userResponse {
	return userResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
