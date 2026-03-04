package main

import "errors"

// Tool: create_task(name)
func (s *Store) CreateTask(name string) Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	t := Task{
		ID:   s.nextID,
		Name: name,
		Done: false,
	}
	s.nextID++
	s.tasks = append(s.tasks, t)
	return t
}

// Tool: list_tasks()
func (s *Store) ListTasks() []Task {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]Task, len(s.tasks))
	copy(out, s.tasks)
	return out
}

// Tool: get_task(id)
func (s *Store) GetTask(id int) (Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, t := range s.tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return Task{}, errors.New("task not found")
}

// Tool: mark_done(id)
func (s *Store) MarkDone(id int) (Task, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.tasks {
		if s.tasks[i].ID == id {
			s.tasks[i].Done = true
			return s.tasks[i], nil
		}
	}
	return Task{}, errors.New("task not found")
}

func (s *Store) AddMemory(entry string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.memory.History = append(s.memory.History, entry)
}

func (s *Store) MemorySnapshot() AgentMemory {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return AgentMemory{History: append([]string{}, s.memory.History...)}
}