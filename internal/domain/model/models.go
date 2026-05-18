package model

import "time"

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	WorkoutStatusActive    = "active"
	WorkoutStatusCompleted = "completed"
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

type State struct {
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
