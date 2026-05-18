package application

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"workout-app/backend/internal/domain/apperrors"
	"workout-app/backend/internal/domain/interfaces"
	"workout-app/backend/internal/domain/model"
	domainservice "workout-app/backend/internal/domain/service"
	"workout-app/backend/pkg/security"
)

type WorkoutService struct {
	repo interfaces.StateRepository
}

type CreateWorkoutInput struct {
	UserID int64
	Title  string
	Notes  string
}

type UpdateWorkoutInput struct {
	UserID int64
	Title  *string
	Notes  *string
}

type AddWorkoutExerciseInput struct {
	UserID     int64
	WorkoutID  int64
	ExerciseID int64
	Notes      string
}

type UpdateWorkoutExerciseInput struct {
	UserID   int64
	ID       int64
	Notes    *string
	Position *int
}

type CreateSetInput struct {
	UserID       int64
	ExerciseID   int64
	ClientID     string
	MetricValues map[string]interface{}
	Notes        string
}

type UpdateSetInput struct {
	UserID       int64
	ID           int64
	MetricValues *map[string]interface{}
	Notes        *string
}

func NewWorkoutService(repo interfaces.StateRepository) *WorkoutService {
	return &WorkoutService{repo: repo}
}

func (s *WorkoutService) CreateWorkout(ctx context.Context, input CreateWorkoutInput) (model.WorkoutDetail, bool, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		title = "Тренировка " + time.Now().Format("02.01.2006 15:04")
	}

	var detail model.WorkoutDetail
	alreadyActive := false
	err := s.repo.Update(ctx, func(state *model.State) error {
		for _, workout := range state.Workouts {
			if workout.UserID == input.UserID && workout.Status == model.WorkoutStatusActive {
				detail = workoutDetail(state, workout)
				alreadyActive = true
				return nil
			}
		}

		now := nowUTC()
		workout := model.Workout{
			ID:        state.NextWorkoutID,
			UserID:    input.UserID,
			Title:     title,
			Status:    model.WorkoutStatusActive,
			StartedAt: now,
			Notes:     strings.TrimSpace(input.Notes),
			CreatedAt: now,
			UpdatedAt: now,
		}
		state.NextWorkoutID++
		state.Workouts = append(state.Workouts, workout)
		detail = workoutDetail(state, workout)
		return nil
	})
	return detail, alreadyActive, err
}

func (s *WorkoutService) ActiveWorkout(ctx context.Context, userID int64) (*model.WorkoutDetail, error) {
	var detail *model.WorkoutDetail
	err := s.repo.View(ctx, func(state model.State) error {
		for _, workout := range state.Workouts {
			if workout.UserID == userID && workout.Status == model.WorkoutStatusActive {
				value := workoutDetail(&state, workout)
				detail = &value
				return nil
			}
		}
		return nil
	})
	return detail, err
}

func (s *WorkoutService) ListWorkouts(ctx context.Context, userID int64) ([]model.WorkoutSummary, error) {
	var workouts []model.WorkoutSummary
	err := s.repo.View(ctx, func(state model.State) error {
		for _, workout := range state.Workouts {
			if workout.UserID == userID {
				workouts = append(workouts, workoutSummary(&state, workout))
			}
		}
		sort.Slice(workouts, func(i, j int) bool {
			return workouts[i].StartedAt.After(workouts[j].StartedAt)
		})
		return nil
	})
	return workouts, err
}

func (s *WorkoutService) GetWorkout(ctx context.Context, userID, workoutID int64) (model.WorkoutDetail, error) {
	var detail model.WorkoutDetail
	err := s.repo.View(ctx, func(state model.State) error {
		workout, ok := findWorkout(&state, workoutID)
		if !ok || workout.UserID != userID {
			return apperrors.NotFound("тренировка не найдена")
		}
		detail = workoutDetail(&state, workout)
		return nil
	})
	return detail, err
}

func (s *WorkoutService) UpdateWorkout(ctx context.Context, workoutID int64, input UpdateWorkoutInput) (model.WorkoutDetail, error) {
	var detail model.WorkoutDetail
	err := s.repo.Update(ctx, func(state *model.State) error {
		for i := range state.Workouts {
			if state.Workouts[i].ID != workoutID || state.Workouts[i].UserID != input.UserID {
				continue
			}
			if input.Title != nil {
				title := strings.TrimSpace(*input.Title)
				if title == "" {
					return apperrors.BadRequest("название тренировки обязательно")
				}
				state.Workouts[i].Title = title
			}
			if input.Notes != nil {
				state.Workouts[i].Notes = strings.TrimSpace(*input.Notes)
			}
			state.Workouts[i].UpdatedAt = nowUTC()
			detail = workoutDetail(state, state.Workouts[i])
			return nil
		}
		return apperrors.NotFound("тренировка не найдена")
	})
	return detail, err
}

