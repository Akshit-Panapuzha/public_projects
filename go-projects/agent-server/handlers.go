package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

type Server struct {
	store *Store
	agent *Agent
}

func NewServer(store *Store, agent *Agent) *Server {
	return &Server{store: store, agent: agent}
}

func (sv *Server) tasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			json.NewEncoder(w).Encode(sv.store.ListTasks())
			return
		}

		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		t, err := sv.store.GetTask(id)
		if err != nil {
			http.Error(w, "Task not found", http.StatusNotFound)
			return
		}
		json.NewEncoder(w).Encode(t)

	case http.MethodPost:
		var in struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Name == "" {
			http.Error(w, "Invalid JSON (need {\"name\":\"...\"})", http.StatusBadRequest)
			return
		}
		t := sv.store.CreateTask(in.Name)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(t)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (sv *Server) agentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var in struct {
		Goal string `json:"goal"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Goal == "" {
		http.Error(w, "Invalid JSON (need {\"goal\":\"...\"})", http.StatusBadRequest)
		return
	}

	out, err := sv.agent.Run(in.Goal)
	if err != nil {
		http.Error(w, "agent error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(out)
}