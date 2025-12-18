package task

import "errors"

var (
	// ErrEmptyTaskName 任务名称为空
	ErrEmptyTaskName = errors.New("task name cannot be empty")
	
	// ErrTaskAlreadyRegistered 任务已注册
	ErrTaskAlreadyRegistered = errors.New("task already registered")
	
	// ErrTaskNotFound 任务未找到
	ErrTaskNotFound = errors.New("task not found")
)

