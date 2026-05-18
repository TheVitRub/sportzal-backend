package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	roleAdmin = "admin"
	roleUser  = "user"

	statusActive    = "active"
	statusCompleted = "completed"
)

type MetricField struct {
	Key       string   `json:"key"`
	Label     string   `json:"label"`
	Unit      string   `json:"unit"`
	ValueType string   `json:"valueType"`
	Required  bool     `json:"required"`
	Min       *float64 `json:"min,omitempty"`
	Max       *float64 `json:"max,omitempty"`
	Step      *float64 `json:"step,omitempty"`
}

type MetricSchema struct {
	Type   string            `json:"type"`
	Fields []MetricField     `json:"fields"`
	Target map[string]string `json:"target,omitempty"`
}

type User struct {
	ID           int64     `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ExerciseCategory struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sortOrder"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ExerciseTemplate struct {
	ID           int64        `json:"id"`
	CategoryID   int64        `json:"categoryId"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	MetricSchema MetricSchema `json:"metricSchema"`
	IsActive     bool         `json:"isActive"`
	CreatedBy    int64        `json:"createdBy"`
	CreatedAt    time.Time    `json:"createdAt"`
	UpdatedAt    time.Time    `json:"updatedAt"`
}

type ExerciseMedia struct {
	ID         int64     `json:"id"`
	ExerciseID int64     `json:"exerciseId"`
	MediaType  string    `json:"mediaType"`
	FileURL    string    `json:"fileUrl"`
	MimeType   string    `json:"mimeType"`
	SizeBytes  int64     `json:"sizeBytes"`
	CreatedAt  time.Time `json:"createdAt"`
}

type ExerciseView struct {
	ExerciseTemplate
	CategoryName string          `json:"categoryName"`
	Media        []ExerciseMedia `json:"media"`
}

type ExerciseSnapshot struct {
	ExerciseID    int64           `json:"exerciseId"`
	Title         string          `json:"title"`
	Description   string          `json:"description"`
	CategoryName  string          `json:"categoryName"`
	MetricSchema  MetricSchema    `json:"metricSchema"`
	Media         []ExerciseMedia `json:"media"`
	SnapshotTaken time.Time       `json:"snapshotTaken"`
}

