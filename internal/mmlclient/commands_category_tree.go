package mmlclient

import (
	"context"
	"fmt"
	"strconv"

	"keepa/internal/model"
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

// CategoryTreeResponse 响应结构（与 handler 中的结构对应）
type CategoryTreeResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		TaskID      string                `json:"task_id"`
		RootTrees   []*model.CategoryTree `json:"root_trees"`
		TotalNodes  int                   `json:"total_nodes"`
		Collections map[string]int        `json:"collections"`
	} `json:"data"`
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

	// 构建请求体（测试模式：不保存到数据库）
	saveToDB := false
	reqBody := make(map[string]interface{})
	if len(categoryIDs) > 0 {
		reqBody["category_id"] = categoryIDs
	}
	reqBody["save_to_db"] = saveToDB

	fmt.Println("正在构建分类树...")

	// 发送请求并解析响应
	var response CategoryTreeResponse
	if err := c.client.PostJSONAndUnmarshal("/api/v1/functions/category-tree", reqBody, &response); err != nil {
		return err
	}

	// 检查响应状态
	if response.Code != 0 {
		return fmt.Errorf("服务器返回错误: %s", response.Message)
	}

	// 按层次关系展示分类树
	c.displayCategoryTree(response.Data.RootTrees, response.Data.TotalNodes)

	return nil
}

// displayCategoryTree 按层次关系展示分类树
func (c *CategoryTreeCommand) displayCategoryTree(rootTrees []*model.CategoryTree, totalNodes int) {
	if len(rootTrees) == 0 {
		fmt.Println("没有找到分类树")
		return
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Printf("分类树展示 (共 %d 个节点)\n", totalNodes)
	fmt.Println("═══════════════════════════════════════════════════════════")
	fmt.Println()

	for i, rootTree := range rootTrees {
		if len(rootTrees) > 1 {
			fmt.Printf("【根分类树 %d】\n", i+1)
			fmt.Println()
		}
		c.displayTreeRecursive(rootTree, "", true)
		if i < len(rootTrees)-1 {
			fmt.Println()
		}
	}

	fmt.Println()
	fmt.Println("═══════════════════════════════════════════════════════════")
}

// displayTreeRecursive 递归展示分类树节点
func (c *CategoryTreeCommand) displayTreeRecursive(tree *model.CategoryTree, prefix string, isLast bool) {
	if tree == nil || tree.Category == nil {
		return
	}

	// 确定当前节点的连接符
	var connector string
	if prefix == "" {
		// 根节点
		connector = ""
	} else if isLast {
		connector = "└── "
	} else {
		connector = "├── "
	}

	// 显示当前节点
	category := tree.Category
	fmt.Printf("%s%s[%d] %s\n", prefix, connector, category.CatID, category.Name)

	// 显示额外信息（如果有）
	if category.ProductCount > 0 {
		fmt.Printf("%s    └─ 产品数: %d\n", prefix, category.ProductCount)
	}

	// 准备子节点的前缀
	var childPrefix string
	if prefix == "" {
		childPrefix = ""
	} else if isLast {
		childPrefix = prefix + "    "
	} else {
		childPrefix = prefix + "│   "
	}

	// 递归显示子节点
	children := tree.Children
	for i, child := range children {
		isLastChild := i == len(children)-1
		c.displayTreeRecursive(child, childPrefix, isLastChild)
	}
}
