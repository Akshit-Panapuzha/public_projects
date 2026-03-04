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

	kb := NewKnowledgeBase()
	ingester := NewIngester(kb, openai)
	retriever := NewRetriever(kb, openai, 5)

	agent := NewAgent(store, openai, retriever)
	server := NewServer(store, agent, ingester, retriever)

	http.HandleFunc("/tasks", server.tasksHandler)
	http.HandleFunc("/agent", server.agentHandler)
	http.HandleFunc("/ingest", server.ingestHandler)
	http.HandleFunc("/search", server.searchHandler)

	fmt.Println("Server running on http://localhost:8080")
	fmt.Println("  POST /ingest  — add documents to the knowledge base")
	fmt.Println("  POST /search  — search the knowledge base")
	fmt.Println("  POST /agent   — chat with the RAG-augmented agent")
	fmt.Println("  GET  /tasks   — list tasks")
	_ = http.ListenAndServe(":8080", nil)
}
