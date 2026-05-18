package application

import (
	"context"
	"mime"
	"path/filepath"
	"sort"
	"strings"

	"workout-app/backend/internal/domain/apperrors"
	"workout-app/backend/internal/domain/interfaces"
	"workout-app/backend/internal/domain/model"
	domainservice "workout-app/backend/internal/domain/service"
)

type CatalogService struct {
	repo interfaces.StateRepository
}

type CreateCategoryInput struct {
	Name        string
	Description string
	SortOrder   int
	IsActive    *bool
}

type UpdateCategoryInput struct {
	Name        *string
	Description *string
	SortOrder   *int
	IsActive    *bool
}

type CreateExerciseInput struct {
	CategoryID   int64
	Title        string
	Description  string
	MetricSchema model.MetricSchema
	IsActive     *bool
	CreatedBy    int64
}

type UpdateExerciseInput struct {
	CategoryID   *int64
	Title        *string
	Description  *string
	MetricSchema *model.MetricSchema
	IsActive     *bool
}

type AddMediaInput struct {
	ExerciseID int64
	MimeType   string
	SizeBytes  int64
	FileURL    string
}

func NewCatalogService(repo interfaces.StateRepository) *CatalogService {
	return &CatalogService{repo: repo}
}

func (s *CatalogService) ListCategories(ctx context.Context) ([]model.ExerciseCategory, error) {
	var categories []model.ExerciseCategory
	err := s.repo.View(ctx, func(state model.State) error {
		categories = append([]model.ExerciseCategory(nil), state.Categories...)
		sort.Slice(categories, func(i, j int) bool {
			if categories[i].SortOrder == categories[j].SortOrder {
				return categories[i].Name < categories[j].Name
			}
			return categories[i].SortOrder < categories[j].SortOrder
		})
		return nil
	})
	return categories, err
}

func (s *CatalogService) CreateCategory(ctx context.Context, input CreateCategoryInput) (model.ExerciseCategory, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return model.ExerciseCategory{}, apperrors.BadRequest("название категории обязательно")
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}

	var created model.ExerciseCategory
	err := s.repo.Update(ctx, func(state *model.State) error {
		now := nowUTC()
		created = model.ExerciseCategory{
			ID:          state.NextCategoryID,
			Name:        name,
			Description: strings.TrimSpace(input.Description),
			SortOrder:   input.SortOrder,
			IsActive:    active,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		state.NextCategoryID++
		state.Categories = append(state.Categories, created)
		return nil
	})
	return created, err
}

func (s *CatalogService) UpdateCategory(ctx context.Context, id int64, input UpdateCategoryInput) (model.ExerciseCategory, error) {
	var updated model.ExerciseCategory
	err := s.repo.Update(ctx, func(state *model.State) error {
		for i := range state.Categories {
			if state.Categories[i].ID != id {
				continue
			}
			if input.Name != nil {
				name := strings.TrimSpace(*input.Name)
				if name == "" {
					return apperrors.BadRequest("название категории обязательно")
				}
				state.Categories[i].Name = name
			}
			if input.Description != nil {
				state.Categories[i].Description = strings.TrimSpace(*input.Description)
			}
			if input.SortOrder != nil {
				state.Categories[i].SortOrder = *input.SortOrder
			}
			if input.IsActive != nil {
				state.Categories[i].IsActive = *input.IsActive
			}
			state.Categories[i].UpdatedAt = nowUTC()
			updated = state.Categories[i]
			return nil
		}
		return apperrors.NotFound("категория не найдена")
	})
	return updated, err
}

func (s *CatalogService) ListExercises(ctx context.Context, includeInactive bool) ([]model.ExerciseView, error) {
	var exercises []model.ExerciseView
	err := s.repo.View(ctx, func(state model.State) error {
		for _, exercise := range state.Exercises {
			if !includeInactive && !exercise.IsActive {
				continue
			}
			exercises = append(exercises, exerciseView(&state, exercise))
		}
		sort.Slice(exercises, func(i, j int) bool {
			if exercises[i].CategoryName == exercises[j].CategoryName {
				return exercises[i].Title < exercises[j].Title
			}
			return exercises[i].CategoryName < exercises[j].CategoryName
		})
		return nil
	})
	return exercises, err
}

func (s *CatalogService) GetExercise(ctx context.Context, id int64) (model.ExerciseView, error) {
	var view model.ExerciseView
	err := s.repo.View(ctx, func(state model.State) error {
		exercise, ok := findExercise(&state, id)
		if !ok {
			return apperrors.NotFound("упражнение не найдено")
		}
		view = exerciseView(&state, exercise)
		return nil
	})
	return view, err
}

