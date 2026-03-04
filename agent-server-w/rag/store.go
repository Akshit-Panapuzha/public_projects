package main

import "sync"

type Store struct {
	mu     sync.RWMutex
	tasks  []Task
	nextID int
	memory AgentMemory
}

func NewStore() *Store {
	return &Store{
		tasks:  []Task{},
		nextID: 1,
		memory: AgentMemory{History: []string{}},
	}
}