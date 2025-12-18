package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"keepa/internal/config"
	"keepa/internal/mmlclient"
)

const Prompt = "mml> "

// App 应用程序主结构
type App struct {
	client   *mmlclient.Client
	registry *mmlclient.CommandRegistry
}

// NewApp 创建新的应用程序实例
func NewApp(serverURL string) *App {
	client := mmlclient.NewClient(serverURL)
	registry := mmlclient.NewCommandRegistry()

	// 注册所有命令
	app := &App{
		client:   client,
		registry: registry,
	}

	app.registerCommands()

	return app
}

// registerCommands 注册所有命令
func (a *App) registerCommands() {
	// 注册分类树命令
	if err := a.registry.Register(mmlclient.NewCategoryTreeCommand(a.client)); err != nil {
		fmt.Printf("警告: 注册 category-tree 命令失败: %v\n", err)
	}

	// 注册帮助命令
	if err := a.registry.Register(mmlclient.NewHelpCommand(a.registry)); err != nil {
		fmt.Printf("警告: 注册 help 命令失败: %v\n", err)
	}

	// 注册 MML 命令
	if err := a.registry.Register(mmlclient.NewMMLCommand(a.registry)); err != nil {
		fmt.Printf("警告: 注册 mml 命令失败: %v\n", err)
	}

	// 注册退出命令
	if err := a.registry.Register(mmlclient.NewExitCommand()); err != nil {
		fmt.Printf("警告: 注册 exit 命令失败: %v\n", err)
	}
}

// parseCommand 解析命令行输入
func parseCommand(line string) (string, []string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", nil
	}

	parts := strings.Fields(line)
	if len(parts) == 0 {
		return "", nil
	}

	command := strings.ToLower(parts[0])
	args := parts[1:]

	return command, args
}

// runInteractive 运行交互式模式
func (a *App) runInteractive() {
	scanner := bufio.NewScanner(os.Stdin)

	fmt.Println("MML Client - Keepa Server 命令行客户端")
	fmt.Println("输入 'help' 查看帮助，输入 'exit' 或 'quit' 退出")
	fmt.Println()

	ctx := context.Background()

	for {
		fmt.Print(Prompt)

		if !scanner.Scan() {
			break
		}

		line := scanner.Text()
		commandName, args := parseCommand(line)

		if commandName == "" {
			continue
		}

		// 查找命令
		cmd, ok := a.registry.Get(commandName)
		if !ok {
			fmt.Printf("未知命令: %s\n", commandName)
			fmt.Println("输入 'help' 查看帮助")
			continue
		}

		// 执行命令
		if err := cmd.Execute(ctx, args); err != nil {
			fmt.Printf("错误: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("读取输入时出错: %v\n", err)
	}
}

// runCommand 运行单个命令（非交互式模式）
func (a *App) runCommand(commandName string, args []string) error {
	ctx := context.Background()

	// 查找命令
	cmd, ok := a.registry.Get(commandName)
	if !ok {
		return fmt.Errorf("未知命令: %s\n输入 'mml-client help' 查看帮助", commandName)
	}

	// 执行命令
	return cmd.Execute(ctx, args)
}

// getServerURL 获取服务器 URL
func getServerURL() string {
	// 从环境变量获取
	serverURL := os.Getenv("MML_SERVER_URL")
	if serverURL != "" {
		return serverURL
	}

	// 从配置文件读取
	cfg, err := config.Load("")
	if err == nil && cfg.Server.Enabled {
		return fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)
	}

	// 使用默认值
	return mmlclient.DefaultServerURL
}

func main() {
	serverURL := getServerURL()
	app := NewApp(serverURL)

	// 检查命令行参数（非交互式模式）
	if len(os.Args) > 1 {
		commandName := strings.ToLower(os.Args[1])
		args := os.Args[2:]

		// 特殊处理 help 命令
		if commandName == "help" || commandName == "h" || commandName == "--help" || commandName == "-h" {
			if len(args) > 0 {
				fmt.Println(app.registry.HelpForCommand(args[0]))
			} else {
				fmt.Println(app.registry.Help())
			}
			return
		}

		// 执行命令
		if err := app.runCommand(commandName, args); err != nil {
			fmt.Printf("错误: %v\n", err)
			os.Exit(1)
		}
		return
	}

	// 交互式模式
	app.runInteractive()
}
