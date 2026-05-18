package httptransport

import (
	"net/http"

	"workout-app/backend/internal/application"
)

type Handler struct {
	authService    *application.AuthService
	catalogService *application.CatalogService
	workoutService *application.WorkoutService
	uploadDir      string
}

func NewHandler(
	authService *application.AuthService,
	catalogService *application.CatalogService,
	workoutService *application.WorkoutService,
	uploadDir string,
) *Handler {
	return &Handler{
		authService:    authService,
		catalogService: catalogService,
		workoutService: workoutService,
		uploadDir:      uploadDir,
	}
}

func (h *Handler) Router() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", h.health)
	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(h.uploadDir))))

	mux.HandleFunc("POST /api/v1/auth/register", h.register)
	mux.HandleFunc("POST /api/v1/auth/login", h.login)
	mux.Handle("GET /api/v1/auth/me", h.auth(http.HandlerFunc(h.me)))

	mux.HandleFunc("GET /api/v1/categories", h.listCategories)
	mux.Handle("POST /api/v1/admin/categories", h.authAdmin(http.HandlerFunc(h.createCategory)))
	mux.Handle("PATCH /api/v1/admin/categories/{id}", h.authAdmin(http.HandlerFunc(h.updateCategory)))

	mux.HandleFunc("GET /api/v1/exercises", h.listExercises)
	mux.HandleFunc("GET /api/v1/exercises/{id}", h.getExercise)
	mux.Handle("POST /api/v1/admin/exercises", h.authAdmin(http.HandlerFunc(h.createExercise)))
	mux.Handle("PATCH /api/v1/admin/exercises/{id}", h.authAdmin(http.HandlerFunc(h.updateExercise)))
	mux.Handle("POST /api/v1/admin/exercises/{id}/media", h.authAdmin(http.HandlerFunc(h.uploadExerciseMedia)))
	mux.Handle("DELETE /api/v1/admin/exercise-media/{id}", h.authAdmin(http.HandlerFunc(h.deleteExerciseMedia)))

	mux.Handle("POST /api/v1/workouts", h.auth(http.HandlerFunc(h.createWorkout)))
	mux.Handle("GET /api/v1/workouts/active", h.auth(http.HandlerFunc(h.activeWorkout)))
	mux.Handle("GET /api/v1/workouts", h.auth(http.HandlerFunc(h.listWorkouts)))
	mux.Handle("GET /api/v1/workouts/{id}", h.auth(http.HandlerFunc(h.getWorkout)))
	mux.Handle("PATCH /api/v1/workouts/{id}", h.auth(http.HandlerFunc(h.updateWorkout)))
	mux.Handle("POST /api/v1/workouts/{id}/finish", h.auth(http.HandlerFunc(h.finishWorkout)))
	mux.Handle("POST /api/v1/workouts/{id}/exercises", h.auth(http.HandlerFunc(h.addWorkoutExercise)))
	mux.Handle("PATCH /api/v1/workout-exercises/{id}", h.auth(http.HandlerFunc(h.updateWorkoutExercise)))
	mux.Handle("POST /api/v1/workout-exercises/{id}/sets", h.auth(http.HandlerFunc(h.createSet)))
	mux.Handle("PATCH /api/v1/workout-sets/{id}", h.auth(http.HandlerFunc(h.updateSet)))
	mux.Handle("DELETE /api/v1/workout-sets/{id}", h.auth(http.HandlerFunc(h.deleteSet)))

	mux.Handle("POST /api/v1/workouts/{id}/share-links", h.auth(http.HandlerFunc(h.createShareLink)))
	mux.Handle("DELETE /api/v1/share-links/{id}", h.auth(http.HandlerFunc(h.disableShareLink)))
	mux.HandleFunc("GET /api/v1/public/workouts/{token}", h.publicWorkout)
	mux.Handle("GET /api/v1/workouts/{id}/export.csv", h.auth(http.HandlerFunc(h.exportCSV)))
	mux.Handle("GET /api/v1/workouts/{id}/export.json", h.auth(http.HandlerFunc(h.exportJSON)))

	return cors(mux)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
