package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
)

type Task struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Done bool   `json:"done"`
}

var tasks = []Task{}
var nextID = 1

func main() {
	http.HandleFunc("/tasks", tasksHandler)

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}

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

		newTask.ID = nextID
		nextID++
		tasks = append(tasks, newTask)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(newTask)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}