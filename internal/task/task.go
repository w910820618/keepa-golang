package task

import (
	"context"
	"time"
)

// Task 定义了所有定时任务必须实现的接口
type Task interface {
	// Name 返回任务名称，用于标识和日志记录
	Name() string

	// Schedule 返回 cron 表达式，定义任务执行时间
	// 支持标准 cron 表达式：秒 分 时 日 月 周
	// 例如: "0 */5 * * * *" 表示每5分钟执行一次
	Schedule() string

	// Run 执行任务逻辑
	// ctx 用于取消和超时控制
	// 返回执行结果和错误
	Run(ctx context.Context) error

	// Timeout 返回任务执行的超时时间
	// 如果返回 0，则使用默认超时时间
	Timeout() time.Duration

	// Enabled 返回任务是否启用
	Enabled() bool
}

// TaskResult 任务执行结果
type TaskResult struct {
	TaskName   string
	StartTime  time.Time
	EndTime    time.Time
	Duration   time.Duration
	Success    bool
	Error      error
	RetryCount int
}

// TaskRegistry 任务注册表
type TaskRegistry struct {
	tasks map[string]Task
}

// NewTaskRegistry 创建新的任务注册表
func NewTaskRegistry() *TaskRegistry {
	return &TaskRegistry{
		tasks: make(map[string]Task),
	}
}

// Register 注册任务
func (r *TaskRegistry) Register(task Task) error {
	name := task.Name()
	if name == "" {
		return ErrEmptyTaskName
	}

	if _, exists := r.tasks[name]; exists {
		return ErrTaskAlreadyRegistered
	}

	r.tasks[name] = task
	return nil
}

// GetTask 获取任务
func (r *TaskRegistry) GetTask(name string) (Task, bool) {
	task, exists := r.tasks[name]
	return task, exists
}

// GetAllTasks 获取所有任务
func (r *TaskRegistry) GetAllTasks() map[string]Task {
	result := make(map[string]Task)
	for k, v := range r.tasks {
		result[k] = v
	}
	return result
}

// GetEnabledTasks 获取所有启用的任务
func (r *TaskRegistry) GetEnabledTasks() map[string]Task {
	result := make(map[string]Task)
	for name, task := range r.tasks {
		if task.Enabled() {
			result[name] = task
		}
	}
	return result
}