type Workout struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"userId"`
	Title      string     `json:"title"`
	Status     string     `json:"status"`
	StartedAt  time.Time  `json:"startedAt"`
	FinishedAt *time.Time `json:"finishedAt,omitempty"`
	Notes      string     `json:"notes"`
	CreatedAt  time.Time  `json:"createdAt"`
	UpdatedAt  time.Time  `json:"updatedAt"`
}

type WorkoutExercise struct {
	ID               int64            `json:"id"`
	WorkoutID        int64            `json:"workoutId"`
	ExerciseID       int64            `json:"exerciseTemplateId"`
	ExerciseSnapshot ExerciseSnapshot `json:"exerciseSnapshot"`
	Position         int              `json:"position"`
	Notes            string           `json:"notes"`
	CreatedAt        time.Time        `json:"createdAt"`
	UpdatedAt        time.Time        `json:"updatedAt"`
}

type WorkoutSet struct {
	ID                int64                  `json:"id"`
	WorkoutExerciseID int64                  `json:"workoutExerciseId"`
	ClientID          string                 `json:"clientId"`
	SetIndex          int                    `json:"setIndex"`
	MetricValues      map[string]interface{} `json:"metricValues"`
	Notes             string                 `json:"notes"`
	CompletedAt       time.Time              `json:"completedAt"`
	CreatedAt         time.Time              `json:"createdAt"`
	UpdatedAt         time.Time              `json:"updatedAt"`
}

type WorkoutExerciseDetail struct {
	WorkoutExercise
	Sets []WorkoutSet `json:"sets"`
}

type WorkoutDetail struct {
	Workout
	Exercises []WorkoutExerciseDetail `json:"exercises"`
}

type WorkoutSummary struct {
	Workout
	ExerciseCount int `json:"exerciseCount"`
	SetCount      int `json:"setCount"`
}

type ShareLink struct {
	ID        int64      `json:"id"`
	WorkoutID int64      `json:"workoutId"`
	Token     string     `json:"token,omitempty"`
	TokenHash string     `json:"-"`
	IsActive  bool       `json:"isActive"`
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
}

type database struct {
	NextUserID            int64              `json:"nextUserId"`
	NextCategoryID        int64              `json:"nextCategoryId"`
	NextExerciseID        int64              `json:"nextExerciseId"`
	NextMediaID           int64              `json:"nextMediaId"`
	NextWorkoutID         int64              `json:"nextWorkoutId"`
	NextWorkoutExerciseID int64              `json:"nextWorkoutExerciseId"`
	NextSetID             int64              `json:"nextSetId"`
	NextShareLinkID       int64              `json:"nextShareLinkId"`
	Sessions              map[string]int64   `json:"sessions"`
	Users                 []User             `json:"users"`
	Categories            []ExerciseCategory `json:"categories"`
	Exercises             []ExerciseTemplate `json:"exercises"`
	Media                 []ExerciseMedia    `json:"media"`
	Workouts              []Workout          `json:"workouts"`
	WorkoutExercises      []WorkoutExercise  `json:"workoutExercises"`
	Sets                  []WorkoutSet       `json:"sets"`
	ShareLinks            []ShareLink        `json:"shareLinks"`
}

type store struct {
	mu   sync.Mutex
	path string
	db   database
}

type app struct {
	store     *store
	uploadDir string
}

type contextKey string

const userContextKey contextKey = "user"

func main() {
	dataPath := envOr("APP_DATA_PATH", filepath.Join("data", "app.json"))
	uploadDir := envOr("UPLOAD_DIR", "uploads")
	port := envOr("PORT", "8080")

	st, err := openStore(dataPath)
	if err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Fatal(err)
	}

	a := &app{store: st, uploadDir: uploadDir}
	mux := http.NewServeMux()
	a.routes(mux)

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      cors(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	log.Printf("Workout API listening on http://localhost:%s", port)
	log.Fatal(server.ListenAndServe())
}

func envOr(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (a *app) routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.Handle("GET /uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir(a.uploadDir))))

	mux.HandleFunc("POST /api/v1/auth/register", a.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", a.handleLogin)
	mux.Handle("GET /api/v1/auth/me", a.auth(http.HandlerFunc(a.handleMe)))

	mux.HandleFunc("GET /api/v1/categories", a.handleListCategories)
	mux.Handle("POST /api/v1/admin/categories", a.authAdmin(http.HandlerFunc(a.handleCreateCategory)))
	mux.Handle("PATCH /api/v1/admin/categories/{id}", a.authAdmin(http.HandlerFunc(a.handleUpdateCategory)))

	mux.HandleFunc("GET /api/v1/exercises", a.handleListExercises)
	mux.HandleFunc("GET /api/v1/exercises/{id}", a.handleGetExercise)
	mux.Handle("POST /api/v1/admin/exercises", a.authAdmin(http.HandlerFunc(a.handleCreateExercise)))
	mux.Handle("PATCH /api/v1/admin/exercises/{id}", a.authAdmin(http.HandlerFunc(a.handleUpdateExercise)))
	mux.Handle("POST /api/v1/admin/exercises/{id}/media", a.authAdmin(http.HandlerFunc(a.handleUploadExerciseMedia)))
	mux.Handle("DELETE /api/v1/admin/exercise-media/{id}", a.authAdmin(http.HandlerFunc(a.handleDeleteExerciseMedia)))

	mux.Handle("POST /api/v1/workouts", a.auth(http.HandlerFunc(a.handleCreateWorkout)))
	mux.Handle("GET /api/v1/workouts/active", a.auth(http.HandlerFunc(a.handleActiveWorkout)))
	mux.Handle("GET /api/v1/workouts", a.auth(http.HandlerFunc(a.handleListWorkouts)))
	mux.Handle("GET /api/v1/workouts/{id}", a.auth(http.HandlerFunc(a.handleGetWorkout)))
	mux.Handle("PATCH /api/v1/workouts/{id}", a.auth(http.HandlerFunc(a.handleUpdateWorkout)))
	mux.Handle("POST /api/v1/workouts/{id}/finish", a.auth(http.HandlerFunc(a.handleFinishWorkout)))
	mux.Handle("POST /api/v1/workouts/{id}/exercises", a.auth(http.HandlerFunc(a.handleAddWorkoutExercise)))
	mux.Handle("PATCH /api/v1/workout-exercises/{id}", a.auth(http.HandlerFunc(a.handleUpdateWorkoutExercise)))
	mux.Handle("POST /api/v1/workout-exercises/{id}/sets", a.auth(http.HandlerFunc(a.handleCreateSet)))
	mux.Handle("PATCH /api/v1/workout-sets/{id}", a.auth(http.HandlerFunc(a.handleUpdateSet)))
	mux.Handle("DELETE /api/v1/workout-sets/{id}", a.auth(http.HandlerFunc(a.handleDeleteSet)))

	mux.Handle("POST /api/v1/workouts/{id}/share-links", a.auth(http.HandlerFunc(a.handleCreateShareLink)))
	mux.Handle("DELETE /api/v1/share-links/{id}", a.auth(http.HandlerFunc(a.handleDisableShareLink)))
	mux.HandleFunc("GET /api/v1/public/workouts/{token}", a.handlePublicWorkout)
	mux.Handle("GET /api/v1/workouts/{id}/export.csv", a.auth(http.HandlerFunc(a.handleExportCSV)))
	mux.Handle("GET /api/v1/workouts/{id}/export.json", a.auth(http.HandlerFunc(a.handleExportJSON)))
}

func openStore(path string) (*store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	s := &store{path: path}
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		s.db = initialDatabase()
		if err := s.saveLocked(); err != nil {
			return nil, err
		}
		return s, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(raw))) == 0 {
		s.db = initialDatabase()
		return s, s.saveLocked()
	}
	if err := json.Unmarshal(raw, &s.db); err != nil {
		return nil, err
	}
	if s.db.Sessions == nil {
		s.db.Sessions = map[string]int64{}
	}
	return s, nil
}

func initialDatabase() database {
	now := time.Now().UTC()
	return database{
		NextUserID:            1,
		NextCategoryID:        4,
		NextExerciseID:        4,
		NextMediaID:           1,
		NextWorkoutID:         1,
		NextWorkoutExerciseID: 1,
		NextSetID:             1,
		NextShareLinkID:       1,
		Sessions:              map[string]int64{},
		Categories: []ExerciseCategory{
			{ID: 1, Name: "Силовые", Description: "Упражнения с повторами и весом", SortOrder: 1, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: 2, Name: "Кардио", Description: "Бег, дорожка, велосипед и другие длительные активности", SortOrder: 2, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: 3, Name: "Статика", Description: "Планка, удержания и упражнения на время", SortOrder: 3, IsActive: true, CreatedAt: now, UpdatedAt: now},
		},
		Exercises: []ExerciseTemplate{
			{ID: 1, CategoryID: 1, Title: "Жим лежа", Description: "Базовое упражнение для грудных мышц", MetricSchema: MetricSchema{Type: "strength", Fields: []MetricField{{Key: "reps", Label: "Повторы", Unit: "раз", ValueType: "int", Required: true, Min: floatPtr(1)}, {Key: "weight_kg", Label: "Вес", Unit: "кг", ValueType: "float", Required: false, Min: floatPtr(0)}}, Target: map[string]string{"reps": "8-12"}}, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: 2, CategoryID: 2, Title: "Беговая дорожка", Description: "Кардио на беговой дорожке с учетом скорости и наклона", MetricSchema: MetricSchema{Type: "treadmill", Fields: []MetricField{{Key: "duration_min", Label: "Время", Unit: "мин", ValueType: "float", Required: true, Min: floatPtr(0)}, {Key: "speed_kmh", Label: "Скорость", Unit: "км/ч", ValueType: "float", Required: false, Min: floatPtr(0)}, {Key: "incline_percent", Label: "Наклон", Unit: "%", ValueType: "float", Required: false}, {Key: "distance_km", Label: "Дистанция", Unit: "км", ValueType: "float", Required: false, Min: floatPtr(0)}}}, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: 3, CategoryID: 3, Title: "Планка", Description: "Статическое упражнение на корпус", MetricSchema: MetricSchema{Type: "duration", Fields: []MetricField{{Key: "duration_sec", Label: "Время", Unit: "сек", ValueType: "float", Required: true, Min: floatPtr(1)}}}, IsActive: true, CreatedAt: now, UpdatedAt: now},
		},
	}
}

func floatPtr(v float64) *float64 {
	return &v
}

func (s *store) saveLocked() error {
	raw, err := json.MarshalIndent(s.db, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *store) save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.saveLocked()
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Vary", "Origin")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *app) auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "нужна авторизация")
			return
		}
		a.store.mu.Lock()
		userID, ok := a.store.db.Sessions[token]
		user, found := a.findUserLocked(userID)
		a.store.mu.Unlock()
		if !ok || !found {
			writeError(w, http.StatusUnauthorized, "сессия не найдена")
			return
		}
		r = r.WithContext(withUser(r.Context(), user))
		next.ServeHTTP(w, r)
	})
}

func (a *app) authAdmin(next http.Handler) http.Handler {
	return a.auth(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user := currentUser(r)
		if user.Role != roleAdmin {
			writeError(w, http.StatusForbidden, "нужны права администратора")
			return
		}
		next.ServeHTTP(w, r)
	}))
}

func withUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

func currentUser(r *http.Request) User {
	user, _ := r.Context().Value(userContextKey).(User)
	return user
}

func bearerToken(r *http.Request) string {
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return strings.TrimSpace(parts[1])
}

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
	if err := encoder.Encode(payload); err != nil {
		log.Println("write json:", err)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func pathID(r *http.Request, name string) (int64, error) {
	raw := strings.TrimSpace(r.PathValue(name))
	if raw == "" {
		return 0, errors.New("empty id")
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}
	return id, nil
}

func nowUTC() time.Time {
	return time.Now().UTC()
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func hashPassword(password string) (string, error) {
	if len(password) < 6 {
		return "", errors.New("пароль должен быть не короче 6 символов")
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	const iterations = 120000
	hash := derivePasswordHash([]byte(password), salt, iterations)
	return fmt.Sprintf("v1$%d$%s$%s", iterations, base64.RawURLEncoding.EncodeToString(salt), base64.RawURLEncoding.EncodeToString(hash)), nil
}

func verifyPassword(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 4 || parts[0] != "v1" {
		return false
	}
	iterations, err := strconv.Atoi(parts[1])
	if err != nil || iterations < 1 {
		return false
	}
	salt, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return false
	}
	expected, err := base64.RawURLEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	actual := derivePasswordHash([]byte(password), salt, iterations)
	return hmac.Equal(actual, expected)
}

func derivePasswordHash(password, salt []byte, iterations int) []byte {
	state := sha256.Sum256(append(append([]byte{}, salt...), password...))
	for i := 1; i < iterations; i++ {
		block := append([]byte{}, state[:]...)
		block = append(block, salt...)
		block = append(block, password...)
		state = sha256.Sum256(block)
	}
	return state[:]
}

func (a *app) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	email := normalizeEmail(req.Email)
	if email == "" || !strings.Contains(email, "@") {
		writeError(w, http.StatusBadRequest, "укажите корректный e-mail")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = email
	}
	passwordHash, err := hashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	token, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать сессию")
		return
	}

	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for _, user := range a.store.db.Users {
		if user.Email == email {
			writeError(w, http.StatusConflict, "пользователь с таким e-mail уже есть")
			return
		}
	}
	role := roleUser
	if len(a.store.db.Users) == 0 {
		role = roleAdmin
	}
	now := nowUTC()
	user := User{ID: a.store.db.NextUserID, Email: email, Name: name, PasswordHash: passwordHash, Role: role, CreatedAt: now, UpdatedAt: now}
	a.store.db.NextUserID++
	a.store.db.Users = append(a.store.db.Users, user)
	a.store.db.Sessions[token] = user.ID
	if err := a.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить пользователя")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"token": token, "user": user})
}

func (a *app) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	token, err := randomToken(32)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать сессию")
		return
	}

	email := normalizeEmail(req.Email)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for _, user := range a.store.db.Users {
		if user.Email == email && verifyPassword(req.Password, user.PasswordHash) {
			a.store.db.Sessions[token] = user.ID
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось сохранить сессию")
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{"token": token, "user": user})
			return
		}
	}
	writeError(w, http.StatusUnauthorized, "неверный e-mail или пароль")
}

func (a *app) handleMe(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, currentUser(r))
}

func (a *app) handleListCategories(w http.ResponseWriter, r *http.Request) {
	a.store.mu.Lock()
	categories := append([]ExerciseCategory(nil), a.store.db.Categories...)
	a.store.mu.Unlock()
	sort.Slice(categories, func(i, j int) bool {
		if categories[i].SortOrder == categories[j].SortOrder {
			return categories[i].Name < categories[j].Name
		}
		return categories[i].SortOrder < categories[j].SortOrder
	})
	writeJSON(w, http.StatusOK, categories)
}

func (a *app) handleCreateCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int    `json:"sortOrder"`
		IsActive    *bool  `json:"isActive"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		writeError(w, http.StatusBadRequest, "название категории обязательно")
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	now := nowUTC()
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	category := ExerciseCategory{ID: a.store.db.NextCategoryID, Name: name, Description: strings.TrimSpace(req.Description), SortOrder: req.SortOrder, IsActive: active, CreatedAt: now, UpdatedAt: now}
	a.store.db.NextCategoryID++
	a.store.db.Categories = append(a.store.db.Categories, category)
	if err := a.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить категорию")
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

