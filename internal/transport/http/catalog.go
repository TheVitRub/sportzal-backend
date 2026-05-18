package httptransport

import (
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"workout-app/backend/internal/application"
	"workout-app/backend/internal/domain/model"
	"workout-app/backend/pkg/security"
)

func (h *Handler) listCategories(w http.ResponseWriter, r *http.Request) {
	categories, err := h.catalogService.ListCategories(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (h *Handler) createCategory(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int    `json:"sortOrder"`
		IsActive    *bool  `json:"isActive"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}

	category, err := h.catalogService.CreateCategory(r.Context(), application.CreateCategoryInput(req))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

func (h *Handler) updateCategory(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		SortOrder   *int    `json:"sortOrder"`
		IsActive    *bool   `json:"isActive"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}

	category, err := h.catalogService.UpdateCategory(r.Context(), id, application.UpdateCategoryInput(req))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, category)
}

func (h *Handler) listExercises(w http.ResponseWriter, r *http.Request) {
	exercises, err := h.catalogService.ListExercises(r.Context(), r.URL.Query().Get("includeInactive") == "1")
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, exercises)
}

func (h *Handler) getExercise(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	exercise, err := h.catalogService.GetExercise(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, exercise)
}

func (h *Handler) createExercise(w http.ResponseWriter, r *http.Request) {
	var req struct {
		CategoryID   int64              `json:"categoryId"`
		Title        string             `json:"title"`
		Description  string             `json:"description"`
		MetricSchema model.MetricSchema `json:"metricSchema"`
		IsActive     *bool              `json:"isActive"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}

	user := currentUser(r)
	exercise, err := h.catalogService.CreateExercise(r.Context(), application.CreateExerciseInput{
		CategoryID:   req.CategoryID,
		Title:        req.Title,
		Description:  req.Description,
		MetricSchema: req.MetricSchema,
		IsActive:     req.IsActive,
		CreatedBy:    user.ID,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, exercise)
}

func (h *Handler) updateExercise(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		CategoryID   *int64              `json:"categoryId"`
		Title        *string             `json:"title"`
		Description  *string             `json:"description"`
		MetricSchema *model.MetricSchema `json:"metricSchema"`
		IsActive     *bool               `json:"isActive"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}

	exercise, err := h.catalogService.UpdateExercise(r.Context(), id, application.UpdateExerciseInput(req))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, exercise)
}

func (h *Handler) uploadExerciseMedia(w http.ResponseWriter, r *http.Request) {
	exerciseID, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 80<<20)
	if err := r.ParseMultipartForm(82 << 20); err != nil {
		writeBadRequest(w, "не удалось прочитать файл")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		writeBadRequest(w, "поле file обязательно")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	if mimeType == "" || mimeType == "application/octet-stream" {
		if guessed := mime.TypeByExtension(strings.ToLower(filepath.Ext(header.Filename))); guessed != "" {
			mimeType = guessed
		}
	}
	if _, ok := application.MediaKind(mimeType); !ok {
		writeBadRequest(w, "поддерживаются только изображения и видео")
		return
	}

	nameToken, err := security.RandomToken(18)
	if err != nil {
		writeError(w, err)
		return
	}
	fileName := fmt.Sprintf("%d_%s%s", exerciseID, nameToken, application.ExtensionByMime(mimeType, header.Filename))
	dstPath := filepath.Join(h.uploadDir, fileName)
	dst, err := os.Create(dstPath)
	if err != nil {
		writeError(w, err)
		return
	}
	size, copyErr := io.Copy(dst, file)
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(dstPath)
		writeBadRequest(w, "не удалось сохранить файл")
		return
	}

	media, err := h.catalogService.AddMedia(r.Context(), application.AddMediaInput{
		ExerciseID: exerciseID,
		MimeType:   mimeType,
		SizeBytes:  size,
		FileURL:    "/uploads/" + fileName,
	})
	if err != nil {
		_ = os.Remove(dstPath)
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, media)
}

func (h *Handler) deleteExerciseMedia(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	media, err := h.catalogService.DeleteMedia(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	if strings.HasPrefix(media.FileURL, "/uploads/") {
		_ = os.Remove(filepath.Join(h.uploadDir, strings.TrimPrefix(media.FileURL, "/uploads/")))
	}
	w.WriteHeader(http.StatusNoContent)
}
