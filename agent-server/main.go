package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/tasks", tasksHandler)
	http.HandleFunc("/agent", agentHandler)

	fmt.Println("Server running on http://localhost:8080")
	http.ListenAndServe(":8080", nil)
}