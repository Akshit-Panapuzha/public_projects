package main

import "strings"

func RunAgent(goal string) map[string]string {

	// Store memory
	memory.History = append(memory.History, goal)

	// Simple decision logic
	if strings.Contains(strings.ToLower(goal), "create task") {
		CreateTask(goal)

		return map[string]string{
			"action": "create_task",
			"status": "success",
		}
	}

	return map[string]string{
		"action": "none",
		"status": "unknown goal",
	}
}