func (a *app) handleUpdateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		SortOrder   *int    `json:"sortOrder"`
		IsActive    *bool   `json:"isActive"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for i := range a.store.db.Categories {
		if a.store.db.Categories[i].ID == id {
			if req.Name != nil {
				name := strings.TrimSpace(*req.Name)
				if name == "" {
					writeError(w, http.StatusBadRequest, "название категории обязательно")
					return
				}
				a.store.db.Categories[i].Name = name
			}
			if req.Description != nil {
				a.store.db.Categories[i].Description = strings.TrimSpace(*req.Description)
			}
			if req.SortOrder != nil {
				a.store.db.Categories[i].SortOrder = *req.SortOrder
			}
			if req.IsActive != nil {
				a.store.db.Categories[i].IsActive = *req.IsActive
			}
			a.store.db.Categories[i].UpdatedAt = nowUTC()
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось сохранить категорию")
				return
			}
			writeJSON(w, http.StatusOK, a.store.db.Categories[i])
			return
		}
	}
	writeError(w, http.StatusNotFound, "категория не найдена")
}

func (a *app) handleListExercises(w http.ResponseWriter, r *http.Request) {
	includeInactive := r.URL.Query().Get("includeInactive") == "1"
	a.store.mu.Lock()
	result := make([]ExerciseView, 0, len(a.store.db.Exercises))
	for _, exercise := range a.store.db.Exercises {
		if !includeInactive && !exercise.IsActive {
			continue
		}
		result = append(result, a.exerciseViewLocked(exercise))
	}
	a.store.mu.Unlock()
	sort.Slice(result, func(i, j int) bool {
		if result[i].CategoryName == result[j].CategoryName {
			return result[i].Title < result[j].Title
		}
		return result[i].CategoryName < result[j].CategoryName
	})
	writeJSON(w, http.StatusOK, result)
}

func (a *app) handleGetExercise(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	exercise, ok := a.findExerciseLocked(id)
	if !ok {
		writeError(w, http.StatusNotFound, "упражнение не найдено")
		return
	}
	writeJSON(w, http.StatusOK, a.exerciseViewLocked(exercise))
}

func (a *app) handleCreateExercise(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CategoryID   int64        `json:"categoryId"`
		Title        string       `json:"title"`
		Description  string       `json:"description"`
		MetricSchema MetricSchema `json:"metricSchema"`
		IsActive     *bool        `json:"isActive"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	if err := validateExerciseInput(req.CategoryID, req.Title, req.MetricSchema); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	active := true
	if req.IsActive != nil {
		active = *req.IsActive
	}
	user := currentUser(r)
	now := nowUTC()
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	if _, ok := a.findCategoryLocked(req.CategoryID); !ok {
		writeError(w, http.StatusBadRequest, "категория не найдена")
		return
	}
	exercise := ExerciseTemplate{ID: a.store.db.NextExerciseID, CategoryID: req.CategoryID, Title: strings.TrimSpace(req.Title), Description: strings.TrimSpace(req.Description), MetricSchema: req.MetricSchema, IsActive: active, CreatedBy: user.ID, CreatedAt: now, UpdatedAt: now}
	a.store.db.NextExerciseID++
	a.store.db.Exercises = append(a.store.db.Exercises, exercise)
	if err := a.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить упражнение")
		return
	}
	writeJSON(w, http.StatusCreated, a.exerciseViewLocked(exercise))
}