func (s *WorkoutService) FinishWorkout(ctx context.Context, userID, workoutID int64) (model.WorkoutDetail, error) {
	var detail model.WorkoutDetail
	err := s.repo.Update(ctx, func(state *model.State) error {
		for i := range state.Workouts {
			if state.Workouts[i].ID != workoutID || state.Workouts[i].UserID != userID {
				continue
			}
			if state.Workouts[i].Status != model.WorkoutStatusCompleted {
				now := nowUTC()
				state.Workouts[i].Status = model.WorkoutStatusCompleted
				state.Workouts[i].FinishedAt = &now
				state.Workouts[i].UpdatedAt = now
			}
			detail = workoutDetail(state, state.Workouts[i])
			return nil
		}
		return apperrors.NotFound("тренировка не найдена")
	})
	return detail, err
}

func (s *WorkoutService) AddWorkoutExercise(ctx context.Context, input AddWorkoutExerciseInput) (model.WorkoutDetail, error) {
	var detail model.WorkoutDetail
	err := s.repo.Update(ctx, func(state *model.State) error {
		workout, ok := findWorkout(state, input.WorkoutID)
		if !ok || workout.UserID != input.UserID {
			return apperrors.NotFound("тренировка не найдена")
		}
		if workout.Status != model.WorkoutStatusActive {
			return apperrors.BadRequest("нельзя менять завершенную тренировку")
		}
		exercise, ok := findExercise(state, input.ExerciseID)
		if !ok || !exercise.IsActive {
			return apperrors.BadRequest("упражнение не найдено или скрыто")
		}

		position := 1
		for _, item := range state.WorkoutExercises {
			if item.WorkoutID == input.WorkoutID && item.Position >= position {
				position = item.Position + 1
			}
		}

		now := nowUTC()
		item := model.WorkoutExercise{
			ID:               state.NextWorkoutExerciseID,
			WorkoutID:        input.WorkoutID,
			ExerciseID:       exercise.ID,
			ExerciseSnapshot: exerciseSnapshot(state, exercise),
			Position:         position,
			Notes:            strings.TrimSpace(input.Notes),
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		state.NextWorkoutExerciseID++
		state.WorkoutExercises = append(state.WorkoutExercises, item)
		touchWorkout(state, input.WorkoutID)
		if updated, ok := findWorkout(state, input.WorkoutID); ok {
			detail = workoutDetail(state, updated)
		}
		return nil
	})
	return detail, err
}

func (s *WorkoutService) UpdateWorkoutExercise(ctx context.Context, input UpdateWorkoutExerciseInput) (model.WorkoutDetail, error) {
	var detail model.WorkoutDetail
	err := s.repo.Update(ctx, func(state *model.State) error {
		for i := range state.WorkoutExercises {
			if state.WorkoutExercises[i].ID != input.ID {
				continue
			}
			workout, ok := findWorkout(state, state.WorkoutExercises[i].WorkoutID)
			if !ok || workout.UserID != input.UserID {
				return apperrors.NotFound("упражнение тренировки не найдено")
			}
			if workout.Status != model.WorkoutStatusActive {
				return apperrors.BadRequest("нельзя менять завершенную тренировку")
			}
			if input.Notes != nil {
				state.WorkoutExercises[i].Notes = strings.TrimSpace(*input.Notes)
			}
			if input.Position != nil && *input.Position > 0 {
				state.WorkoutExercises[i].Position = *input.Position
			}
			state.WorkoutExercises[i].UpdatedAt = nowUTC()
			touchWorkout(state, workout.ID)
			if updated, ok := findWorkout(state, workout.ID); ok {
				detail = workoutDetail(state, updated)
			}
			return nil
		}
		return apperrors.NotFound("упражнение тренировки не найдено")
	})
	return detail, err
}

func (s *WorkoutService) CreateSet(ctx context.Context, input CreateSetInput) (model.WorkoutSet, error) {
	if strings.TrimSpace(input.ClientID) == "" {
		token, err := security.RandomToken(12)
		if err != nil {
			return model.WorkoutSet{}, err
		}
		input.ClientID = token
	}

	var created model.WorkoutSet
	err := s.repo.Update(ctx, func(state *model.State) error {
		item, workout, ok := findWorkoutExerciseWithWorkout(state, input.ExerciseID)
		if !ok || workout.UserID != input.UserID {
			return apperrors.NotFound("упражнение тренировки не найдено")
		}
		if workout.Status != model.WorkoutStatusActive {
			return apperrors.BadRequest("нельзя менять завершенную тренировку")
		}

		for _, set := range state.Sets {
			if set.WorkoutExerciseID == input.ExerciseID && set.ClientID == input.ClientID {
				created = set
				return nil
			}
		}

		values, err := domainservice.NormalizeMetricValues(item.ExerciseSnapshot.MetricSchema, input.MetricValues)
		if err != nil {
			return err
		}

		setIndex := 1
		for _, set := range state.Sets {
			if set.WorkoutExerciseID == input.ExerciseID && set.SetIndex >= setIndex {
				setIndex = set.SetIndex + 1
			}
		}

		now := nowUTC()
		created = model.WorkoutSet{
			ID:                state.NextSetID,
			WorkoutExerciseID: input.ExerciseID,
			ClientID:          input.ClientID,
			SetIndex:          setIndex,
			MetricValues:      values,
			Notes:             strings.TrimSpace(input.Notes),
			CompletedAt:       now,
			CreatedAt:         now,
			UpdatedAt:         now,
		}
		state.NextSetID++
		state.Sets = append(state.Sets, created)
		touchWorkout(state, workout.ID)
		return nil
	})
	return created, err
}

func (s *WorkoutService) UpdateSet(ctx context.Context, input UpdateSetInput) (model.WorkoutSet, error) {
	var updated model.WorkoutSet
	err := s.repo.Update(ctx, func(state *model.State) error {
		for i := range state.Sets {
			if state.Sets[i].ID != input.ID {
				continue
			}
			item, workout, ok := findWorkoutExerciseWithWorkout(state, state.Sets[i].WorkoutExerciseID)
			if !ok || workout.UserID != input.UserID {
				return apperrors.NotFound("подход не найден")
			}
			if workout.Status != model.WorkoutStatusActive {
				return apperrors.BadRequest("нельзя менять завершенную тренировку")
			}
			if input.MetricValues != nil {
				values, err := domainservice.NormalizeMetricValues(item.ExerciseSnapshot.MetricSchema, *input.MetricValues)
				if err != nil {
					return err
				}
				state.Sets[i].MetricValues = values
			}
			if input.Notes != nil {
				state.Sets[i].Notes = strings.TrimSpace(*input.Notes)
			}
			state.Sets[i].UpdatedAt = nowUTC()
			touchWorkout(state, workout.ID)
			updated = state.Sets[i]
			return nil
		}
		return apperrors.NotFound("подход не найден")
	})
	return updated, err
}

func (s *WorkoutService) DeleteSet(ctx context.Context, userID, setID int64) error {
	return s.repo.Update(ctx, func(state *model.State) error {
		for i, set := range state.Sets {
			if set.ID != setID {
				continue
			}
			_, workout, ok := findWorkoutExerciseWithWorkout(state, set.WorkoutExerciseID)
			if !ok || workout.UserID != userID {
				return apperrors.NotFound("подход не найден")
			}
			if workout.Status != model.WorkoutStatusActive {
				return apperrors.BadRequest("нельзя менять завершенную тренировку")
			}
			state.Sets = append(state.Sets[:i], state.Sets[i+1:]...)
			reindexSets(state, set.WorkoutExerciseID)
			touchWorkout(state, workout.ID)
			return nil
		}
		return apperrors.NotFound("подход не найден")
	})
}

func (s *WorkoutService) CreateShareLink(ctx context.Context, userID, workoutID int64) (model.ShareLink, error) {
	token, err := security.RandomToken(24)
	if err != nil {
		return model.ShareLink{}, err
	}

	var link model.ShareLink
	err = s.repo.Update(ctx, func(state *model.State) error {
		workout, ok := findWorkout(state, workoutID)
		if !ok || workout.UserID != userID {
			return apperrors.NotFound("тренировка не найдена")
		}
		link = model.ShareLink{
			ID:        state.NextShareLinkID,
			WorkoutID: workoutID,
			Token:     token,
			TokenHash: security.TokenHash(token),
			IsActive:  true,
			CreatedAt: nowUTC(),
		}
		state.NextShareLinkID++
		state.ShareLinks = append(state.ShareLinks, link)
		return nil
	})
	return link, err
}

func (s *WorkoutService) DisableShareLink(ctx context.Context, userID, linkID int64) error {
	return s.repo.Update(ctx, func(state *model.State) error {
		for i := range state.ShareLinks {
			if state.ShareLinks[i].ID != linkID {
				continue
			}
			workout, ok := findWorkout(state, state.ShareLinks[i].WorkoutID)
			if !ok || workout.UserID != userID {
				return apperrors.NotFound("ссылка не найдена")
			}
			state.ShareLinks[i].IsActive = false
			return nil
		}
		return apperrors.NotFound("ссылка не найдена")
	})
}

func (s *WorkoutService) PublicWorkout(ctx context.Context, token string) (model.WorkoutDetail, error) {
	hash := security.TokenHash(strings.TrimSpace(token))
	var detail model.WorkoutDetail
	err := s.repo.View(ctx, func(state model.State) error {
		for _, link := range state.ShareLinks {
			if link.TokenHash != hash || !link.IsActive {
				continue
			}
			if link.ExpiresAt != nil && link.ExpiresAt.Before(nowUTC()) {
				return apperrors.NotFound("ссылка устарела")
			}
			workout, ok := findWorkout(&state, link.WorkoutID)
			if !ok {
				return apperrors.NotFound("тренировка не найдена")
			}
			detail = workoutDetail(&state, workout)
			return nil
		}
		return apperrors.NotFound("ссылка не найдена")
	})
	return detail, err
}

func OrderedMetricKeys(detail model.WorkoutDetail) []string {
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

func MetricToString(value interface{}) string {
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
