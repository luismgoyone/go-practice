package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/luismgoyone/go-practice/internal/handler"
	"github.com/luismgoyone/go-practice/internal/store"
)

func withLogging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next(w, r)
		log.Printf("%s %s %v", r.Method, r.URL.Path, time.Since(start))
	}
}

func main() {
	mem := store.NewMemoryStore()

	http.HandleFunc("GET /health", withLogging(handler.HealthHandler))
	http.HandleFunc("POST /projects", withLogging(handler.CreateProjectHandler(mem)))
	http.HandleFunc("GET /projects", withLogging(handler.ListProjectsHandler(mem)))
	http.HandleFunc("GET /projects/{id}", withLogging(handler.GetProjectHandler(mem)))
	http.HandleFunc("GET /projects/{id}/tasks", withLogging(handler.ListTasksByProjectHandler(mem)))
	http.HandleFunc("POST /projects/{id}/tasks", withLogging(handler.CreateTaskHandler(mem)))
	http.HandleFunc("GET /tasks/{id}", withLogging(handler.GetTaskHandler(mem)))
	http.HandleFunc("PATCH /projects/{id}", withLogging(handler.UpdateProjectHandler(mem)))
	http.HandleFunc("DELETE /projects/{id}", withLogging(handler.DeleteProjectHandler(mem)))
	http.HandleFunc("PATCH /tasks/{id}", withLogging(handler.UpdateTaskHandler(mem)))
	http.HandleFunc("DELETE /tasks/{id}", withLogging(handler.DeleteTaskHandler(mem)))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
