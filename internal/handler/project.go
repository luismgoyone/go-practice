package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/luismgoyone/go-practice/internal/model"
	"github.com/luismgoyone/go-practice/internal/store"
)

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func newID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func CreateProjectHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		if req.Name == "" {
			writeError(w, http.StatusBadRequest, "name is required")
			return
		}

		now := time.Now().UTC()
		project := model.Project{
			ID:          newID(),
			Name:        req.Name,
			Description: req.Description,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		project = mem.CreateProject(project)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // 201
		_ = json.NewEncoder(w).Encode(project)
	}
}

func ListProjectsHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projects := mem.ListProjects()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200
		_ = json.NewEncoder(w).Encode(projects)
	}
}

func GetProjectHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		project, ok := mem.GetProject(id)
		if !ok {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // 200
		_ = json.NewEncoder(w).Encode(project)
	}
}

func UpdateProjectHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("id")
		project, ok := mem.GetProject(projectID)
		if !ok {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid JSON")
			return
		}
		defer r.Body.Close()

		if req.Name != "" {
			project.Name = req.Name
		}

		project.Description = req.Description
		project.UpdatedAt = time.Now().UTC()

		if !mem.UpdateProject(project) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(project)
	}
}

func DeleteProjectHandler(mem *store.MemoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectID := r.PathValue("id")
		if !mem.DeleteProject(projectID) {
			writeError(w, http.StatusNotFound, "project not found")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