func (s *CatalogService) CreateExercise(ctx context.Context, input CreateExerciseInput) (model.ExerciseView, error) {
	if err := validateExerciseInput(input.CategoryID, input.Title, input.MetricSchema); err != nil {
		return model.ExerciseView{}, err
	}
	active := true
	if input.IsActive != nil {
		active = *input.IsActive
	}

	var created model.ExerciseView
	err := s.repo.Update(ctx, func(state *model.State) error {
		if _, ok := findCategory(state, input.CategoryID); !ok {
			return apperrors.BadRequest("категория не найдена")
		}
		now := nowUTC()
		exercise := model.ExerciseTemplate{
			ID:           state.NextExerciseID,
			CategoryID:   input.CategoryID,
			Title:        strings.TrimSpace(input.Title),
			Description:  strings.TrimSpace(input.Description),
			MetricSchema: input.MetricSchema,
			IsActive:     active,
			CreatedBy:    input.CreatedBy,
			CreatedAt:    now,
			UpdatedAt:    now,
		}
		state.NextExerciseID++
		state.Exercises = append(state.Exercises, exercise)
		created = exerciseView(state, exercise)
		return nil
	})
	return created, err
}

func (s *CatalogService) UpdateExercise(ctx context.Context, id int64, input UpdateExerciseInput) (model.ExerciseView, error) {
	var updated model.ExerciseView
	err := s.repo.Update(ctx, func(state *model.State) error {
		for i := range state.Exercises {
			if state.Exercises[i].ID != id {
				continue
			}

			categoryID := state.Exercises[i].CategoryID
			title := state.Exercises[i].Title
			schema := state.Exercises[i].MetricSchema
			if input.CategoryID != nil {
				categoryID = *input.CategoryID
			}
			if input.Title != nil {
				title = *input.Title
			}
			if input.MetricSchema != nil {
				schema = *input.MetricSchema
			}
			if err := validateExerciseInput(categoryID, title, schema); err != nil {
				return err
			}
			if _, ok := findCategory(state, categoryID); !ok {
				return apperrors.BadRequest("категория не найдена")
			}

			state.Exercises[i].CategoryID = categoryID
			state.Exercises[i].Title = strings.TrimSpace(title)
			if input.Description != nil {
				state.Exercises[i].Description = strings.TrimSpace(*input.Description)
			}
			state.Exercises[i].MetricSchema = schema
			if input.IsActive != nil {
				state.Exercises[i].IsActive = *input.IsActive
			}
			state.Exercises[i].UpdatedAt = nowUTC()
			updated = exerciseView(state, state.Exercises[i])
			return nil
		}
		return apperrors.NotFound("упражнение не найдено")
	})
	return updated, err
}

func (s *CatalogService) AddMedia(ctx context.Context, input AddMediaInput) (model.ExerciseMedia, error) {
	mediaType, ok := MediaKind(input.MimeType)
	if !ok {
		return model.ExerciseMedia{}, apperrors.BadRequest("поддерживаются только изображения и видео")
	}
	if strings.TrimSpace(input.FileURL) == "" {
		return model.ExerciseMedia{}, apperrors.BadRequest("file_url обязателен")
	}

	var created model.ExerciseMedia
	err := s.repo.Update(ctx, func(state *model.State) error {
		if _, ok := findExercise(state, input.ExerciseID); !ok {
			return apperrors.NotFound("упражнение не найдено")
		}
		created = model.ExerciseMedia{
			ID:         state.NextMediaID,
			ExerciseID: input.ExerciseID,
			MediaType:  mediaType,
			FileURL:    input.FileURL,
			MimeType:   input.MimeType,
			SizeBytes:  input.SizeBytes,
			CreatedAt:  nowUTC(),
		}
		state.NextMediaID++
		state.Media = append(state.Media, created)
		return nil
	})
	return created, err
}

func (s *CatalogService) DeleteMedia(ctx context.Context, id int64) (model.ExerciseMedia, error) {
	var deleted model.ExerciseMedia
	err := s.repo.Update(ctx, func(state *model.State) error {
		for i, media := range state.Media {
			if media.ID != id {
				continue
			}
			deleted = media
			state.Media = append(state.Media[:i], state.Media[i+1:]...)
			return nil
		}
		return apperrors.NotFound("медиа не найдено")
	})
	return deleted, err
}

func validateExerciseInput(categoryID int64, title string, schema model.MetricSchema) error {
	if categoryID <= 0 {
		return apperrors.BadRequest("категория обязательна")
	}
	if strings.TrimSpace(title) == "" {
		return apperrors.BadRequest("название упражнения обязательно")
	}
	return domainservice.ValidateMetricSchema(schema)
}

func MediaKind(mimeType string) (string, bool) {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return "photo", true
	case strings.HasPrefix(mimeType, "video/"):
		return "video", true
	default:
		return "", false
	}
}

func ExtensionByMime(mimeType, originalName string) string {
	if ext := strings.ToLower(filepath.Ext(originalName)); ext != "" {
		return ext
	}
	if ext, err := mime.ExtensionsByType(mimeType); err == nil && len(ext) > 0 {
		return ext[0]
	}
	if strings.HasPrefix(mimeType, "video/") {
		return ".mp4"
	}
	return ".bin"
}
