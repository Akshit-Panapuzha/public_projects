package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

type WebSearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Source  string `json:"source"`
}

type WebSearcher struct {
	http *http.Client
}

func NewWebSearcher() *WebSearcher {
	return &WebSearcher{
		http: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Search queries both Google News RSS and Wikipedia, merges the results.
func (ws *WebSearcher) Search(query string, maxResults int) ([]WebSearchResult, error) {
	if maxResults <= 0 {
		maxResults = 10
	}

	var allResults []WebSearchResult

	newsResults, err := ws.searchGoogleNews(query, maxResults)
	if err != nil {
		fmt.Printf("[websearch] google news error: %v\n", err)
	} else {
		allResults = append(allResults, newsResults...)
	}

	wikiResults, err := ws.searchWikipedia(query, 5)
	if err != nil {
		fmt.Printf("[websearch] wikipedia error: %v\n", err)
	} else {
		allResults = append(allResults, wikiResults...)
	}

	if len(allResults) == 0 {
		return nil, fmt.Errorf("no results from any search source")
	}

	if len(allResults) > maxResults {
		allResults = allResults[:maxResults]
	}

	fmt.Printf("[websearch] total: %d results (%d news, %d wiki)\n",
		len(allResults), len(newsResults), len(wikiResults))
	return allResults, nil
}

// --- Google News RSS (free, no key, returns current news) ---

type rssResponse struct {
	Channel struct {
		Items []rssItem `xml:"item"`
	} `xml:"channel"`
}

type rssItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Source      string `xml:"source"`
}

func (ws *WebSearcher) searchGoogleNews(query string, maxResults int) ([]WebSearchResult, error) {
	endpoint := fmt.Sprintf(
		"https://news.google.com/rss/search?q=%s&hl=en-US&gl=US&ceid=US:en",
		url.QueryEscape(query))

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; GoRAGBot/1.0)")

	resp, err := ws.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rss rssResponse
	if err := xml.Unmarshal(body, &rss); err != nil {
		return nil, fmt.Errorf("xml parse error: %w", err)
	}

	var results []WebSearchResult
	for _, item := range rss.Channel.Items {
		if len(results) >= maxResults {
			break
		}
		results = append(results, WebSearchResult{
			Title:   strings.TrimSpace(item.Title),
			URL:     strings.TrimSpace(item.Link),
			Snippet: stripTags(item.Description),
			Source:  "google_news",
		})
	}
	return results, nil
}

// --- Wikipedia API (free, no key, factual content) ---

func (ws *WebSearcher) searchWikipedia(query string, maxResults int) ([]WebSearchResult, error) {
	endpoint := fmt.Sprintf(
		"https://en.wikipedia.org/w/api.php?action=query&list=search&srsearch=%s&format=json&srlimit=%d&utf8=1",
		url.QueryEscape(query), maxResults)

	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "GoRAGBot/1.0 (educational project)")

	resp, err := ws.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Query struct {
			Search []struct {
				Title   string `json:"title"`
				Snippet string `json:"snippet"`
				PageID  int    `json:"pageid"`
			} `json:"search"`
		} `json:"query"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("json parse error: %w", err)
	}

	var results []WebSearchResult
	for _, r := range parsed.Query.Search {
		results = append(results, WebSearchResult{
			Title:   r.Title,
			URL:     fmt.Sprintf("https://en.wikipedia.org/wiki/%s", url.PathEscape(strings.ReplaceAll(r.Title, " ", "_"))),
			Snippet: stripTags(r.Snippet),
			Source:  "wikipedia",
		})
	}
	return results, nil
}

var tagStripRe = regexp.MustCompile(`<[^>]*>`)

func stripTags(s string) string {
	return strings.TrimSpace(tagStripRe.ReplaceAllString(s, ""))
}
