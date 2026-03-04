package main

import "fmt"

const (
	defaultChunkSize    = 500
	defaultChunkOverlap = 100
	embeddingBatchSize  = 64
)

type Ingester struct {
	kb     *KnowledgeBase
	openai *OpenAIClient
}

func NewIngester(kb *KnowledgeBase, openai *OpenAIClient) *Ingester {
	return &Ingester{kb: kb, openai: openai}
}

func (ing *Ingester) Ingest(docID, content, metadata string) (int, error) {
	texts := ChunkText(content, defaultChunkSize, defaultChunkOverlap)
	if len(texts) == 0 {
		return 0, fmt.Errorf("document is empty after chunking")
	}

	chunks := make([]Chunk, len(texts))
	for i, t := range texts {
		chunks[i] = Chunk{
			DocID: docID,
			Index: i,
			Text:  t,
		}
	}

	for batchStart := 0; batchStart < len(chunks); batchStart += embeddingBatchSize {
		batchEnd := batchStart + embeddingBatchSize
		if batchEnd > len(chunks) {
			batchEnd = len(chunks)
		}

		batch := chunks[batchStart:batchEnd]
		batchTexts := make([]string, len(batch))
		for i, c := range batch {
			batchTexts[i] = c.Text
		}

		embeddings, err := ing.openai.CreateEmbeddings(batchTexts, "")
		if err != nil {
			return 0, fmt.Errorf("embedding batch starting at %d: %w", batchStart, err)
		}

		for i := range batch {
			chunks[batchStart+i].Embedding = embeddings[i]
		}
	}

	ing.kb.AddChunks(chunks)
	return len(chunks), nil
}

type Retriever struct {
	kb     *KnowledgeBase
	openai *OpenAIClient
	topK   int
}

func NewRetriever(kb *KnowledgeBase, openai *OpenAIClient, topK int) *Retriever {
	if topK <= 0 {
		topK = 5
	}
	return &Retriever{kb: kb, openai: openai, topK: topK}
}

func (r *Retriever) Retrieve(query string) ([]SearchResult, error) {
	embeddings, err := r.openai.CreateEmbeddings([]string{query}, "")
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, fmt.Errorf("no embedding returned for query")
	}
	return r.kb.Search(embeddings[0], r.topK), nil
}

func (r *Retriever) BuildContext(query string) (string, error) {
	results, err := r.Retrieve(query)
	if err != nil {
		return "", err
	}
	if len(results) == 0 {
		return "", nil
	}

	ctx := "Relevant knowledge:\n\n"
	for i, sr := range results {
		ctx += fmt.Sprintf("--- [%d] (doc: %s, score: %.3f) ---\n%s\n\n", i+1, sr.Chunk.DocID, sr.Score, sr.Chunk.Text)
	}
	return ctx, nil
}
