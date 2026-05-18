package application

import (
	"sort"
	"time"

	"workout-app/backend/internal/domain/model"
)

func nowUTC() time.Time {
	return time.Now().UTC()
}

func findUser(state *model.State, id int64) (model.User, bool) {
	for _, user := range state.Users {
		if user.ID == id {
			return user, true
		}
	}
	return model.User{}, false
}

func findCategory(state *model.State, id int64) (model.ExerciseCategory, bool) {
	for _, category := range state.Categories {
		if category.ID == id {
			return category, true
		}
	}
	return model.ExerciseCategory{}, false
}

func findExercise(state *model.State, id int64) (model.ExerciseTemplate, bool) {
	for _, exercise := range state.Exercises {
		if exercise.ID == id {
			return exercise, true
		}
	}
	return model.ExerciseTemplate{}, false
}

func findWorkout(state *model.State, id int64) (model.Workout, bool) {
	for _, workout := range state.Workouts {
		if workout.ID == id {
			return workout, true
		}
	}
	return model.Workout{}, false
}

func findWorkoutExerciseWithWorkout(state *model.State, id int64) (model.WorkoutExercise, model.Workout, bool) {
	for _, item := range state.WorkoutExercises {
		if item.ID == id {
			workout, ok := findWorkout(state, item.WorkoutID)
			return item, workout, ok
		}
	}
	return model.WorkoutExercise{}, model.Workout{}, false
}

func exerciseView(state *model.State, exercise model.ExerciseTemplate) model.ExerciseView {
	categoryName := ""
	if category, ok := findCategory(state, exercise.CategoryID); ok {
		categoryName = category.Name
	}

	return model.ExerciseView{
		ExerciseTemplate: exercise,
		CategoryName:     categoryName,
		Media:            exerciseMedia(state, exercise.ID),
	}
}

func exerciseMedia(state *model.State, exerciseID int64) []model.ExerciseMedia {
	media := []model.ExerciseMedia{}
	for _, item := range state.Media {
		if item.ExerciseID == exerciseID {
			media = append(media, item)
		}
	}
	sort.Slice(media, func(i, j int) bool {
		return media[i].ID < media[j].ID
	})
	return media
}

func exerciseSnapshot(state *model.State, exercise model.ExerciseTemplate) model.ExerciseSnapshot {
	categoryName := ""
	if category, ok := findCategory(state, exercise.CategoryID); ok {
		categoryName = category.Name
	}

	return model.ExerciseSnapshot{
		ExerciseID:    exercise.ID,
		Title:         exercise.Title,
		Description:   exercise.Description,
		CategoryName:  categoryName,
		MetricSchema:  exercise.MetricSchema,
		Media:         exerciseMedia(state, exercise.ID),
		SnapshotTaken: nowUTC(),
	}
}

func workoutDetail(state *model.State, workout model.Workout) model.WorkoutDetail {
	items := []model.WorkoutExerciseDetail{}
	for _, item := range state.WorkoutExercises {
		if item.WorkoutID != workout.ID {
			continue
		}
		detail := model.WorkoutExerciseDetail{WorkoutExercise: item, Sets: []model.WorkoutSet{}}
		for _, set := range state.Sets {
			if set.WorkoutExerciseID == item.ID {
				detail.Sets = append(detail.Sets, set)
			}
		}
		sort.Slice(detail.Sets, func(i, j int) bool {
			return detail.Sets[i].SetIndex < detail.Sets[j].SetIndex
		})
		items = append(items, detail)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Position < items[j].Position
	})
	return model.WorkoutDetail{Workout: workout, Exercises: items}
}

func workoutSummary(state *model.State, workout model.Workout) model.WorkoutSummary {
	summary := model.WorkoutSummary{Workout: workout}
	for _, item := range state.WorkoutExercises {
		if item.WorkoutID != workout.ID {
			continue
		}
		summary.ExerciseCount++
		for _, set := range state.Sets {
			if set.WorkoutExerciseID == item.ID {
				summary.SetCount++
			}
		}
	}
	return summary
}

func touchWorkout(state *model.State, id int64) {
	for i := range state.Workouts {
		if state.Workouts[i].ID == id {
			state.Workouts[i].UpdatedAt = nowUTC()
			return
		}
	}
}

func reindexSets(state *model.State, workoutExerciseID int64) {
	indexes := []int{}
	for i, set := range state.Sets {
		if set.WorkoutExerciseID == workoutExerciseID {
			indexes = append(indexes, i)
		}
	}
	sort.Slice(indexes, func(i, j int) bool {
		return state.Sets[indexes[i]].SetIndex < state.Sets[indexes[j]].SetIndex
	})
	for order, idx := range indexes {
		state.Sets[idx].SetIndex = order + 1
	}
}