func (a *app) handleUpdateExercise(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var req struct {
		CategoryID   *int64        `json:"categoryId"`
		Title        *string       `json:"title"`
		Description  *string       `json:"description"`
		MetricSchema *MetricSchema `json:"metricSchema"`
		IsActive     *bool         `json:"isActive"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for i := range a.store.db.Exercises {
		if a.store.db.Exercises[i].ID == id {
			categoryID := a.store.db.Exercises[i].CategoryID
			title := a.store.db.Exercises[i].Title
			schema := a.store.db.Exercises[i].MetricSchema
			if req.CategoryID != nil {
				categoryID = *req.CategoryID
			}
			if req.Title != nil {
				title = *req.Title
			}
			if req.MetricSchema != nil {
				schema = *req.MetricSchema
			}
			if err := validateExerciseInput(categoryID, title, schema); err != nil {
				writeError(w, http.StatusBadRequest, err.Error())
				return
			}
			if _, ok := a.findCategoryLocked(categoryID); !ok {
				writeError(w, http.StatusBadRequest, "категория не найдена")
				return
			}
			a.store.db.Exercises[i].CategoryID = categoryID
			a.store.db.Exercises[i].Title = strings.TrimSpace(title)
			if req.Description != nil {
				a.store.db.Exercises[i].Description = strings.TrimSpace(*req.Description)
			}
			a.store.db.Exercises[i].MetricSchema = schema
			if req.IsActive != nil {
				a.store.db.Exercises[i].IsActive = *req.IsActive
			}
			a.store.db.Exercises[i].UpdatedAt = nowUTC()
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось сохранить упражнение")
				return
			}
			writeJSON(w, http.StatusOK, a.exerciseViewLocked(a.store.db.Exercises[i]))
			return
		}
	}
	writeError(w, http.StatusNotFound, "упражнение не найдено")
}

func (a *app) handleUploadExerciseMedia(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 80<<20)
	if err := r.ParseMultipartForm(82 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "не удалось прочитать файл")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "поле file обязательно")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		if guessed := mime.TypeByExtension(strings.ToLower(filepath.Ext(header.Filename))); guessed != "" {
			mimeType = guessed
		}
	}
	mediaType, ok := mediaKind(mimeType)
	if !ok {
		writeError(w, http.StatusBadRequest, "поддерживаются только изображения и видео")
		return
	}
	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = extByMime(mimeType)
	}
	nameToken, err := randomToken(18)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать имя файла")
		return
	}
	fileName := fmt.Sprintf("%d_%s%s", id, nameToken, ext)
	dstPath := filepath.Join(a.uploadDir, fileName)
	dst, err := os.Create(dstPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить файл")
		return
	}
	size, copyErr := io.Copy(dst, file)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(dstPath)
		writeError(w, http.StatusInternalServerError, "не удалось сохранить файл")
		return
	}

	now := nowUTC()
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	if _, ok := a.findExerciseLocked(id); !ok {
		_ = os.Remove(dstPath)
		writeError(w, http.StatusNotFound, "упражнение не найдено")
		return
	}
	media := ExerciseMedia{ID: a.store.db.NextMediaID, ExerciseID: id, MediaType: mediaType, FileURL: "/uploads/" + fileName, MimeType: mimeType, SizeBytes: size, CreatedAt: now}
	a.store.db.NextMediaID++
	a.store.db.Media = append(a.store.db.Media, media)
	if err := a.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить медиа")
		return
	}
	writeJSON(w, http.StatusCreated, media)
}

func (a *app) handleDeleteExerciseMedia(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for i, media := range a.store.db.Media {
		if media.ID == id {
			a.store.db.Media = append(a.store.db.Media[:i], a.store.db.Media[i+1:]...)
			if strings.HasPrefix(media.FileURL, "/uploads/") {
				_ = os.Remove(filepath.Join(a.uploadDir, strings.TrimPrefix(media.FileURL, "/uploads/")))
			}
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось удалить медиа")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	writeError(w, http.StatusNotFound, "медиа не найдено")
}

func (a *app) handleCreateWorkout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
		Notes string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil && err != io.EOF {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	user := currentUser(r)
	now := nowUTC()
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Тренировка " + time.Now().Format("02.01.2006 15:04")
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for _, workout := range a.store.db.Workouts {
		if workout.UserID == user.ID && workout.Status == statusActive {
			writeJSON(w, http.StatusOK, map[string]interface{}{"alreadyActive": true, "workout": a.workoutDetailLocked(workout)})
			return
		}
	}
	workout := Workout{ID: a.store.db.NextWorkoutID, UserID: user.ID, Title: title, Status: statusActive, StartedAt: now, Notes: strings.TrimSpace(req.Notes), CreatedAt: now, UpdatedAt: now}
	a.store.db.NextWorkoutID++
	a.store.db.Workouts = append(a.store.db.Workouts, workout)
	if err := a.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать тренировку")
		return
	}
	writeJSON(w, http.StatusCreated, a.workoutDetailLocked(workout))
}

func (a *app) handleActiveWorkout(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for _, workout := range a.store.db.Workouts {
		if workout.UserID == user.ID && workout.Status == statusActive {
			writeJSON(w, http.StatusOK, a.workoutDetailLocked(workout))
			return
		}
	}
	writeJSON(w, http.StatusOK, nil)
}

func (a *app) handleListWorkouts(w http.ResponseWriter, r *http.Request) {
	user := currentUser(r)
	a.store.mu.Lock()
	result := make([]WorkoutSummary, 0)
	for _, workout := range a.store.db.Workouts {
		if workout.UserID == user.ID {
			result = append(result, a.workoutSummaryLocked(workout))
		}
	}
	a.store.mu.Unlock()
	sort.Slice(result, func(i, j int) bool {
		return result[i].StartedAt.After(result[j].StartedAt)
	})
	writeJSON(w, http.StatusOK, result)
}

func (a *app) handleGetWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	user := currentUser(r)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	workout, ok := a.findWorkoutLocked(id)
	if !ok || workout.UserID != user.ID {
		writeError(w, http.StatusNotFound, "тренировка не найдена")
		return
	}
	writeJSON(w, http.StatusOK, a.workoutDetailLocked(workout))
}

func (a *app) handleUpdateWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var req struct {
		Title *string `json:"title"`
		Notes *string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	user := currentUser(r)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for i := range a.store.db.Workouts {
		if a.store.db.Workouts[i].ID == id && a.store.db.Workouts[i].UserID == user.ID {
			if req.Title != nil {
				title := strings.TrimSpace(*req.Title)
				if title == "" {
					writeError(w, http.StatusBadRequest, "название тренировки обязательно")
					return
				}
				a.store.db.Workouts[i].Title = title
			}
			if req.Notes != nil {
				a.store.db.Workouts[i].Notes = strings.TrimSpace(*req.Notes)
			}
			a.store.db.Workouts[i].UpdatedAt = nowUTC()
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось сохранить тренировку")
				return
			}
			writeJSON(w, http.StatusOK, a.workoutDetailLocked(a.store.db.Workouts[i]))
			return
		}
	}
	writeError(w, http.StatusNotFound, "тренировка не найдена")
}

func (a *app) handleFinishWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	user := currentUser(r)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for i := range a.store.db.Workouts {
		if a.store.db.Workouts[i].ID == id && a.store.db.Workouts[i].UserID == user.ID {
			if a.store.db.Workouts[i].Status != statusCompleted {
				now := nowUTC()
				a.store.db.Workouts[i].Status = statusCompleted
				a.store.db.Workouts[i].FinishedAt = &now
				a.store.db.Workouts[i].UpdatedAt = now
			}
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось завершить тренировку")
				return
			}
			writeJSON(w, http.StatusOK, a.workoutDetailLocked(a.store.db.Workouts[i]))
			return
		}
	}
	writeError(w, http.StatusNotFound, "тренировка не найдена")
}

func (a *app) handleAddWorkoutExercise(w http.ResponseWriter, r *http.Request) {
	workoutID, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var req struct {
		ExerciseID int64  `json:"exerciseId"`
		Notes      string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	user := currentUser(r)
	now := nowUTC()
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	workout, ok := a.findWorkoutLocked(workoutID)
	if !ok || workout.UserID != user.ID {
		writeError(w, http.StatusNotFound, "тренировка не найдена")
		return
	}
	if workout.Status != statusActive {
		writeError(w, http.StatusBadRequest, "нельзя менять завершенную тренировку")
		return
	}
	exercise, ok := a.findExerciseLocked(req.ExerciseID)
	if !ok || !exercise.IsActive {
		writeError(w, http.StatusBadRequest, "упражнение не найдено или скрыто")
		return
	}
	position := 1
	for _, item := range a.store.db.WorkoutExercises {
		if item.WorkoutID == workoutID && item.Position >= position {
			position = item.Position + 1
		}
	}
	item := WorkoutExercise{ID: a.store.db.NextWorkoutExerciseID, WorkoutID: workoutID, ExerciseID: exercise.ID, ExerciseSnapshot: a.exerciseSnapshotLocked(exercise), Position: position, Notes: strings.TrimSpace(req.Notes), CreatedAt: now, UpdatedAt: now}
	a.store.db.NextWorkoutExerciseID++
	a.store.db.WorkoutExercises = append(a.store.db.WorkoutExercises, item)
	if err := a.touchWorkoutLocked(workoutID); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if err := a.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось добавить упражнение")
		return
	}
	writeJSON(w, http.StatusCreated, a.workoutDetailLocked(workout))
}

func (a *app) handleUpdateWorkoutExercise(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var req struct {
		Notes    *string `json:"notes"`
		Position *int    `json:"position"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	user := currentUser(r)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for i := range a.store.db.WorkoutExercises {
		if a.store.db.WorkoutExercises[i].ID == id {
			workout, ok := a.findWorkoutLocked(a.store.db.WorkoutExercises[i].WorkoutID)
			if !ok || workout.UserID != user.ID {
				writeError(w, http.StatusNotFound, "упражнение тренировки не найдено")
				return
			}
			if workout.Status != statusActive {
				writeError(w, http.StatusBadRequest, "нельзя менять завершенную тренировку")
				return
			}
			if req.Notes != nil {
				a.store.db.WorkoutExercises[i].Notes = strings.TrimSpace(*req.Notes)
			}
			if req.Position != nil && *req.Position > 0 {
				a.store.db.WorkoutExercises[i].Position = *req.Position
			}
			a.store.db.WorkoutExercises[i].UpdatedAt = nowUTC()
			_ = a.touchWorkoutLocked(workout.ID)
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось сохранить упражнение тренировки")
				return
			}
			writeJSON(w, http.StatusOK, a.workoutDetailLocked(workout))
			return
		}
	}
	writeError(w, http.StatusNotFound, "упражнение тренировки не найдено")
}

