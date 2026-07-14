package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/luismgoyone/go-practice/internal/model"
	"github.com/luismgoyone/go-practice/internal/store"
)

func validStatus(s string) bool {
	return s == "todo" || s == "in_progress" || s == "done"
}

func ListTasksByProjectHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("id")
		if _, ok := mem.GetProject(projectID); !ok {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		tasks := mem.ListTasksByProject(projectID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200
		_ = json.NewEncoder(w).Encode(tasks)
	}
}

func CreateTaskHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("id")
		if _, ok := mem.GetProject(projectID); !ok {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		var req struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}
		defer r.Body.Close()

		if req.Title == "" {
			writeError(w, http.StatusBadRequest, "title is required")
			return
		}

		if req.Status == "" {
			req.Status = "todo"
		}

		if !validStatus(req.Status) {
			writeError(w, http.StatusBadRequest, "status must be todo, doing, or done")
			return
		}

		now := time.Now().UTC()
		task := model.Task{
			ID:        newID(),
			ProjectID: projectID, // from path, not body
			Title:     req.Title,
			Status:    req.Status,
			CreatedAt: now,
			UpdatedAt: now,
		}

		task = mem.CreateTask(task)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // 201
		_ = json.NewEncoder(w).Encode(task)
	}
}

func GetTaskHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := r.PathValue("id")
		task, ok := mem.GetTask(taskID)

		if !ok {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(task)
	}
}

func UpdateTaskHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := r.PathValue("id")
		task, ok := mem.GetTask(taskID)

		if !ok {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}

		var req struct {
			Title  string `json:"title"`
			Status string `json:"status"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}

		defer r.Body.Close()
		if req.Title != "" {
			task.Title = req.Title
		}

		if !validStatus(req.Status) {
			writeError(w, http.StatusBadRequest, "status must be todo, doing, or done")
			return
		}

		task.Status = req.Status
		task.UpdatedAt = time.Now().UTC()

		if !mem.UpdateTask(task) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(task)
	}
}

func DeleteTaskHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		taskID := r.PathValue("id")
		if !mem.DeleteTask(taskID) {
			writeError(w, http.StatusNotFound, "task not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
