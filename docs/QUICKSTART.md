# 快速开始指南

## 安装和运行

### 1. 克隆项目（如果适用）

```bash
git clone <repository-url>
cd keepa
```

### 2. 安装依赖

```bash
make deps
# 或
go mod download
```

### 3. 运行程序

```bash
make run
# 或
go run cmd/keepa/main.go
```

程序会：
- 加载 `configs/config.yaml` 配置文件
- 注册所有已定义的任务
- 启动调度器
- 开始执行定时任务

### 4. 停止程序

使用 `Ctrl+C` 发送 SIGINT 信号，程序会优雅关闭。

## 创建你的第一个任务

### 步骤 1: 创建任务文件

在 `internal/tasks/` 目录下创建 `hello.go`:

```go
package tasks

import (
    "context"
    "fmt"
    "time"
    
    "keepa/internal/task"
)

type HelloTask struct {
    name    string
    enabled bool
}

func NewHelloTask() task.Task {
    return &HelloTask{
        name:    "hello_task",
        enabled: true,
    }
}

func (t *HelloTask) Name() string {
    return t.name
}

func (t *HelloTask) Schedule() string {
    // 每分钟执行一次
    return "0 * * * * *"
}

func (t *HelloTask) Run(ctx context.Context) error {
    fmt.Printf("[%s] Hello, World!\n", time.Now().Format(time.RFC3339))
    return nil
}

func (t *HelloTask) Timeout() time.Duration {
    return 0 // 使用默认超时
}

func (t *HelloTask) Enabled() bool {
    return t.enabled
}
```

### 步骤 2: 注册任务

在 `cmd/keepa/main.go` 的 `registerTasks` 函数中添加：

```go
func registerTasks(registry *task.TaskRegistry) error {
    // ... 现有任务注册
    
    if err := registry.Register(tasks.NewHelloTask()); err != nil {
        return fmt.Errorf("failed to register hello task: %w", err)
    }
    
    return nil
}
```

### 步骤 3: 重新运行

```bash
make run
```

现在你的任务会按照设定的时间执行！

## Cron 表达式示例

| 表达式 | 说明 |
|--------|------|
| `0 * * * * *` | 每分钟执行 |
| `0 */5 * * * *` | 每5分钟执行 |
| `0 0 * * * *` | 每小时执行 |
| `0 0 */2 * * *` | 每2小时执行 |
| `0 0 0 * * *` | 每天午夜执行 |
| `0 0 9 * * *` | 每天上午9点执行 |
| `0 0 9 * * 1-5` | 工作日上午9点执行 |
| `0 0 0 1 * *` | 每月1号执行 |
| `0 30 14 * * *` | 每天下午2点30分执行 |

## 常见用例

### 1. 数据库清理任务

```go
func (t *DBCleanupTask) Run(ctx context.Context) error {
    // 清理过期数据
    err := t.db.DeleteExpiredRecords(ctx)
    if err != nil {
        return fmt.Errorf("failed to cleanup database: %w", err)
    }
    return nil
}
```

### 2. 文件备份任务

```go
func (t *BackupTask) Run(ctx context.Context) error {
    // 执行备份
    backupPath := fmt.Sprintf("backup_%s.tar.gz", time.Now().Format("20060102"))
    err := t.createBackup(ctx, backupPath)
    if err != nil {
        return fmt.Errorf("backup failed: %w", err)
    }
    return nil
}
```

### 3. API 数据同步任务

```go
func (t *SyncTask) Run(ctx context.Context) error {
    // 同步外部API数据
    data, err := t.apiClient.FetchData(ctx)
    if err != nil {
        return fmt.Errorf("failed to fetch data: %w", err)
    }
    
    return t.storage.Save(ctx, data)
}
```

## 配置说明

### 修改时区

编辑 `configs/config.yaml`:

```yaml
scheduler:
  location: America/New_York  # 修改为你的时区
```

### 修改日志级别

```yaml
logger:
  level: debug  # debug, info, warn, error
```

### 使用环境变量

```bash
export KEEPA_LOGGER_LEVEL=debug
export KEEPA_SCHEDULER_DEFAULT_TIMEOUT=1h
./bin/keepa
```

## 调试技巧

### 1. 启用 Debug 日志

```yaml
logger:
  level: debug
  format: console
```

### 2. 临时禁用任务

在任务实现中：

```go
func (t *MyTask) Enabled() bool {
    return false  // 临时禁用
}
```

### 3. 测试任务逻辑

创建测试文件 `internal/tasks/my_task_test.go`:

```go
func TestMyTask_Run(t *testing.T) {
    task := NewMyTask()
    ctx := context.Background()
    
    err := task.Run(ctx)
    assert.NoError(t, err)
}
```

## 部署建议

### 1. 作为系统服务 (systemd)

创建 `/etc/systemd/system/keepa.service`:

```ini
[Unit]
Description=Keepa Task Scheduler
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/keepa
ExecStart=/path/to/keepa/bin/keepa
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### 2. 使用 Docker

创建 `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o keepa ./cmd/keepa

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/
COPY --from=builder /app/keepa .
COPY --from=builder /app/configs ./configs
CMD ["./keepa"]
```

## 下一步

- 阅读 [架构设计文档](./ARCHITECTURE.md) 了解系统设计
- 查看 `internal/tasks/` 目录下的示例任务
- 根据需要扩展和定制你的任务

