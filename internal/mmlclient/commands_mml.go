package mmlclient

import (
	"context"
	"fmt"
	"strings"
)

// MMLCommand MML 命令，显示所有命令及中文解释
type MMLCommand struct {
	registry *CommandRegistry
}

// NewMMLCommand 创建 MML 命令
func NewMMLCommand(registry *CommandRegistry) *MMLCommand {
	return &MMLCommand{
		registry: registry,
	}
}

// Name 返回命令名称
func (c *MMLCommand) Name() string {
	return "mml"
}

// Aliases 返回命令别名
func (c *MMLCommand) Aliases() []string {
	return []string{}
}

// Description 返回命令描述
func (c *MMLCommand) Description() string {
	return "显示所有 MML 命令及中文解释"
}

// Usage 返回使用说明
func (c *MMLCommand) Usage() string {
	return "mml\n" +
		"  显示所有可用的 MML 命令及其详细的中文解释\n" +
		"  示例:\n" +
		"    mml  # 显示所有命令"
}

// Execute 执行命令
func (c *MMLCommand) Execute(ctx context.Context, args []string) error {
	commands := c.registry.List()
	if len(commands) == 0 {
		fmt.Println("没有可用的命令")
		return nil
	}

	var output strings.Builder
	output.WriteString("═══════════════════════════════════════════════════════════\n")
	output.WriteString("                    MML 命令列表\n")
	output.WriteString("═══════════════════════════════════════════════════════════\n\n")

	for i, cmd := range commands {
		// 命令名称和描述
		output.WriteString(fmt.Sprintf("【命令 %d】%s\n", i+1, cmd.Name()))
		output.WriteString(fmt.Sprintf("  描述: %s\n", cmd.Description()))

		// 别名
		if len(cmd.Aliases()) > 0 {
			aliases := strings.Join(cmd.Aliases(), ", ")
			output.WriteString(fmt.Sprintf("  别名: %s\n", aliases))
		}

		// 用法说明
		usage := cmd.Usage()
		if usage != "" {
			// 将用法说明按行分割，每行添加适当的缩进
			usageLines := strings.Split(usage, "\n")
			output.WriteString("  用法:\n")
			for _, line := range usageLines {
				if strings.TrimSpace(line) != "" {
					output.WriteString(fmt.Sprintf("    %s\n", line))
				}
			}
		}

		// 分隔线（最后一个命令后不加）
		if i < len(commands)-1 {
			output.WriteString("\n")
		}
	}

	output.WriteString("\n")
	output.WriteString("═══════════════════════════════════════════════════════════\n")
	output.WriteString("提示: 使用 'help <command>' 查看特定命令的详细帮助\n")
	output.WriteString("═══════════════════════════════════════════════════════════\n")

	fmt.Print(output.String())
	return nil
}

