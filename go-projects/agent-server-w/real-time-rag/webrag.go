package main

import (
	"fmt"
	"strings"
	"sync"
)

type WebRAG struct {
	searcher *WebSearcher
	scraper  *WebScraper
	openai   *OpenAIClient
}

func NewWebRAG(openai *OpenAIClient) *WebRAG {
	return &WebRAG{
		searcher: NewWebSearcher(),
		scraper:  NewWebScraper(),
		openai:   openai,
	}
}

type WebRAGResult struct {
	SearchResults []WebSearchResult `json:"search_results"`
	PagesScraped  int               `json:"pages_scraped"`
	ChunksCreated int               `json:"chunks_created"`
	Context       string            `json:"context"`
}

const defaultMaxPages = 10

func (wr *WebRAG) Retrieve(query string, topK int) (*WebRAGResult, error) {
	maxPages := defaultMaxPages
	if topK <= 0 {
		topK = 5
	}

	// Step 1: Search the web
	searchResults, err := wr.searcher.Search(query, maxPages)
	if err != nil {
		return nil, fmt.Errorf("web search failed: %w", err)
	}
	if len(searchResults) == 0 {
		return &WebRAGResult{Context: ""}, nil
	}

	// Step 2: Separate scrapable URLs (wikipedia) from snippet-only sources (news)
	var scrapableResults []WebSearchResult
	var snippetResults []WebSearchResult
	for _, sr := range searchResults {
		if sr.Source == "wikipedia" {
			scrapableResults = append(scrapableResults, sr)
		} else {
			snippetResults = append(snippetResults, sr)
		}
	}

	var allChunks []Chunk

	// Step 2a: Use news headlines + snippets directly as chunks (no scraping)
	for _, sr := range snippetResults {
		text := sr.Title
		if sr.Snippet != "" {
			text += ". " + sr.Snippet
		}
		if len(strings.TrimSpace(text)) < 20 {
			continue
		}
		allChunks = append(allChunks, Chunk{
			DocID: sr.Title,
			Index: 0,
			Text:  text,
		})
	}
	fmt.Printf("[webrag] added %d news headline chunks\n", len(allChunks))

	// Step 2b: Scrape wikipedia pages concurrently for full content
	type scrapeResult struct {
		url  string
		text string
		err  error
	}

	pagesScraped := 0
	if len(scrapableResults) > 0 {
		ch := make(chan scrapeResult, len(scrapableResults))
		var wg sync.WaitGroup
		for _, sr := range scrapableResults {
			wg.Add(1)
			go func(pageURL string) {
				defer wg.Done()
				text, err := wr.scraper.Scrape(pageURL)
				ch <- scrapeResult{url: pageURL, text: text, err: err}
			}(sr.URL)
		}
		wg.Wait()
		close(ch)

		for res := range ch {
			if res.err != nil {
				fmt.Printf("[webrag] scrape error for %s: %v\n", res.url, res.err)
				continue
			}
			if len(strings.TrimSpace(res.text)) < 50 {
				continue
			}
			pagesScraped++

			texts := ChunkText(res.text, 500, 100)
			for i, t := range texts {
				allChunks = append(allChunks, Chunk{
					DocID: res.url,
					Index: i,
					Text:  t,
				})
			}
		}
		fmt.Printf("[webrag] scraped %d wikipedia pages\n", pagesScraped)
	}

	if len(allChunks) == 0 {
		return &WebRAGResult{
			SearchResults: searchResults,
			Context:       "",
		}, nil
	}

	// Step 3: Embed all chunks
	batchSize := 64
	for batchStart := 0; batchStart < len(allChunks); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(allChunks) {
			batchEnd = len(allChunks)
		}
		batch := allChunks[batchStart:batchEnd]

		texts := make([]string, len(batch))
		for i, c := range batch {
			texts[i] = c.Text
		}

		embeddings, err := wr.openai.CreateEmbeddings(texts, "")
		if err != nil {
			return nil, fmt.Errorf("embedding error: %w", err)
		}
		for i := range batch {
			allChunks[batchStart+i].Embedding = embeddings[i]
		}
	}

	// Step 4: Embed the query and find top-K chunks by cosine similarity
	queryEmb, err := wr.openai.CreateEmbeddings([]string{query}, "")
	if err != nil {
		return nil, fmt.Errorf("query embedding error: %w", err)
	}

	kb := NewKnowledgeBase()
	kb.AddChunks(allChunks)
	topResults := kb.Search(queryEmb[0], topK)

	// Step 5: Format the context string
	ctx := "Web search results for: " + query + "\n\n"
	for i, sr := range topResults {
		ctx += fmt.Sprintf("--- [%d] (source: %s, relevance: %.3f) ---\n%s\n\n",
			i+1, sr.Chunk.DocID, sr.Score, sr.Chunk.Text)
	}

	return &WebRAGResult{
		SearchResults: searchResults,
		PagesScraped:  pagesScraped,
		ChunksCreated: len(allChunks),
		Context:       ctx,
	}, nil
}
