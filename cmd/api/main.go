package main

import (
	"log"
	"net/http"
	"os"

	"github.com/luismgoyone/go-practice/internal/handler"
	"github.com/luismgoyone/go-practice/internal/store"
)

func main() {
	mem := store.NewMemoryStore()

	http.HandleFunc("GET /health", handler.HealthHandler)
	http.HandleFunc("POST /projects", handler.CreateProjectHandler(mem))
	http.HandleFunc("GET /projects", handler.ListProjectsHandler(mem))
	http.HandleFunc("GET /projects/{id}", handler.GetProjectHandler(mem))
	http.HandleFunc("GET /projects/{id}/tasks", handler.ListTasksByProjectHandler(mem))
	http.HandleFunc("POST /projects/{id}/tasks", handler.CreateTaskHandler(mem))
	http.HandleFunc("GET /tasks/{id}", handler.GetTaskHandler(mem))
	http.HandleFunc("PATCH /projects/{id}", handler.UpdateProjectHandler(mem))
	http.HandleFunc("DELETE /projects/{id}", handler.DeleteProjectHandler(mem))
	http.HandleFunc("PATCH /tasks/{id}", handler.UpdateTaskHandler(mem))
	http.HandleFunc("DELETE /tasks/{id}", handler.DeleteTaskHandler(mem))
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
