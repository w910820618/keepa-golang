package mmlclient

import (
	"context"
	"fmt"
	"os"
)

// ExitCommand 退出命令
type ExitCommand struct{}

// NewExitCommand 创建退出命令
func NewExitCommand() *ExitCommand {
	return &ExitCommand{}
}

// Name 返回命令名称
func (c *ExitCommand) Name() string {
	return "exit"
}

// Aliases 返回命令别名
func (c *ExitCommand) Aliases() []string {
	return []string{"quit", "q"}
}

// Description 返回命令描述
func (c *ExitCommand) Description() string {
	return "退出客户端"
}

// Usage 返回使用说明
func (c *ExitCommand) Usage() string {
	return "exit\n" +
		"  退出交互式客户端\n" +
		"  也可以使用别名: quit, q"
}

// Execute 执行命令
func (c *ExitCommand) Execute(ctx context.Context, args []string) error {
	fmt.Println("再见!")
	os.Exit(0)
	return nil
}