func (a *app) handleCreateSet(w http.ResponseWriter, r *http.Request) {
	workoutExerciseID, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var req struct {
		ClientID     string                 `json:"clientId"`
		MetricValues map[string]interface{} `json:"metricValues"`
		Notes        string                 `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	user := currentUser(r)
	if strings.TrimSpace(req.ClientID) == "" {
		req.ClientID, _ = randomToken(12)
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	item, workout, ok := a.findWorkoutExerciseWithWorkoutLocked(workoutExerciseID)
	if !ok || workout.UserID != user.ID {
		writeError(w, http.StatusNotFound, "упражнение тренировки не найдено")
		return
	}
	if workout.Status != statusActive {
		writeError(w, http.StatusBadRequest, "нельзя менять завершенную тренировку")
		return
	}
	for _, set := range a.store.db.Sets {
		if set.WorkoutExerciseID == workoutExerciseID && set.ClientID == req.ClientID {
			writeJSON(w, http.StatusOK, set)
			return
		}
	}
	values, err := normalizeMetricValues(item.ExerciseSnapshot.MetricSchema, req.MetricValues)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	setIndex := 1
	for _, set := range a.store.db.Sets {
		if set.WorkoutExerciseID == workoutExerciseID && set.SetIndex >= setIndex {
			setIndex = set.SetIndex + 1
		}
	}
	now := nowUTC()
	set := WorkoutSet{ID: a.store.db.NextSetID, WorkoutExerciseID: workoutExerciseID, ClientID: req.ClientID, SetIndex: setIndex, MetricValues: values, Notes: strings.TrimSpace(req.Notes), CompletedAt: now, CreatedAt: now, UpdatedAt: now}
	a.store.db.NextSetID++
	a.store.db.Sets = append(a.store.db.Sets, set)
	_ = a.touchWorkoutLocked(workout.ID)
	if err := a.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить подход")
		return
	}
	writeJSON(w, http.StatusCreated, set)
}

func (a *app) handleUpdateSet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	var req struct {
		MetricValues *map[string]interface{} `json:"metricValues"`
		Notes        *string                 `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "некорректный JSON")
		return
	}
	user := currentUser(r)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for i := range a.store.db.Sets {
		if a.store.db.Sets[i].ID == id {
			item, workout, ok := a.findWorkoutExerciseWithWorkoutLocked(a.store.db.Sets[i].WorkoutExerciseID)
			if !ok || workout.UserID != user.ID {
				writeError(w, http.StatusNotFound, "подход не найден")
				return
			}
			if workout.Status != statusActive {
				writeError(w, http.StatusBadRequest, "нельзя менять завершенную тренировку")
				return
			}
			if req.MetricValues != nil {
				values, err := normalizeMetricValues(item.ExerciseSnapshot.MetricSchema, *req.MetricValues)
				if err != nil {
					writeError(w, http.StatusBadRequest, err.Error())
					return
				}
				a.store.db.Sets[i].MetricValues = values
			}
			if req.Notes != nil {
				a.store.db.Sets[i].Notes = strings.TrimSpace(*req.Notes)
			}
			a.store.db.Sets[i].UpdatedAt = nowUTC()
			_ = a.touchWorkoutLocked(workout.ID)
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось сохранить подход")
				return
			}
			writeJSON(w, http.StatusOK, a.store.db.Sets[i])
			return
		}
	}
	writeError(w, http.StatusNotFound, "подход не найден")
}

