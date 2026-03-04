package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"
)

type OpenAIClient struct {
	apiKey string
	http   *http.Client
}

func NewOpenAIClientFromEnv() (*OpenAIClient, error) {
	k := os.Getenv("OPENAI_API_KEY")
	if k == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY is not set")
	}
	return &OpenAIClient{
		apiKey: k,
		http: &http.Client{
			Timeout: 30 * time.Second,
		},
	}, nil
}

/*
Docs:
- POST https://api.openai.com/v1/responses with tools
- model can return output items of type "function_call" including "call_id" + "arguments"
- you send back input items of type "function_call_output" with the same call_id
 [oai_citation:1‡OpenAI Developers](https://developers.openai.com/api/reference/resources/responses/methods/create/)
*/

type responseCreateReq struct {
	Model              string        `json:"model"`
	Input              any           `json:"input"`
	Instructions        string        `json:"instructions,omitempty"`
	Tools              []toolDef     `json:"tools,omitempty"`
	ToolChoice         string        `json:"tool_choice,omitempty"` // "auto" | "none" | "required"
	PreviousResponseID string        `json:"previous_response_id,omitempty"`
	Reasoning          *reasoningCfg `json:"reasoning,omitempty"`
}

type reasoningCfg struct {
	Effort string `json:"effort,omitempty"` // low|medium|high|xhigh depending on model
}

type toolDef struct {
	Type        string                 `json:"type"` // "function"
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	Parameters  map[string]any         `json:"parameters"`
	Strict      bool                   `json:"strict,omitempty"`
}

type responseObj struct {
	ID     string       `json:"id"`
	Status string       `json:"status"`
	Output []outputItem `json:"output"`
}

type outputItem struct {
	Type      string `json:"type"` // "message" | "function_call" | ...
	ID        string `json:"id,omitempty"`
	Role      string `json:"role,omitempty"`
	CallID    string `json:"call_id,omitempty"`
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"` // JSON string
	// message content omitted for simplicity
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text,omitempty"`
	} `json:"content,omitempty"`
}

func (c *OpenAIClient) CreateResponse(req responseCreateReq) (*responseObj, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/responses", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var b bytes.Buffer
		_, _ = b.ReadFrom(resp.Body)
		return nil, fmt.Errorf("openai error: status=%d body=%s", resp.StatusCode, b.String())
	}

	var out responseObj
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	return &out, nil
}

func extractText(r *responseObj) string {
	for _, item := range r.Output {
		if item.Type == "message" {
			for _, part := range item.Content {
				if part.Type == "output_text" && part.Text != "" {
					return part.Text
				}
			}
		}
	}
	return ""
}

func extractFunctionCalls(r *responseObj) []outputItem {
	var calls []outputItem
	for _, item := range r.Output {
		if item.Type == "function_call" {
			calls = append(calls, item)
		}
	}
	return calls
}

// --- Embeddings API ---

type embeddingReq struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type embeddingResp struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
}

func (c *OpenAIClient) CreateEmbeddings(texts []string, model string) ([][]float64, error) {
	if model == "" {
		model = "text-embedding-3-small"
	}

	body, err := json.Marshal(embeddingReq{
		Model: model,
		Input: texts,
	})
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequest("POST", "https://api.openai.com/v1/embeddings", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		var b bytes.Buffer
		_, _ = b.ReadFrom(resp.Body)
		return nil, fmt.Errorf("openai embeddings error: status=%d body=%s", resp.StatusCode, b.String())
	}

	var out embeddingResp
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}

	embeddings := make([][]float64, len(out.Data))
	for _, d := range out.Data {
		embeddings[d.Index] = d.Embedding
	}
	return embeddings, nil
}