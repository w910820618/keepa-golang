package tasks

import (
	"context"
	"fmt"
	"time"

	"keepa/internal/task"
)

// CleanupTask 清理任务示例
// 演示如何创建一个需要执行清理操作的任务
type CleanupTask struct {
	name    string
	enabled bool
}

// NewCleanupTask 创建清理任务
func NewCleanupTask() task.Task {
	return &CleanupTask{
		name:    "cleanup_task",
		enabled: true,
	}
}

func (t *CleanupTask) Name() string {
	return t.name
}

func (t *CleanupTask) Schedule() string {
	// 每天凌晨2点执行
	return "0 0 2 * * *"
}

func (t *CleanupTask) Run(ctx context.Context) error {
	// 模拟清理操作
	fmt.Printf("[%s] Starting cleanup task...\n", time.Now().Format(time.RFC3339))

	// 检查上下文是否已取消
	if ctx.Err() != nil {
		return ctx.Err()
	}

	// 模拟清理步骤
	steps := []string{
		"清理临时文件",
		"清理过期数据",
		"压缩日志文件",
		"更新统计信息",
	}

	for i, step := range steps {
		select {
		case <-ctx.Done():
			return fmt.Errorf("cleanup interrupted at step %d: %s", i+1, step)
		default:
			// 模拟每个步骤需要的时间
			time.Sleep(500 * time.Millisecond)
			fmt.Printf("[%s] Completed: %s\n", time.Now().Format(time.RFC3339), step)
		}
	}

	fmt.Printf("[%s] Cleanup task completed successfully\n", time.Now().Format(time.RFC3339))
	return nil
}

func (t *CleanupTask) Timeout() time.Duration {
	// 设置任务超时时间为10分钟
	return 10 * time.Minute
}

func (t *CleanupTask) Enabled() bool {
	return t.enabled
}
