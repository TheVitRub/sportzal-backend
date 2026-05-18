package jsonstore

import (
	"time"

	"workout-app/backend/internal/domain/model"
)

func SeedState() model.State {
	now := time.Now().UTC()
	return model.State{
		NextUserID:            1,
		NextCategoryID:        4,
		NextExerciseID:        4,
		NextMediaID:           1,
		NextWorkoutID:         1,
		NextWorkoutExerciseID: 1,
		NextSetID:             1,
		NextShareLinkID:       1,
		Sessions:              map[string]int64{},
		Categories: []model.ExerciseCategory{
			{ID: 1, Name: "Силовые", Description: "Упражнения с повторами и весом", SortOrder: 1, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: 2, Name: "Кардио", Description: "Бег, дорожка, велосипед и другие длительные активности", SortOrder: 2, IsActive: true, CreatedAt: now, UpdatedAt: now},
			{ID: 3, Name: "Статика", Description: "Планка, удержания и упражнения на время", SortOrder: 3, IsActive: true, CreatedAt: now, UpdatedAt: now},
		},
		Exercises: []model.ExerciseTemplate{
			strengthExercise(now),
			treadmillExercise(now),
			plankExercise(now),
		},
	}
}

func strengthExercise(now time.Time) model.ExerciseTemplate {
	return model.ExerciseTemplate{
		ID:          1,
		CategoryID:  1,
		Title:       "Жим лежа",
		Description: "Базовое упражнение для грудных мышц",
		MetricSchema: model.MetricSchema{
			Type: "strength",
			Fields: []model.MetricField{
				{Key: "reps", Label: "Повторы", Unit: "раз", ValueType: "int", Required: true, Min: floatPtr(1)},
				{Key: "weight_kg", Label: "Вес", Unit: "кг", ValueType: "float", Required: false, Min: floatPtr(0)},
			},
			Target: map[string]string{"reps": "8-12"},
		},
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func treadmillExercise(now time.Time) model.ExerciseTemplate {
	return model.ExerciseTemplate{
		ID:          2,
		CategoryID:  2,
		Title:       "Беговая дорожка",
		Description: "Кардио на беговой дорожке с учетом скорости и наклона",
		MetricSchema: model.MetricSchema{
			Type: "treadmill",
			Fields: []model.MetricField{
				{Key: "duration_min", Label: "Время", Unit: "мин", ValueType: "float", Required: true, Min: floatPtr(0)},
				{Key: "speed_kmh", Label: "Скорость", Unit: "км/ч", ValueType: "float", Required: false, Min: floatPtr(0)},
				{Key: "incline_percent", Label: "Наклон", Unit: "%", ValueType: "float", Required: false},
				{Key: "distance_km", Label: "Дистанция", Unit: "км", ValueType: "float", Required: false, Min: floatPtr(0)},
			},
		},
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func plankExercise(now time.Time) model.ExerciseTemplate {
	return model.ExerciseTemplate{
		ID:          3,
		CategoryID:  3,
		Title:       "Планка",
		Description: "Статическое упражнение на корпус",
		MetricSchema: model.MetricSchema{
			Type: "duration",
			Fields: []model.MetricField{
				{Key: "duration_sec", Label: "Время", Unit: "сек", ValueType: "float", Required: true, Min: floatPtr(1)},
			},
		},
		IsActive:  true,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
