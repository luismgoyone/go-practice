package store

import (
	"sync"

	"github.com/luismgoyone/go-practice/internal/model"
)

type MemoryStore struct {
	mu       sync.Mutex
	projects map[string]model.Project
	tasks    map[string]model.Task
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		projects: make(map[string]model.Project),
		tasks:    make(map[string]model.Task),
	}
}

func (s *MemoryStore) CreateProject(p model.Project) model.Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.projects[p.ID] = p
	return p
}

func (s *MemoryStore) ListProjects() []model.Project {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Project, 0, len(s.projects))
	for _, p := range s.projects {
		out = append(out, p)
	}
	return out
}

func (s *MemoryStore) GetProject(id string) (model.Project, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.projects[id]
	return p, ok
}

func (s *MemoryStore) UpdateProject(p model.Project) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[p.ID]; !ok {
		return false
	}
	s.projects[p.ID] = p
	return true
}

func (s *MemoryStore) DeleteProject(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.projects[id]; !ok {
		return false
	}
	delete(s.projects, id)
	for tid, t := range s.tasks {
		if t.ProjectID == id {
			delete(s.tasks, tid)
		}
	}
	return true
}

func (s *MemoryStore) CreateTask(t model.Task) model.Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[t.ID] = t
	return t
}

func (s *MemoryStore) ListTasksByProject(projectID string) []model.Task {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]model.Task, 0)
	for _, t := range s.tasks {
		if t.ProjectID == projectID {
			out = append(out, t)
		}
	}
	return out
}

func (s *MemoryStore) GetTask(id string) (model.Task, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.tasks[id]
	return t, ok
}

func (s *MemoryStore) UpdateTask(t model.Task) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[t.ID]; !ok {
		return false
	}
	s.tasks[t.ID] = t
	return true
}

func (s *MemoryStore) DeleteTask(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[id]; !ok {
		return false
	}
	delete(s.tasks, id)
	return true
}
