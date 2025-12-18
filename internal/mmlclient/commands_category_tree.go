package mmlclient

import (
	"context"
	"fmt"
	"strconv"
	// 如果需要使用 model 包中的结构体，可以取消注释：
	// "keepa/internal/model"
)

// CategoryTreeCommand 分类树构建命令
type CategoryTreeCommand struct {
	client *Client
}

// NewCategoryTreeCommand 创建分类树命令
func NewCategoryTreeCommand(client *Client) *CategoryTreeCommand {
	return &CategoryTreeCommand{
		client: client,
	}
}

// Name 返回命令名称
func (c *CategoryTreeCommand) Name() string {
	return "category-tree"
}

// Aliases 返回命令别名
func (c *CategoryTreeCommand) Aliases() []string {
	return []string{"ct"}
}

// Description 返回命令描述
func (c *CategoryTreeCommand) Description() string {
	return "构建分类树"
}

// Usage 返回使用说明
func (c *CategoryTreeCommand) Usage() string {
	return "category-tree [category_id...]\n" +
		"  如果不提供 category_id，则从配置文件读取\n" +
		"  可以提供一个或多个 category_id\n" +
		"  示例:\n" +
		"    category-tree                 # 使用配置文件中的 category_id\n" +
		"    category-tree 1036592        # 使用指定的 category_id\n" +
		"    category-tree 1036592 1036684 # 使用多个 category_id"
}

// Execute 执行命令
func (c *CategoryTreeCommand) Execute(ctx context.Context, args []string) error {
	// 解析 category_id 参数
	var categoryIDs []int64
	if len(args) > 0 {
		categoryIDs = make([]int64, 0, len(args))
		for _, arg := range args {
			id, err := strconv.ParseInt(arg, 10, 64)
			if err != nil {
				return fmt.Errorf("invalid category ID: %s", arg)
			}
			categoryIDs = append(categoryIDs, id)
		}
	}

	// 构建请求体
	reqBody := make(map[string]interface{})
	if len(categoryIDs) > 0 {
		reqBody["category_id"] = categoryIDs
	}

	fmt.Println("正在构建分类树...")
	return c.client.PostJSON("/api/v1/functions/category-tree", reqBody)
}
