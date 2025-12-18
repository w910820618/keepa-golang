# Function 模块

这个模块用于管理所有通过 Gin 框架暴露的一次性任务接口。

## 目录结构

```
internal/function/
├── handlers/          # 所有功能处理器
│   ├── base.go       # 基础处理器，提供通用功能
│   ├── health.go     # 健康检查处理器
│   └── example.go    # 示例一次性任务处理器
├── router.go         # 路由管理器
├── server.go         # HTTP 服务器
└── README.md         # 本文件
```

## 架构说明

### 1. BaseHandler (handlers/base.go)

所有处理器的基类，提供：
- 统一的响应格式（成功/错误）
- 日志记录功能
- 错误处理

### 2. 创建新的 Handler

在 `handlers/` 目录下创建新的处理器文件，例如 `my_task.go`:

```go
package handlers

import (
	"github.com/gin-gonic/gin"
)

// MyTaskHandler 我的任务处理器
type MyTaskHandler struct {
	*BaseHandler
}

// NewMyTaskHandler 创建我的任务处理器
func NewMyTaskHandler(base *BaseHandler) *MyTaskHandler {
	return &MyTaskHandler{
		BaseHandler: base,
	}
}

// Execute 执行任务
func (h *MyTaskHandler) Execute(c *gin.Context) {
	// 实现你的任务逻辑
	h.JSONSuccess(c, gin.H{
		"message": "task completed",
	})
}
```

### 3. 注册路由

在 `router.go` 的 `SetupRoutes()` 方法中注册新路由：

```go
func (r *Router) SetupRoutes() {
	// ... 现有代码 ...
	
	// 注册新任务
	myTaskHandler := handlers.NewMyTaskHandler(baseHandler)
	functions.POST("/my-task", myTaskHandler.Execute)
}
```

### 4. 配置文件

服务器配置在 `configs/config.yaml` 中：

```yaml
server:
  enabled: true   # 是否启用 HTTP 服务器
  host: "0.0.0.0" # 监听地址
  port: 8080      # 监听端口
  mode: "debug"   # gin 模式: debug, release, test
```

## API 端点

### 健康检查
- `GET /health` - 检查服务器状态

### 一次性任务
- `POST /api/v1/functions/execute` - 执行一次性任务（示例）
- `POST /api/v1/functions/category-tree` - 构建分类树（从配置读取并递归遍历）

## 使用示例

### 调用示例接口

```bash
curl -X POST http://localhost:8080/api/v1/functions/execute \
  -H "Content-Type: application/json" \
  -d '{
    "task_name": "example_task",
    "params": {
      "key": "value"
    }
  }'
```

### 调用分类树构建接口

```bash
curl -X POST http://localhost:8080/api/v1/functions/category-tree \
  -H "Content-Type: application/json"
```

此接口会：
1. 自动读取 `configs/keepa_api_queries.yaml` 中 `category_lookup.us_specific_categories` 的配置
2. 获取配置中的 `category` 数组（例如：`[1036592, 1036684]`）
3. 对每个分类 ID 递归调用 Keepa API 获取其子分类
4. 构建完整的分类树结构
5. 将树结构存储到 MongoDB（集合名称由配置中的 `collection` 字段指定，默认为 `category_lookup`）

### 响应格式

成功响应：
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "task_id": "example-task-id",
    "task_name": "example_task",
    "status": "completed"
  }
}
```

错误响应：
```json
{
  "code": 400,
  "message": "invalid request parameters",
  "error": "具体错误信息"
}
```

## 扩展功能

如果需要注册额外的路由，可以在 `main.go` 中使用 `RegisterCustomRoutes` 方法：

```go
httpServer.RegisterCustomRoutes(func(api *gin.RouterGroup, base *handlers.BaseHandler) {
	// 注册自定义路由
	customHandler := handlers.NewCustomHandler(base)
	api.POST("/custom", customHandler.Handle)
})
```

