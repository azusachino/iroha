package httpapi

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/azusachino/iroha/apps/iroha-runtime/ids"
	"github.com/azusachino/iroha/apps/iroha-runtime/models"
	"github.com/azusachino/iroha/apps/iroha-server/pkg/tasks"
	"github.com/go-chi/chi/v5"
)

type taskResponse struct {
	ID          string     `json:"id"`
	Title       string     `json:"title"`
	Notes       string     `json:"notes,omitempty"`
	Status      string     `json:"status"`
	DueDate     *string    `json:"due_date,omitempty"`
	Priority    int        `json:"priority"`
	Source      string     `json:"source"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type createTaskRequest struct {
	Title    string  `json:"title"`
	Notes    string  `json:"notes"`
	DueDate  *string `json:"due_date"`
	Priority int     `json:"priority"`
}

type updateTaskRequest struct {
	Status string `json:"status"`
}

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	if s.deps.TaskService == nil {
		writeError(w, http.StatusServiceUnavailable, "task service unavailable")
		return
	}
	limit := 50
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed <= 0 {
			writeError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = parsed
	}
	status := r.URL.Query().Get("status")
	if status != "" && status != tasks.StatusOpen && status != tasks.StatusCompleted && status != tasks.StatusCanceled {
		writeError(w, http.StatusBadRequest, "invalid status")
		return
	}
	var dueOn *time.Time
	if value := r.URL.Query().Get("due"); value != "" {
		parsed, err := time.Parse("2006-01-02", value)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid due date")
			return
		}
		dueOn = &parsed
	}
	rows, err := s.deps.TaskService.List(tasks.ListFilters{Status: status, DueOn: dueOn, Limit: limit})
	if err != nil {
		s.deps.Logger.Error("list tasks", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to list tasks")
		return
	}
	response := make([]taskResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, toTaskResponse(row))
	}
	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	if s.deps.TaskService == nil {
		writeError(w, http.StatusServiceUnavailable, "task service unavailable")
		return
	}
	var request createTaskRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	var dueDate *time.Time
	if request.DueDate != nil && strings.TrimSpace(*request.DueDate) != "" {
		parsed, err := time.Parse("2006-01-02", strings.TrimSpace(*request.DueDate))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid due date")
			return
		}
		dueDate = &parsed
	}
	task, err := s.deps.TaskService.Create(tasks.CreateInput{
		Title: request.Title, Notes: request.Notes, DueDate: dueDate, Priority: request.Priority,
	})
	if err != nil {
		if err == tasks.ErrEmptyTitle {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		s.deps.Logger.Error("create task", "error", err)
		writeError(w, http.StatusInternalServerError, "failed to create task")
		return
	}
	writeJSON(w, http.StatusCreated, toTaskResponse(task))
}

func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request) {
	if s.deps.TaskService == nil {
		writeError(w, http.StatusServiceUnavailable, "task service unavailable")
		return
	}
	id, err := ids.Decode(ids.TaskPrefix, chi.URLParam(r, "taskId"))
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid task id")
		return
	}
	var request updateTaskRequest
	if err := decodeJSONBody(w, r, &request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	var task models.Task
	switch request.Status {
	case tasks.StatusCompleted:
		task, err = s.deps.TaskService.Complete(id)
	case tasks.StatusCanceled:
		task, err = s.deps.TaskService.Cancel(id)
	default:
		writeError(w, http.StatusBadRequest, "unsupported task status")
		return
	}
	if err != nil {
		if err == tasks.ErrNotFound {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to update task")
		return
	}
	writeJSON(w, http.StatusOK, toTaskResponse(task))
}

func toTaskResponse(task models.Task) taskResponse {
	var dueDate *string
	if task.DueDate != nil {
		value := task.DueDate.UTC().Format("2006-01-02")
		dueDate = &value
	}
	return taskResponse{ID: ids.Encode(ids.TaskPrefix, task.ID), Title: task.Title, Notes: task.Notes, Status: task.Status, DueDate: dueDate, Priority: task.Priority, Source: task.Source, CompletedAt: task.CompletedAt, CreatedAt: task.CreatedAt.UTC(), UpdatedAt: task.UpdatedAt.UTC()}
}
