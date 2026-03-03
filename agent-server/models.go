package main

type Task struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Done bool   `json:"done"`
}

type AgentMemory struct {
	History []string `json:"history"`
}

var tasks = []Task{}
var nextID = 1
var memory = AgentMemory{}