func (a *app) handleDeleteSet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	user := currentUser(r)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for i, set := range a.store.db.Sets {
		if set.ID == id {
			_, workout, ok := a.findWorkoutExerciseWithWorkoutLocked(set.WorkoutExerciseID)
			if !ok || workout.UserID != user.ID {
				writeError(w, http.StatusNotFound, "подход не найден")
				return
			}
			if workout.Status != statusActive {
				writeError(w, http.StatusBadRequest, "нельзя менять завершенную тренировку")
				return
			}
			a.store.db.Sets = append(a.store.db.Sets[:i], a.store.db.Sets[i+1:]...)
			a.reindexSetsLocked(set.WorkoutExerciseID)
			_ = a.touchWorkoutLocked(workout.ID)
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось удалить подход")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	writeError(w, http.StatusNotFound, "подход не найден")
}

func (a *app) handleCreateShareLink(w http.ResponseWriter, r *http.Request) {
	workoutID, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	user := currentUser(r)
	token, err := randomToken(24)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось создать ссылку")
		return
	}
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	workout, ok := a.findWorkoutLocked(workoutID)
	if !ok || workout.UserID != user.ID {
		writeError(w, http.StatusNotFound, "тренировка не найдена")
		return
	}
	link := ShareLink{ID: a.store.db.NextShareLinkID, WorkoutID: workoutID, Token: token, TokenHash: tokenHash(token), IsActive: true, CreatedAt: nowUTC()}
	a.store.db.NextShareLinkID++
	a.store.db.ShareLinks = append(a.store.db.ShareLinks, link)
	if err := a.store.saveLocked(); err != nil {
		writeError(w, http.StatusInternalServerError, "не удалось сохранить ссылку")
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

func (a *app) handleDisableShareLink(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	user := currentUser(r)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for i := range a.store.db.ShareLinks {
		if a.store.db.ShareLinks[i].ID == id {
			workout, ok := a.findWorkoutLocked(a.store.db.ShareLinks[i].WorkoutID)
			if !ok || workout.UserID != user.ID {
				writeError(w, http.StatusNotFound, "ссылка не найдена")
				return
			}
			a.store.db.ShareLinks[i].IsActive = false
			if err := a.store.saveLocked(); err != nil {
				writeError(w, http.StatusInternalServerError, "не удалось отключить ссылку")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	writeError(w, http.StatusNotFound, "ссылка не найдена")
}

func (a *app) handlePublicWorkout(w http.ResponseWriter, r *http.Request) {
	token := strings.TrimSpace(r.PathValue("token"))
	if token == "" {
		writeError(w, http.StatusBadRequest, "некорректная ссылка")
		return
	}
	hash := tokenHash(token)
	a.store.mu.Lock()
	defer a.store.mu.Unlock()
	for _, link := range a.store.db.ShareLinks {
		if link.TokenHash == hash && link.IsActive {
			if link.ExpiresAt != nil && link.ExpiresAt.Before(nowUTC()) {
				writeError(w, http.StatusGone, "ссылка устарела")
				return
			}
			workout, ok := a.findWorkoutLocked(link.WorkoutID)
			if !ok {
				writeError(w, http.StatusNotFound, "тренировка не найдена")
				return
			}
			writeJSON(w, http.StatusOK, a.workoutDetailLocked(workout))
			return
		}
	}
	writeError(w, http.StatusNotFound, "ссылка не найдена")
}

func (a *app) handleExportJSON(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	user := currentUser(r)
	a.store.mu.Lock()
	workout, ok := a.findWorkoutLocked(id)
	if !ok || workout.UserID != user.ID {
		a.store.mu.Unlock()
		writeError(w, http.StatusNotFound, "тренировка не найдена")
		return
	}
	detail := a.workoutDetailLocked(workout)
	a.store.mu.Unlock()
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"workout_%d.json\"", id))
	_ = json.NewEncoder(w).Encode(detail)
}

func (a *app) handleExportCSV(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, http.StatusBadRequest, "некорректный id")
		return
	}
	user := currentUser(r)
	a.store.mu.Lock()
	workout, ok := a.findWorkoutLocked(id)
	if !ok || workout.UserID != user.ID {
		a.store.mu.Unlock()
		writeError(w, http.StatusNotFound, "тренировка не найдена")
		return
	}
	detail := a.workoutDetailLocked(workout)
	a.store.mu.Unlock()

	fieldKeys := orderedMetricKeys(detail)
	header := []string{"workout_id", "workout_title", "started_at", "finished_at", "exercise", "set_index"}
	header = append(header, fieldKeys...)
	header = append(header, "notes")

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"workout_%d.csv\"", id))
	writer := csv.NewWriter(w)
	_ = writer.Write(header)
	for _, exercise := range detail.Exercises {
		for _, set := range exercise.Sets {
			row := []string{
				strconv.FormatInt(detail.ID, 10),
				detail.Title,
				detail.StartedAt.Format(time.RFC3339),
				formatOptionalTime(detail.FinishedAt),
				exercise.ExerciseSnapshot.Title,
				strconv.Itoa(set.SetIndex),
			}
			for _, key := range fieldKeys {
				row = append(row, metricToString(set.MetricValues[key]))
			}
			row = append(row, set.Notes)
			_ = writer.Write(row)
		}
	}
	writer.Flush()
}

