package httptransport

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"workout-app/backend/internal/application"
)

func (h *Handler) createWorkout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title string `json:"title"`
		Notes string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil && err != io.EOF {
		writeBadRequest(w, "некорректный JSON")
		return
	}

	user := currentUser(r)
	workout, alreadyActive, err := h.workoutService.CreateWorkout(r.Context(), application.CreateWorkoutInput{
		UserID: user.ID,
		Title:  req.Title,
		Notes:  req.Notes,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	if alreadyActive {
		writeJSON(w, http.StatusOK, map[string]interface{}{"alreadyActive": true, "workout": workout})
		return
	}
	writeJSON(w, http.StatusCreated, workout)
}

func (h *Handler) activeWorkout(w http.ResponseWriter, r *http.Request) {
	workout, err := h.workoutService.ActiveWorkout(r.Context(), currentUser(r).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workout)
}

func (h *Handler) listWorkouts(w http.ResponseWriter, r *http.Request) {
	workouts, err := h.workoutService.ListWorkouts(r.Context(), currentUser(r).ID)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workouts)
}

func (h *Handler) getWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	workout, err := h.workoutService.GetWorkout(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workout)
}

func (h *Handler) updateWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Title *string `json:"title"`
		Notes *string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}
	workout, err := h.workoutService.UpdateWorkout(r.Context(), id, application.UpdateWorkoutInput{
		UserID: currentUser(r).ID,
		Title:  req.Title,
		Notes:  req.Notes,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workout)
}

func (h *Handler) finishWorkout(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	workout, err := h.workoutService.FinishWorkout(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workout)
}

func (h *Handler) addWorkoutExercise(w http.ResponseWriter, r *http.Request) {
	workoutID, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		ExerciseID int64  `json:"exerciseId"`
		Notes      string `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}
	workout, err := h.workoutService.AddWorkoutExercise(r.Context(), application.AddWorkoutExerciseInput{
		UserID:     currentUser(r).ID,
		WorkoutID:  workoutID,
		ExerciseID: req.ExerciseID,
		Notes:      req.Notes,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, workout)
}

func (h *Handler) updateWorkoutExercise(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Notes    *string `json:"notes"`
		Position *int    `json:"position"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}
	workout, err := h.workoutService.UpdateWorkoutExercise(r.Context(), application.UpdateWorkoutExerciseInput{
		UserID:   currentUser(r).ID,
		ID:       id,
		Notes:    req.Notes,
		Position: req.Position,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workout)
}

func (h *Handler) createSet(w http.ResponseWriter, r *http.Request) {
	exerciseID, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		ClientID     string                 `json:"clientId"`
		MetricValues map[string]interface{} `json:"metricValues"`
		Notes        string                 `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}
	set, err := h.workoutService.CreateSet(r.Context(), application.CreateSetInput{
		UserID:       currentUser(r).ID,
		ExerciseID:   exerciseID,
		ClientID:     req.ClientID,
		MetricValues: req.MetricValues,
		Notes:        req.Notes,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, set)
}

func (h *Handler) updateSet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		MetricValues *map[string]interface{} `json:"metricValues"`
		Notes        *string                 `json:"notes"`
	}
	if err := decodeJSON(r, &req); err != nil {
		writeBadRequest(w, "некорректный JSON")
		return
	}
	set, err := h.workoutService.UpdateSet(r.Context(), application.UpdateSetInput{
		UserID:       currentUser(r).ID,
		ID:           id,
		MetricValues: req.MetricValues,
		Notes:        req.Notes,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, set)
}

func (h *Handler) deleteSet(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.workoutService.DeleteSet(r.Context(), currentUser(r).ID, id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) createShareLink(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	link, err := h.workoutService.CreateShareLink(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, link)
}

func (h *Handler) disableShareLink(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := h.workoutService.DisableShareLink(r.Context(), currentUser(r).ID, id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) publicWorkout(w http.ResponseWriter, r *http.Request) {
	workout, err := h.workoutService.PublicWorkout(r.Context(), r.PathValue("token"))
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, workout)
}

func (h *Handler) exportJSON(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	workout, err := h.workoutService.GetWorkout(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"workout_%d.json\"", id))
	_ = json.NewEncoder(w).Encode(workout)
}

func (h *Handler) exportCSV(w http.ResponseWriter, r *http.Request) {
	id, err := pathID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	workout, err := h.workoutService.GetWorkout(r.Context(), currentUser(r).ID, id)
	if err != nil {
		writeError(w, err)
		return
	}

	fieldKeys := application.OrderedMetricKeys(workout)
	header := []string{"workout_id", "workout_title", "started_at", "finished_at", "exercise", "set_index"}
	header = append(header, fieldKeys...)
	header = append(header, "notes")

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"workout_%d.csv\"", id))
	writer := csv.NewWriter(w)
	_ = writer.Write(header)
	for _, exercise := range workout.Exercises {
		for _, set := range exercise.Sets {
			row := []string{
				strconv.FormatInt(workout.ID, 10),
				workout.Title,
				workout.StartedAt.Format(time.RFC3339),
				formatOptionalTime(workout.FinishedAt),
				exercise.ExerciseSnapshot.Title,
				strconv.Itoa(set.SetIndex),
			}
			for _, key := range fieldKeys {
				row = append(row, application.MetricToString(set.MetricValues[key]))
			}
			row = append(row, set.Notes)
			_ = writer.Write(row)
		}
	}
	writer.Flush()
}

func formatOptionalTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}
