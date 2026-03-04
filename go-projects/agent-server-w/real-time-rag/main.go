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
	webRAG := NewWebRAG(openai)

	agent := NewAgent(store, openai, retriever, webRAG)
	server := NewServer(store, agent, ingester, retriever)

	http.HandleFunc("/tasks", server.tasksHandler)
	http.HandleFunc("/agent", server.agentHandler)
	http.HandleFunc("/ingest", server.ingestHandler)
	http.HandleFunc("/search", server.searchHandler)

	fmt.Println("Server running on http://localhost:8080")
	fmt.Println("  POST /agent   — chat (auto web-search if no local docs)")
	fmt.Println("  POST /ingest  — add documents to local knowledge base")
	fmt.Println("  POST /search  — search the local knowledge base")
	fmt.Println("  GET  /tasks   — list tasks")
	_ = http.ListenAndServe(":8080", nil)
}
