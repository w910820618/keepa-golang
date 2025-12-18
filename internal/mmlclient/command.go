package mmlclient

import (
	"context"
	"fmt"
	"strings"
)

// Command 命令接口
// 所有命令都需要实现这个接口
type Command interface {
	// Name 返回命令名称（主命令名）
	Name() string

	// Aliases 返回命令的别名列表
	Aliases() []string

	// Description 返回命令的简短描述
	Description() string

	// Usage 返回命令的使用说明
	Usage() string

	// Execute 执行命令
	// ctx: 上下文
	// args: 命令参数
	// 返回错误信息
	Execute(ctx context.Context, args []string) error
}

// CommandRegistry 命令注册表
type CommandRegistry struct {
	commands map[string]Command
}

// NewCommandRegistry 创建新的命令注册表
func NewCommandRegistry() *CommandRegistry {
	return &CommandRegistry{
		commands: make(map[string]Command),
	}
}

// Register 注册命令
func (r *CommandRegistry) Register(cmd Command) error {
	// 注册主命令名
	if _, exists := r.commands[cmd.Name()]; exists {
		return fmt.Errorf("command '%s' already registered", cmd.Name())
	}
	r.commands[cmd.Name()] = cmd

	// 注册别名
	for _, alias := range cmd.Aliases() {
		if _, exists := r.commands[alias]; exists {
			return fmt.Errorf("command alias '%s' already registered", alias)
		}
		r.commands[alias] = cmd
	}

	return nil
}

// Get 根据名称或别名获取命令
func (r *CommandRegistry) Get(name string) (Command, bool) {
	cmd, ok := r.commands[name]
	return cmd, ok
}

// List 返回所有已注册的命令列表
func (r *CommandRegistry) List() []Command {
	seen := make(map[string]bool)
	commands := make([]Command, 0)

	for _, cmd := range r.commands {
		name := cmd.Name()
		if !seen[name] {
			seen[name] = true
			commands = append(commands, cmd)
		}
	}

	return commands
}

// Help 返回帮助信息
func (r *CommandRegistry) Help() string {
	commands := r.List()
	if len(commands) == 0 {
		return "没有可用的命令"
	}

	var help strings.Builder
	help.WriteString("MML Client - Keepa Server 命令行客户端\n\n")
	help.WriteString("可用命令:\n")

	for _, cmd := range commands {
		help.WriteString(fmt.Sprintf("  %-20s - %s\n", cmd.Name(), cmd.Description()))
		if len(cmd.Aliases()) > 0 {
			aliases := strings.Join(cmd.Aliases(), ", ")
			help.WriteString(fmt.Sprintf("    (别名: %s)\n", aliases))
		}
		help.WriteString(fmt.Sprintf("    用法: %s\n", cmd.Usage()))
		help.WriteString("\n")
	}

	help.WriteString("输入 'help <command>' 查看特定命令的详细帮助\n")

	return help.String()
}

// HelpForCommand 返回特定命令的详细帮助
func (r *CommandRegistry) HelpForCommand(name string) string {
	cmd, ok := r.Get(name)
	if !ok {
		return fmt.Sprintf("未知命令: %s\n输入 'help' 查看所有可用命令", name)
	}

	var help strings.Builder
	help.WriteString(fmt.Sprintf("命令: %s\n", cmd.Name()))
	if len(cmd.Aliases()) > 0 {
		aliases := strings.Join(cmd.Aliases(), ", ")
		help.WriteString(fmt.Sprintf("别名: %s\n", aliases))
	}
	help.WriteString(fmt.Sprintf("描述: %s\n", cmd.Description()))
	help.WriteString(fmt.Sprintf("用法: %s\n", cmd.Usage()))

	return help.String()
}