func validateExerciseInput(categoryID int64, title string, schema MetricSchema) error {
	if categoryID <= 0 {
		return errors.New("категория обязательна")
	}
	if strings.TrimSpace(title) == "" {
		return errors.New("название упражнения обязательно")
	}
	if strings.TrimSpace(schema.Type) == "" {
		return errors.New("тип схемы метрик обязателен")
	}
	if len(schema.Fields) == 0 {
		return errors.New("добавьте хотя бы одну метрику")
	}
	seen := map[string]bool{}
	for _, field := range schema.Fields {
		key := strings.TrimSpace(field.Key)
		if key == "" {
			return errors.New("ключ метрики обязателен")
		}
		if seen[key] {
			return fmt.Errorf("метрика %s дублируется", key)
		}
		seen[key] = true
		if strings.TrimSpace(field.Label) == "" {
			return fmt.Errorf("у метрики %s нет названия", key)
		}
		switch field.ValueType {
		case "int", "float", "text":
		default:
			return fmt.Errorf("метрика %s имеет неподдерживаемый тип", key)
		}
	}
	return nil
}

func normalizeMetricValues(schema MetricSchema, input map[string]interface{}) (map[string]interface{}, error) {
	if input == nil {
		input = map[string]interface{}{}
	}
	values := map[string]interface{}{}
	for _, field := range schema.Fields {
		raw, exists := input[field.Key]
		if !exists || raw == nil || raw == "" {
			if field.Required {
				return nil, fmt.Errorf("поле %s обязательно", field.Label)
			}
			continue
		}
		switch field.ValueType {
		case "int":
			n, err := asFloat(raw)
			if err != nil || math.Mod(n, 1) != 0 {
				return nil, fmt.Errorf("поле %s должно быть целым числом", field.Label)
			}
			if err := checkBounds(field, n); err != nil {
				return nil, err
			}
			values[field.Key] = int64(n)
		case "float":
			n, err := asFloat(raw)
			if err != nil {
				return nil, fmt.Errorf("поле %s должно быть числом", field.Label)
			}
			if err := checkBounds(field, n); err != nil {
				return nil, err
			}
			values[field.Key] = n
		case "text":
			text := strings.TrimSpace(fmt.Sprint(raw))
			if text == "" && field.Required {
				return nil, fmt.Errorf("поле %s обязательно", field.Label)
			}
			if text != "" {
				values[field.Key] = text
			}
		}
	}
	return values, nil
}

