package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"keepa/internal/task"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"
)

// Scheduler 任务调度器
type Scheduler struct {
	cron           *cron.Cron
	registry       *task.TaskRegistry
	logger         *zap.Logger
	running        bool
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	jobEntries     map[string]cron.EntryID
	defaultTimeout time.Duration
}

// Config 调度器配置
type Config struct {
	Logger         *zap.Logger
	Registry       *task.TaskRegistry
	DefaultTimeout time.Duration
	Location       *time.Location
}

// NewScheduler 创建新的调度器
func NewScheduler(cfg Config) *Scheduler {
	if cfg.Logger == nil {
		cfg.Logger = zap.NewNop()
	}

	if cfg.Registry == nil {
		cfg.Registry = task.NewTaskRegistry()
	}

	if cfg.DefaultTimeout == 0 {
		cfg.DefaultTimeout = 30 * time.Minute
	}

	if cfg.Location == nil {
		cfg.Location = time.Local
	}

	opts := []cron.Option{
		cron.WithLocation(cfg.Location),
		cron.WithSeconds(), // 支持秒级精度
		cron.WithChain(
			cron.Recover(cron.DefaultLogger), // 恢复 panic
		),
	}

	c := cron.New(opts...)

	ctx, cancel := context.WithCancel(context.Background())

	return &Scheduler{
		cron:           c,
		registry:       cfg.Registry,
		logger:         cfg.Logger,
		ctx:            ctx,
		cancel:         cancel,
		jobEntries:     make(map[string]cron.EntryID),
		defaultTimeout: cfg.DefaultTimeout,
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return fmt.Errorf("scheduler is already running")
	}

	// 注册所有启用的任务
	tasks := s.registry.GetEnabledTasks()
	for name, t := range tasks {
		if err := s.addTask(name, t); err != nil {
			s.logger.Error("failed to add task",
				zap.String("task", name),
				zap.Error(err),
			)
			continue
		}
		s.logger.Info("task registered",
			zap.String("task", name),
			zap.String("schedule", t.Schedule()),
		)
	}

	s.cron.Start()
	s.running = true

	s.logger.Info("scheduler started",
		zap.Int("total_tasks", len(tasks)),
	)

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return nil
	}

	s.logger.Info("stopping scheduler...")

	stopCtx := s.cron.Stop()
	select {
	case <-stopCtx.Done():
		s.logger.Info("scheduler stopped gracefully")
	case <-ctx.Done():
		s.logger.Warn("context cancelled while stopping scheduler")
		return ctx.Err()
	}

	s.cancel()
	s.running = false

	return nil
}

// addTask 添加任务到调度器
func (s *Scheduler) addTask(name string, t task.Task) error {
	schedule := t.Schedule()
	if schedule == "" {
		return fmt.Errorf("task schedule cannot be empty")
	}

	entryID, err := s.cron.AddFunc(schedule, s.wrapTask(name, t))
	if err != nil {
		return fmt.Errorf("failed to parse schedule: %w", err)
	}

	s.jobEntries[name] = entryID
	return nil
}

// wrapTask 包装任务执行逻辑
func (s *Scheduler) wrapTask(name string, t task.Task) func() {
	return func() {
		startTime := time.Now()
		s.logger.Info("task started",
			zap.String("task", name),
			zap.Time("start_time", startTime),
		)

		// 确定超时时间
		timeout := t.Timeout()
		if timeout == 0 {
			timeout = s.defaultTimeout
		}

		// 创建带超时的上下文
		ctx, cancel := context.WithTimeout(s.ctx, timeout)
		defer cancel()

		// 执行任务
		err := t.Run(ctx)
		duration := time.Since(startTime)

		result := task.TaskResult{
			TaskName:  name,
			StartTime: startTime,
			EndTime:   time.Now(),
			Duration:  duration,
			Success:   err == nil,
			Error:     err,
		}

		// 记录结果
		s.logTaskResult(result)
	}
}

// logTaskResult 记录任务执行结果
func (s *Scheduler) logTaskResult(result task.TaskResult) {
	fields := []zap.Field{
		zap.String("task", result.TaskName),
		zap.Time("start_time", result.StartTime),
		zap.Time("end_time", result.EndTime),
		zap.Duration("duration", result.Duration),
		zap.Bool("success", result.Success),
	}

	if result.Error != nil {
		fields = append(fields, zap.Error(result.Error))
		s.logger.Error("task completed with error", fields...)
	} else {
		s.logger.Info("task completed successfully", fields...)
	}
}

// IsRunning 检查调度器是否运行中
func (s *Scheduler) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// GetTaskCount 获取任务数量
func (s *Scheduler) GetTaskCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.jobEntries)
}

// RemoveTask 移除任务（需要停止调度器后调用）
func (s *Scheduler) RemoveTask(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	entryID, exists := s.jobEntries[name]
	if !exists {
		return fmt.Errorf("task not found: %s", name)
	}

	s.cron.Remove(entryID)
	delete(s.jobEntries, name)

	s.logger.Info("task removed",
		zap.String("task", name),
	)

	return nil
}
