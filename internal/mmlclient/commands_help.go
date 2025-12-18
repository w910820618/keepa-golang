package mmlclient

import (
	"context"
	"fmt"
)

// HelpCommand 帮助命令
type HelpCommand struct {
	registry *CommandRegistry
}

// NewHelpCommand 创建帮助命令
func NewHelpCommand(registry *CommandRegistry) *HelpCommand {
	return &HelpCommand{
		registry: registry,
	}
}

// Name 返回命令名称
func (c *HelpCommand) Name() string {
	return "help"
}

// Aliases 返回命令别名
func (c *HelpCommand) Aliases() []string {
	return []string{"h", "?"}
}

// Description 返回命令描述
func (c *HelpCommand) Description() string {
	return "显示帮助信息"
}

// Usage 返回使用说明
func (c *HelpCommand) Usage() string {
	return "help [command]\n" +
		"  如果不提供 command，显示所有命令的帮助\n" +
		"  如果提供 command，显示特定命令的详细帮助\n" +
		"  示例:\n" +
		"    help              # 显示所有命令\n" +
		"    help category-tree # 显示 category-tree 命令的详细帮助"
}

// Execute 执行命令
func (c *HelpCommand) Execute(ctx context.Context, args []string) error {
	if len(args) == 0 {
		// 显示所有命令的帮助
		fmt.Println(c.registry.Help())
	} else {
		// 显示特定命令的帮助
		commandName := args[0]
		fmt.Println(c.registry.HelpForCommand(commandName))
	}
	return nil
}

