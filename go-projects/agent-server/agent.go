package main

import (
	"encoding/json"
	"fmt"
)

type Agent struct {
	store  *Store
	openai *OpenAIClient
	model  string
}

func NewAgent(store *Store, openai *OpenAIClient) *Agent {
	return &Agent{
		store:  store,
		openai: openai,
		model:  "gpt-4.1", // pick a model you have access to
	}
}

type AgentResult struct {
	Final     string        `json:"final"`
	ToolCalls []ToolCallLog `json:"tool_calls"`
	Memory    AgentMemory   `json:"memory"`
}

type ToolCallLog struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
	Output    string `json:"output"`
}

func (a *Agent) tools() []toolDef {
	// JSON-schema parameters, strict mode (so args are shaped correctly)
	// The Responses API supports function tool definitions like this.  [oai_citation:2‡OpenAI Developers](https://developers.openai.com/api/reference/resources/responses/methods/create/)
	return []toolDef{
		{
			Type:        "function",
			Name:        "create_task",
			Description: "Create a new task in the task list.",
			Strict:      true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"name": map[string]any{"type": "string", "description": "Task title"},
				},
				"required":             []string{"name"},
				"additionalProperties": false,
			},
		},
		{
			Type:        "function",
			Name:        "list_tasks",
			Description: "List all tasks.",
			Strict:      true,
			Parameters: map[string]any{
				"type":                 "object",
				"properties":           map[string]any{},
				"required":             []string{},
				"additionalProperties": false,
			},
		},
		{
			Type:        "function",
			Name:        "get_task",
			Description: "Get a task by id.",
			Strict:      true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "integer", "description": "Task id"},
				},
				"required":             []string{"id"},
				"additionalProperties": false,
			},
		},
		{
			Type:        "function",
			Name:        "mark_done",
			Description: "Mark a task as done by id.",
			Strict:      true,
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"id": map[string]any{"type": "integer", "description": "Task id"},
				},
				"required":             []string{"id"},
				"additionalProperties": false,
			},
		},
	}
}

func (a *Agent) Run(goal string) (*AgentResult, error) {
	a.store.AddMemory(goal)

	instructions := `You are a helpful agent for a tiny task API.
If the user asks to create/list/get/complete tasks, call the appropriate tool.
If no tool is needed, respond normally.`

	// 1) First model call with tools available
	first, err := a.openai.CreateResponse(responseCreateReq{
		Model:        a.model,
		Input:        goal,
		Instructions: instructions,
		Tools:        a.tools(),
		ToolChoice:   "auto",
	})
	if err != nil {
		return nil, err
	}

	logs := []ToolCallLog{}
	calls := extractFunctionCalls(first)
	if len(calls) == 0 {
		return &AgentResult{
			Final:     extractText(first),
			ToolCalls: logs,
			Memory:    a.store.MemorySnapshot(),
		}, nil
	}

	// 2) Execute tool calls & build function_call_output items
	toolOutputs := make([]map[string]any, 0, len(calls))

	for _, call := range calls {
		outStr, execErr := a.executeTool(call.Name, call.Arguments)

		// we still send tool output (even if error) so model can respond gracefully
		if execErr != nil {
			outStr = fmt.Sprintf(`{"error":%q}`, execErr.Error())
		}

		logs = append(logs, ToolCallLog{
			Name:      call.Name,
			Arguments: call.Arguments,
			Output:    outStr,
		})

		// Responses API expects input items with type function_call_output + call_id.  [oai_citation:3‡OpenAI Developers](https://developers.openai.com/api/reference/resources/responses/methods/create/)
		toolOutputs = append(toolOutputs, map[string]any{
			"type":    "function_call_output",
			"call_id": call.CallID,
			"output":  outStr, // JSON string output
		})
	}

	// 3) Send tool outputs back to the model to get the final answer
	second, err := a.openai.CreateResponse(responseCreateReq{
		Model:              a.model,
		PreviousResponseID: first.ID,
		Input:              toolOutputs,
		Instructions:       instructions,
		Tools:              a.tools(),
		ToolChoice:         "auto",
	})
	if err != nil {
		return nil, err
	}

	return &AgentResult{
		Final:     extractText(second),
		ToolCalls: logs,
		Memory:    a.store.MemorySnapshot(),
	}, nil
}

func (a *Agent) executeTool(name string, argsJSON string) (string, error) {
	switch name {

	case "create_task":
		var args struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", err
		}
		t := a.store.CreateTask(args.Name)
		b, _ := json.Marshal(t)
		return string(b), nil

	case "list_tasks":
		tasks := a.store.ListTasks()
		b, _ := json.Marshal(tasks)
		return string(b), nil

	case "get_task":
		var args struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", err
		}
		t, err := a.store.GetTask(args.ID)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(t)
		return string(b), nil

	case "mark_done":
		var args struct {
			ID int `json:"id"`
		}
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", err
		}
		t, err := a.store.MarkDone(args.ID)
		if err != nil {
			return "", err
		}
		b, _ := json.Marshal(t)
		return string(b), nil

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}