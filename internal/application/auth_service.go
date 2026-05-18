package application

import (
	"context"
	"strings"

	"workout-app/backend/internal/domain/apperrors"
	"workout-app/backend/internal/domain/interfaces"
	"workout-app/backend/internal/domain/model"
	"workout-app/backend/pkg/security"
)

type AuthService struct {
	repo interfaces.StateRepository
}

func NewAuthService(repo interfaces.StateRepository) *AuthService {
	return &AuthService{repo: repo}
}

func (s *AuthService) Register(ctx context.Context, email, name, password string) (string, model.User, error) {
	email = normalizeEmail(email)
	if email == "" || !strings.Contains(email, "@") {
		return "", model.User{}, apperrors.BadRequest("укажите корректный e-mail")
	}
	name = strings.TrimSpace(name)
	if name == "" {
		name = email
	}

	passwordHash, err := security.HashPassword(password)
	if err != nil {
		return "", model.User{}, apperrors.BadRequest(err.Error())
	}
	token, err := security.RandomToken(32)
	if err != nil {
		return "", model.User{}, err
	}

	var created model.User
	err = s.repo.Update(ctx, func(state *model.State) error {
		for _, user := range state.Users {
			if user.Email == email {
				return apperrors.Conflict("пользователь с таким e-mail уже есть")
			}
		}

		role := model.RoleUser
		if len(state.Users) == 0 {
			role = model.RoleAdmin
		}
		now := nowUTC()
		created = model.User{
			ID:           state.NextUserID,
			Email:        email,
			Name:         name,
			PasswordHash: passwordHash,
			Role:         role,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		state.NextUserID++
		state.Users = append(state.Users, created)
		if state.Sessions == nil {
			state.Sessions = map[string]int64{}
		}
		state.Sessions[token] = created.ID
		return nil
	})

	return token, created, err
}

func (s *AuthService) Login(ctx context.Context, email, password string) (string, model.User, error) {
	email = normalizeEmail(email)
	token, err := security.RandomToken(32)
	if err != nil {
		return "", model.User{}, err
	}

	var loggedIn model.User
	err = s.repo.Update(ctx, func(state *model.State) error {
		for _, user := range state.Users {
			if user.Email == email && security.VerifyPassword(password, user.PasswordHash) {
				if state.Sessions == nil {
					state.Sessions = map[string]int64{}
				}
				state.Sessions[token] = user.ID
				loggedIn = user
				return nil
			}
		}
		return apperrors.Unauthorized("неверный e-mail или пароль")
	})

	return token, loggedIn, err
}

func (s *AuthService) UserByToken(ctx context.Context, token string) (model.User, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return model.User{}, apperrors.Unauthorized("нужна авторизация")
	}

	var current model.User
	err := s.repo.View(ctx, func(state model.State) error {
		userID, ok := state.Sessions[token]
		if !ok {
			return apperrors.Unauthorized("сессия не найдена")
		}
		user, ok := findUser(&state, userID)
		if !ok {
			return apperrors.Unauthorized("сессия не найдена")
		}
		current = user
		return nil
	})
	return current, err
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}
