package handler

import (
	"encoding/json"
	"fmt"
	"log"
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

func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	}
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

