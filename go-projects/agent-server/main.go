package main

import (
	"fmt"
	"net/http"
)

func main() {
	store := NewStore()

	openai, err := NewOpenAIClientFromEnv()
	if err != nil {
		panic(err)
	}

	agent := NewAgent(store, openai)
	server := NewServer(store, agent)

	http.HandleFunc("/tasks", server.tasksHandler)
	http.HandleFunc("/agent", server.agentHandler)

	fmt.Println("Server running on http://localhost:8080")
	_ = http.ListenAndServe(":8080", nil)
}