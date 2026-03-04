package main

import (
	"math"
	"sort"
	"strings"
	"sync"
)

type Document struct {
	ID       string    `json:"id"`
	Content  string    `json:"content"`
	Metadata string    `json:"metadata,omitempty"`
	Embedding []float64 `json:"-"`
}

type Chunk struct {
	DocID     string    `json:"doc_id"`
	Index     int       `json:"index"`
	Text      string    `json:"text"`
	Embedding []float64 `json:"-"`
}

type SearchResult struct {
	Chunk Chunk   `json:"chunk"`
	Score float64 `json:"score"`
}

type KnowledgeBase struct {
	mu     sync.RWMutex
	chunks []Chunk
}

func NewKnowledgeBase() *KnowledgeBase {
	return &KnowledgeBase{
		chunks: []Chunk{},
	}
}

func (kb *KnowledgeBase) AddChunks(chunks []Chunk) {
	kb.mu.Lock()
	defer kb.mu.Unlock()
	kb.chunks = append(kb.chunks, chunks...)
}

func (kb *KnowledgeBase) Search(queryEmbedding []float64, topK int) []SearchResult {
	kb.mu.RLock()
	defer kb.mu.RUnlock()

	results := make([]SearchResult, 0, len(kb.chunks))
	for _, c := range kb.chunks {
		score := cosineSimilarity(queryEmbedding, c.Embedding)
		results = append(results, SearchResult{Chunk: c, Score: score})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	if topK > len(results) {
		topK = len(results)
	}
	return results[:topK]
}

func (kb *KnowledgeBase) ChunkCount() int {
	kb.mu.RLock()
	defer kb.mu.RUnlock()
	return len(kb.chunks)
}

// ChunkText splits a document into overlapping chunks of roughly chunkSize
// characters with overlap characters of overlap between consecutive chunks.
func ChunkText(text string, chunkSize, overlap int) []string {
	text = strings.TrimSpace(text)
	if len(text) == 0 {
		return nil
	}
	if chunkSize <= 0 {
		chunkSize = 500
	}
	if overlap < 0 || overlap >= chunkSize {
		overlap = chunkSize / 5
	}

	var chunks []string
	start := 0
	for start < len(text) {
		end := start + chunkSize
		if end > len(text) {
			end = len(text)
		}
		chunks = append(chunks, text[start:end])
		start += chunkSize - overlap
	}
	return chunks
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}
	return dot / denom
}
