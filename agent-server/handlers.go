package main

import (
	"encoding/json"
	"net/http"
	"strconv"
)

func tasksHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {

	case http.MethodGet:
		idStr := r.URL.Query().Get("id")

		if idStr == "" {
			json.NewEncoder(w).Encode(tasks)
			return
		}

		id, _ := strconv.Atoi(idStr)

		for _, t := range tasks {
			if t.ID == id {
				json.NewEncoder(w).Encode(t)
				return
			}
		}

		http.Error(w, "Task not found", http.StatusNotFound)

	case http.MethodPost:
		var newTask Task

		err := json.NewDecoder(r.Body).Decode(&newTask)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		created := CreateTask(newTask.Name)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(created)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func agentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var input struct {
		Goal string `json:"goal"`
	}

	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	response := RunAgent(input.Goal)
	json.NewEncoder(w).Encode(response)
}