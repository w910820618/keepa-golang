# 架构设计文档

## 整体架构

本项目采用清晰的分层架构设计，遵循 Go 语言的最佳实践，确保代码的可维护性和可扩展性。

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                    │
│                  (cmd/keepa/main.go)                    │
│  - 程序入口点                                           │
│  - 配置加载和初始化                                      │
│  - 信号处理和优雅关闭                                    │
└─────────────────┬───────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────┐
│                    Scheduler Layer                      │
│              (internal/scheduler/)                      │
│  - 任务调度器核心                                        │
│  - Cron 表达式解析和执行                                  │
│  - 任务执行包装和错误处理                                 │
│  - 超时控制                                             │
└─────────────────┬───────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────┐
│                     Task Layer                          │
│              (internal/task/, tasks/)                   │
│  - 任务接口定义                                          │
│  - 任务注册表                                            │
│  - 具体任务实现                                          │
└─────────────────────────────────────────────────────────┘
                  │
┌─────────────────▼───────────────────────────────────────┐
│                  Infrastructure Layer                   │
│        (internal/config/, logger/)                      │
│  - 配置管理                                              │
│  - 日志系统                                              │
└─────────────────────────────────────────────────────────┘
```

## 核心组件

### 1. 任务接口 (Task Interface)

所有定时任务必须实现 `task.Task` 接口：

```go
type Task interface {
    Name() string           // 任务名称
    Schedule() string       // Cron 表达式
    Run(ctx context.Context) error  // 执行逻辑
    Timeout() time.Duration // 超时时间
    Enabled() bool          // 是否启用
}
```

**设计优势**：
- 统一的接口规范，易于扩展
- 支持任务级别的超时控制
- 支持动态启用/禁用任务

### 2. 任务注册表 (Task Registry)

`TaskRegistry` 负责管理和注册所有任务：

**功能**：
- 任务注册和查找
- 过滤已启用的任务
- 防止重复注册

**使用示例**：
```go
registry := task.NewTaskRegistry()
registry.Register(NewMyTask())
enabledTasks := registry.GetEnabledTasks()
```

### 3. 调度器 (Scheduler)

`Scheduler` 是系统的核心组件，负责：

**核心功能**：
- 解析 Cron 表达式
- 定时触发任务执行
- 任务执行的包装和监控
- 错误处理和日志记录
- 优雅关闭

**关键特性**：
- 基于 `robfig/cron/v3`，支持秒级精度
- 支持自定义时区
- 自动恢复 panic
- 任务执行结果记录

### 4. 配置管理 (Config)

使用 `viper` 实现灵活的配置管理：

**支持方式**：
- YAML 配置文件
- 环境变量（前缀：`KEEPA_`）
- 默认值

**配置结构**：
```yaml
app:
  name: keepa
  version: 1.0.0
  env: development

scheduler:
  default_timeout: 30m
  location: Asia/Shanghai

logger:
  level: info
  format: console
  output_path: logs/app.log
```

### 5. 日志系统 (Logger)

基于 `zap` 的高性能日志系统：

**特性**：
- 结构化日志输出
- 支持 JSON 和 Console 两种格式
- 文件轮转（基于 lumberjack）
- 可配置的日志级别
- 自动压缩旧日志

## 数据流

### 任务执行流程

```
1. Scheduler.Start()
   ├─> 获取所有启用的任务
   ├─> 解析每个任务的 Cron 表达式
   ├─> 注册到 cron 引擎
   └─> 启动 cron 引擎

2. 定时触发 (Cron Engine)
   ├─> 调用 wrapTask 包装函数
   ├─> 创建带超时的 Context
   ├─> 执行 Task.Run(ctx)
   └─> 记录执行结果

3. 优雅关闭
   ├─> 接收 SIGINT/SIGTERM 信号
   ├─> 停止接收新任务
   ├─> 等待当前任务完成
   └─> 释放资源
```

## 扩展指南

### 添加新任务

1. **创建任务文件**（在 `internal/tasks/` 目录下）

```go
package tasks

type MyTask struct {
    name    string
    enabled bool
}

func NewMyTask() task.Task {
    return &MyTask{
        name:    "my_task",
        enabled: true,
    }
}

// 实现 Task 接口的所有方法
```

2. **注册任务**（在 `cmd/keepa/main.go` 中）

```go
func registerTasks(registry *task.TaskRegistry) error {
    if err := registry.Register(tasks.NewMyTask()); err != nil {
        return fmt.Errorf("failed to register my task: %w", err)
    }
    return nil
}
```

### 添加配置项

1. **更新配置结构**（`internal/config/config.go`）

```go
type Config struct {
    // ... 现有配置
    NewFeature NewFeatureConfig `mapstructure:"new_feature"`
}
```

2. **设置默认值**

```go
viper.SetDefault("new_feature.enabled", true)
```

3. **更新配置文件**（`configs/config.yaml`）

```yaml
new_feature:
  enabled: true
```

### 添加中间件

可以在 `Scheduler.wrapTask` 中添加中间件逻辑：

- 任务执行前后的钩子
- 性能监控
- 指标收集
- 分布式锁

## 设计原则

1. **单一职责原则**：每个组件只负责一个功能
2. **接口隔离**：通过接口定义契约，便于测试和替换
3. **依赖注入**：通过构造函数注入依赖，提高可测试性
4. **优雅关闭**：正确处理信号，确保资源清理
5. **错误处理**：所有错误都应该被记录和处理
6. **可观测性**：通过日志和指标了解系统运行状态

## 性能考虑

1. **轻量级**：使用高效的 cron 库，最小化开销
2. **并发安全**：使用互斥锁保护共享状态
3. **资源控制**：通过超时机制防止任务无限运行
4. **日志优化**：使用结构化日志，支持异步写入

## 安全性

1. **输入验证**：验证 Cron 表达式格式
2. **超时控制**：防止任务无限运行
3. **错误隔离**：单个任务失败不影响其他任务
4. **资源限制**：通过 Context 控制资源使用

## 测试建议

1. **单元测试**：测试各个组件的独立功能
2. **集成测试**：测试组件间的交互
3. **模拟测试**：使用 mock 测试任务执行
4. **基准测试**：测试调度器的性能

