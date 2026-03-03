package main

func CreateTask(name string) Task {
	task := Task{
		ID:   nextID,
		Name: name,
		Done: false,
	}

	nextID++
	tasks = append(tasks, task)

	return task
}