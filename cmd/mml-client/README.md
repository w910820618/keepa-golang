# MML Client

MML Client 是一个命令行客户端工具，用于与 Keepa Server 进行交互。它采用命令模式设计，易于扩展和维护。

## 架构设计

### 核心组件

1. **Command 接口** (`internal/mmlclient/command.go`)
   - 定义所有命令必须实现的接口
   - 包含 `Name()`, `Aliases()`, `Description()`, `Usage()`, `Execute()` 方法

2. **CommandRegistry** (`internal/mmlclient/command.go`)
   - 命令注册表，管理所有已注册的命令
   - 支持命令名称和别名查找
   - 提供帮助信息生成功能

3. **Client** (`internal/mmlclient/client.go`)
   - HTTP 客户端，负责与服务器通信
   - 封装了 JSON 请求和响应处理

4. **App** (`cmd/mml-client/main.go`)
   - 应用程序主结构
   - 负责命令注册和路由

### 命令实现

每个命令都是独立的文件，位于 `internal/mmlclient/` 目录下，实现 `Command` 接口：

- `commands_category_tree.go` - 分类树构建命令
- `commands_help.go` - 帮助命令
- `commands_mml.go` - MML 命令（显示所有命令及中文解释）
- `commands_exit.go` - 退出命令

**优势**：由于命令实现位于 `internal/mmlclient` 包中，可以直接使用 `internal/model` 包中的结构体，实现代码共享。

## 添加新命令

要添加新命令，只需：

1. **在 `internal/mmlclient/` 目录下创建命令文件**（如 `commands_mycommand.go`）：

```go
package mmlclient

import (
	"context"
	"fmt"
	
	"keepa/internal/model"  // 可以使用 model 包中的结构体
)

// MyCommand 我的命令
type MyCommand struct {
	client *Client
}

// NewMyCommand 创建我的命令
func NewMyCommand(client *Client) *MyCommand {
	return &MyCommand{
		client: client,
	}
}

// Name 返回命令名称
func (c *MyCommand) Name() string {
	return "mycommand"
}

// Aliases 返回命令别名
func (c *MyCommand) Aliases() []string {
	return []string{"mc", "my"}
}

// Description 返回命令描述
func (c *MyCommand) Description() string {
	return "我的命令描述"
}

// Usage 返回使用说明
func (c *MyCommand) Usage() string {
	return "mycommand [args...]\n" +
		"  命令使用说明"
}

// Execute 执行命令
func (c *MyCommand) Execute(ctx context.Context, args []string) error {
	// 实现命令逻辑
	// 可以使用 model 包中的结构体
	var category model.Category
	_ = category
	
	fmt.Println("执行我的命令...")
	return nil
}
```

2. **在 `cmd/mml-client/main.go` 的 `registerCommands()` 方法中注册**：

```go
func (a *App) registerCommands() {
	// ... 现有命令 ...
	
	// 注册新命令
	if err := a.registry.Register(mmlclient.NewMyCommand(a.client)); err != nil {
		fmt.Printf("警告: 注册 mycommand 命令失败: %v\n", err)
	}
}
```

完成！新命令会自动出现在帮助信息中，并可以通过交互式或非交互式模式使用。

## 使用示例

### 交互式模式

```bash
./bin/mml-client
mml> mml              # 显示所有 MML 命令及中文解释
mml> help             # 显示帮助信息
mml> category-tree    # 构建分类树
mml> category-tree 1036592 1036684
mml> exit             # 退出客户端
```

### 非交互式模式

```bash
# 使用配置文件中的 category_id
./bin/mml-client category-tree

# 使用指定的 category_id
./bin/mml-client category-tree 1036592

# 使用多个 category_id
./bin/mml-client category-tree 1036592 1036684

# 查看所有 MML 命令及中文解释
./bin/mml-client mml

# 查看帮助
./bin/mml-client help
./bin/mml-client help category-tree
```

## 配置

服务器 URL 可以通过以下方式配置（按优先级）：

1. 环境变量 `MML_SERVER_URL`
2. 配置文件 `configs/config.yaml` 中的 `server.host` 和 `server.port`
3. 默认值 `http://localhost:8080`

## 文件结构

```
cmd/mml-client/
├── main.go                    # 应用程序入口和主逻辑
└── README.md                  # 本文档

internal/mmlclient/
├── command.go                 # Command 接口和 CommandRegistry
├── client.go                  # HTTP 客户端
├── commands_category_tree.go  # 分类树命令
├── commands_help.go           # 帮助命令
└── commands_exit.go           # 退出命令
```

## 使用 Model 包

由于命令实现位于 `internal/mmlclient` 包中，可以直接使用 `internal/model` 包中的结构体：

```go
import (
	"keepa/internal/model"
)

// 在命令中使用 model 结构体
func (c *MyCommand) Execute(ctx context.Context, args []string) error {
	var category model.Category
	// 使用 category...
	
	var tree model.CategoryTree
	// 使用 tree...
	
	return nil
}
```

