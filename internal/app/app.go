package app

import (
	"fmt"
	"os"

	"workout-app/backend/internal/application"
	"workout-app/backend/internal/config"
	"workout-app/backend/internal/repository/jsonstore"
	httptransport "workout-app/backend/internal/transport/http"
)

type App struct {
	config *config.Config
	server *Server
}

func New(cfg *config.Config) (*App, error) {
	if err := os.MkdirAll(cfg.Storage.UploadDir, 0o755); err != nil {
		return nil, fmt.Errorf("создание папки uploads: %w", err)
	}

	repo, err := jsonstore.New(cfg.Storage.DataPath)
	if err != nil {
		return nil, fmt.Errorf("создание json repository: %w", err)
	}

	authService := application.NewAuthService(repo)
	catalogService := application.NewCatalogService(repo)
	workoutService := application.NewWorkoutService(repo)
	handler := httptransport.NewHandler(authService, catalogService, workoutService, cfg.Storage.UploadDir)

	return &App{
		config: cfg,
		server: NewServer(cfg, handler.Router()),
	}, nil
}

func (a *App) Run() error {
	return a.server.Run()
}