func asFloat(v interface{}) (float64, error) {
	switch value := v.(type) {
	case float64:
		return value, nil
	case float32:
		return float64(value), nil
	case int:
		return float64(value), nil
	case int64:
		return float64(value), nil
	case string:
		text := strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
		if text == "" {
			return 0, errors.New("empty")
		}
		return strconv.ParseFloat(text, 64)
	default:
		return 0, fmt.Errorf("unsupported %T", v)
	}
}

func checkBounds(field MetricField, value float64) error {
	if field.Min != nil && value < *field.Min {
		return fmt.Errorf("поле %s меньше минимума", field.Label)
	}
	if field.Max != nil && value > *field.Max {
		return fmt.Errorf("поле %s больше максимума", field.Label)
	}
	return nil
}

func mediaKind(mimeType string) (string, bool) {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "photo", true
	case strings.HasPrefix(mimeType, "video/"):
		return "video", true
	default:
		return "", false
	}
}

func extByMime(mimeType string) string {
	switch mimeType {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "video/webm":
		return ".webm"
	default:
		if strings.HasPrefix(mimeType, "video/") {
			return ".mp4"
		}
		return ".bin"
	}
}

func (a *app) findUserLocked(id int64) (User, bool) {
	for _, user := range a.store.db.Users {
		if user.ID == id {
			return user, true
		}
	}
	return User{}, false
}

func (a *app) findCategoryLocked(id int64) (ExerciseCategory, bool) {
	for _, category := range a.store.db.Categories {
		if category.ID == id {
			return category, true
		}
	}
	return ExerciseCategory{}, false
}

func (a *app) findExerciseLocked(id int64) (ExerciseTemplate, bool) {
	for _, exercise := range a.store.db.Exercises {
		if exercise.ID == id {
			return exercise, true
		}
	}
	return ExerciseTemplate{}, false
}

func (a *app) findWorkoutLocked(id int64) (Workout, bool) {
	for _, workout := range a.store.db.Workouts {
		if workout.ID == id {
			return workout, true
		}
	}
	return Workout{}, false
}

func (a *app) findWorkoutExerciseWithWorkoutLocked(id int64) (WorkoutExercise, Workout, bool) {
	for _, item := range a.store.db.WorkoutExercises {
		if item.ID == id {
			workout, ok := a.findWorkoutLocked(item.WorkoutID)
			return item, workout, ok
		}
	}
	return WorkoutExercise{}, Workout{}, false
}

func (a *app) exerciseViewLocked(exercise ExerciseTemplate) ExerciseView {
	categoryName := ""
	if category, ok := a.findCategoryLocked(exercise.CategoryID); ok {
		categoryName = category.Name
	}
	return ExerciseView{ExerciseTemplate: exercise, CategoryName: categoryName, Media: a.exerciseMediaLocked(exercise.ID)}
}

func (a *app) exerciseMediaLocked(exerciseID int64) []ExerciseMedia {
	media := []ExerciseMedia{}
	for _, item := range a.store.db.Media {
		if item.ExerciseID == exerciseID {
			media = append(media, item)
		}
	}
	sort.Slice(media, func(i, j int) bool { return media[i].ID < media[j].ID })
	return media
}

func (a *app) exerciseSnapshotLocked(exercise ExerciseTemplate) ExerciseSnapshot {
	categoryName := ""
	if category, ok := a.findCategoryLocked(exercise.CategoryID); ok {
		categoryName = category.Name
	}
	return ExerciseSnapshot{ExerciseID: exercise.ID, Title: exercise.Title, Description: exercise.Description, CategoryName: categoryName, MetricSchema: exercise.MetricSchema, Media: a.exerciseMediaLocked(exercise.ID), SnapshotTaken: nowUTC()}
}

func (a *app) workoutDetailLocked(workout Workout) WorkoutDetail {
	items := []WorkoutExerciseDetail{}
	for _, item := range a.store.db.WorkoutExercises {
		if item.WorkoutID == workout.ID {
			detail := WorkoutExerciseDetail{WorkoutExercise: item}
			for _, set := range a.store.db.Sets {
				if set.WorkoutExerciseID == item.ID {
					detail.Sets = append(detail.Sets, set)
				}
			}
			sort.Slice(detail.Sets, func(i, j int) bool {
				return detail.Sets[i].SetIndex < detail.Sets[j].SetIndex
			})
			items = append(items, detail)
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Position < items[j].Position
	})
	return WorkoutDetail{Workout: workout, Exercises: items}
}

func (a *app) workoutSummaryLocked(workout Workout) WorkoutSummary {
	summary := WorkoutSummary{Workout: workout}
	for _, item := range a.store.db.WorkoutExercises {
		if item.WorkoutID == workout.ID {
			summary.ExerciseCount++
			for _, set := range a.store.db.Sets {
				if set.WorkoutExerciseID == item.ID {
					summary.SetCount++
				}
			}
		}
	}
	return summary
}

func (a *app) touchWorkoutLocked(id int64) error {
	for i := range a.store.db.Workouts {
		if a.store.db.Workouts[i].ID == id {
			a.store.db.Workouts[i].UpdatedAt = nowUTC()
			return nil
		}
	}
	return errors.New("тренировка не найдена")
}

func (a *app) reindexSetsLocked(workoutExerciseID int64) {
	indexes := []int{}
	for i, set := range a.store.db.Sets {
		if set.WorkoutExerciseID == workoutExerciseID {
			indexes = append(indexes, i)
		}
	}
	sort.Slice(indexes, func(i, j int) bool {
		return a.store.db.Sets[indexes[i]].SetIndex < a.store.db.Sets[indexes[j]].SetIndex
	})
	for order, idx := range indexes {
		a.store.db.Sets[idx].SetIndex = order + 1
	}
}

func orderedMetricKeys(detail WorkoutDetail) []string {
	seen := map[string]bool{}
	keys := []string{}
	for _, exercise := range detail.Exercises {
		for _, field := range exercise.ExerciseSnapshot.MetricSchema.Fields {
			if !seen[field.Key] {
				seen[field.Key] = true
				keys = append(keys, field.Key)
			}
		}
	}
	return keys
}

func metricToString(value interface{}) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int64:
		return strconv.FormatInt(v, 10)
	case int:
		return strconv.Itoa(v)
	default:
		return fmt.Sprint(v)
	}
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}
