package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"keepa/internal/api"
	"keepa/internal/api/keepa/category_lookup"
	"keepa/internal/config"
	"keepa/internal/database"
	"keepa/internal/model"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// DependenciesProvider 依赖提供者接口
type DependenciesProvider interface {
	GetConfig() *config.Config
	GetQueriesConfig() *config.KeepaQueriesConfig
	GetLogger() *zap.Logger
}

// SuccessResponse 成功响应
type SuccessResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// ErrorResponse 错误响应
type ErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Error   string `json:"error,omitempty"`
}

// CategoryTreeHandler 分类树处理器
type CategoryTreeHandler struct {
	logger        *zap.Logger
	apiClient     *api.Client
	categorySvc   *category_lookup.Service
	storage       *database.Storage
	config        *config.Config
	queriesConfig *config.KeepaQueriesConfig
}

// NewCategoryTreeHandler 创建分类树处理器
func NewCategoryTreeHandler(
	logger *zap.Logger,
	apiClient *api.Client,
	categorySvc *category_lookup.Service,
	storage *database.Storage,
	cfg *config.Config,
	queriesConfig *config.KeepaQueriesConfig,
) *CategoryTreeHandler {
	return &CategoryTreeHandler{
		logger:        logger,
		apiClient:     apiClient,
		categorySvc:   categorySvc,
		storage:       storage,
		config:        cfg,
		queriesConfig: queriesConfig,
	}
}

// NewCategoryTreeHandlerWithDeps 通过依赖创建分类树处理器
func NewCategoryTreeHandlerWithDeps(deps DependenciesProvider) *CategoryTreeHandler {
	if deps == nil {
		return nil
	}

	var cfg *config.Config
	var queriesConfig *config.KeepaQueriesConfig
	var logger *zap.Logger

	cfg = deps.GetConfig()
	queriesConfig = deps.GetQueriesConfig()
	logger = deps.GetLogger()

	if cfg == nil || logger == nil {
		return nil
	}

	// 获取 MongoDB 数据库
	var storage *database.Storage
	dbs := database.GetGlobal()
	if dbs != nil && dbs.MongoDB != nil {
		// 创建 Storage
		storage = database.NewStorage(dbs.MongoDB, logger)
	} else {
		// MongoDB 不可用，记录警告但不阻止 handler 创建
		// handler 仍然可以工作，只是保存功能不可用
		logger.Warn("MongoDB not available, database save functionality will be disabled")
		storage = nil
	}

	// 创建 API Client
	timeout, err := time.ParseDuration(cfg.KeepaAPI.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	// 创建 Token Manager（如果配置启用）
	var tokenManager *api.TokenManager
	if cfg.KeepaAPI.Token.EnableRateLimit {
		tokenManager = api.NewTokenManager(api.TokenManagerConfig{
			MinTokensThreshold: cfg.KeepaAPI.Token.MinTokensThreshold,
			MaxWaitTime:        cfg.KeepaAPI.Token.MaxWaitTimeDuration,
			EnableRateLimit:    cfg.KeepaAPI.Token.EnableRateLimit,
			Logger:             logger,
		})
	}

	apiClient := api.NewClient(api.Config{
		AccessKey:         cfg.KeepaAPI.AccessKey,
		Timeout:           timeout,
		Logger:            logger,
		PrintCurlCommand:  cfg.KeepaAPI.PrintCurlCommand,
		PrintResponseBody: cfg.KeepaAPI.PrintResponseBody,
		TokenManager:      tokenManager,
	})

	// 创建 Category Lookup Service
	categorySvc := category_lookup.NewService(apiClient, logger)

	return &CategoryTreeHandler{
		logger:        logger,
		apiClient:     apiClient,
		categorySvc:   categorySvc,
		storage:       storage,
		config:        cfg,
		queriesConfig: queriesConfig,
	}
}

// JSONSuccess 返回成功响应
func (h *CategoryTreeHandler) JSONSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, SuccessResponse{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// JSONError 返回错误响应
func (h *CategoryTreeHandler) JSONError(c *gin.Context, code int, message string, err error) {
	response := ErrorResponse{
		Code:    code,
		Message: message,
	}
	if err != nil {
		response.Error = err.Error()
		h.logger.Error(message, zap.Error(err))
	}
	c.JSON(http.StatusOK, response)
}

// JSONBadRequest 返回400错误
func (h *CategoryTreeHandler) JSONBadRequest(c *gin.Context, message string, err error) {
	response := ErrorResponse{
		Code:    400,
		Message: message,
	}
	if err != nil {
		response.Error = err.Error()
		h.logger.Error(message, zap.Error(err))
	}
	c.JSON(http.StatusBadRequest, response)
}

// JSONInternalError 返回500错误
func (h *CategoryTreeHandler) JSONInternalError(c *gin.Context, message string, err error) {
	response := ErrorResponse{
		Code:    500,
		Message: message,
	}
	if err != nil {
		response.Error = err.Error()
		h.logger.Error(message, zap.Error(err))
	}
	c.JSON(http.StatusInternalServerError, response)
}

// CategoryTreeRequest 请求结构
type CategoryTreeRequest struct {
	TaskName   string  `json:"task_name,omitempty"`   // 可选，如果提供则从配置读取
	CategoryID []int64 `json:"category_id,omitempty"` // 可选，如果提供则使用此参数，否则从配置读取
	SaveToDB   *bool   `json:"save_to_db,omitempty"`  // 可选，是否保存到数据库，默认为 false（测试模式）
}

// CategoryTreeResponse 响应结构
type CategoryTreeResponse struct {
	TaskID      string                `json:"task_id"`
	RootTrees   []*model.CategoryTree `json:"root_trees"`
	TotalNodes  int                   `json:"total_nodes"`
	Collections map[string]int        `json:"collections"`
}

// BuildCategoryTree 构建分类树
func (h *CategoryTreeHandler) BuildCategoryTree(c *gin.Context) {
	ctx := c.Request.Context()

	// 解析请求体
	var req CategoryTreeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		// 如果请求体为空或解析失败，继续使用配置
		req = CategoryTreeRequest{}
	}

	// 读取配置
	queriesConfig, err := config.LoadKeepaQueriesConfig("", h.logger)
	if err != nil {
		h.JSONInternalError(c, "failed to load queries config", err)
		return
	}

	// 获取 category_lookup 配置
	categoryLookupConfig := queriesConfig.CategoryLookup
	if !categoryLookupConfig.Enabled {
		h.JSONError(c, 400, "category_lookup is not enabled in config", nil)
		return
	}

	// 确定任务名称
	taskName := req.TaskName
	if taskName == "" {
		taskName = "us_specific_categories" // 默认任务名
	}

	// 查找任务配置
	var taskConfig *config.KeepaQueryTaskConfig
	for i := range categoryLookupConfig.Tasks {
		if categoryLookupConfig.Tasks[i].Name == taskName {
			taskConfig = &categoryLookupConfig.Tasks[i]
			break
		}
	}

	if taskConfig == nil {
		h.JSONError(c, 404, fmt.Sprintf("task '%s' not found in config", taskName), nil)
		return
	}

	// 获取 domain
	domain := 1 // 默认 US
	if d, ok := taskConfig.Params["domain"]; ok {
		if domainInt, ok := d.(int); ok {
			domain = domainInt
		} else if domainFloat, ok := d.(float64); ok {
			domain = int(domainFloat)
		}
	}

	// 确定 category IDs：优先使用请求中的参数，否则从配置读取
	var categoryIDs []int
	if len(req.CategoryID) > 0 {
		// 使用请求中提供的 category_id
		categoryIDs = make([]int, len(req.CategoryID))
		for i, id := range req.CategoryID {
			categoryIDs[i] = int(id)
		}
	} else {
		// 从配置中读取
		params, ok := taskConfig.Params["category"]
		if !ok {
			h.JSONError(c, 400, "category parameter not found in task config and not provided in request", nil)
			return
		}

		// 转换 category 为 int 数组
		switch v := params.(type) {
		case []interface{}:
			categoryIDs = make([]int, len(v))
			for i, item := range v {
				if id, ok := item.(int); ok {
					categoryIDs[i] = id
				} else if id, ok := item.(float64); ok {
					categoryIDs[i] = int(id)
				} else {
					h.JSONError(c, 400, fmt.Sprintf("invalid category ID at index %d", i), nil)
					return
				}
			}
		case []int:
			categoryIDs = v
		default:
			h.JSONError(c, 400, "category must be an array of integers", nil)
			return
		}
	}

	h.logger.Info("building category tree",
		zap.Ints("root_category_ids", categoryIDs),
		zap.Int("domain", domain),
	)

	// 构建分类树
	rootTrees := make([]*model.CategoryTree, 0, len(categoryIDs))
	visited := make(map[int64]bool) // 防止重复访问
	totalNodes := 0

	for _, rootID := range categoryIDs {
		rootID64 := int64(rootID)
		if visited[rootID64] {
			continue
		}
		rootTree, count, err := h.buildCategoryTreeRecursive(ctx, rootID64, domain, visited)
		if err != nil {
			h.logger.Error("failed to build category tree for root",
				zap.Int64("root_id", rootID64),
				zap.Error(err),
			)
			continue
		}
		rootTrees = append(rootTrees, rootTree)
		totalNodes += count
	}

	// 存储到 MongoDB（仅在 SaveToDB 为 true 时保存）
	collections := make(map[string]int)
	saveToDB := req.SaveToDB != nil && *req.SaveToDB

	if saveToDB {
		// 检查 MongoDB 是否可用
		if h.storage == nil {
			h.JSONError(c, 503, "MongoDB is not available, cannot save to database", nil)
			return
		}

		collectionName := categoryLookupConfig.Collection
		if collectionName == "" {
			collectionName = "category_tree"
		}

		// 为每个根节点创建存储文档
		for _, rootTree := range rootTrees {
			// 使用 cat_id 和 domain_id 作为唯一标识
			filter := bson.M{
				"category.cat_id":    rootTree.Category.CatID,
				"category.domain_id": rootTree.Category.DomainID,
			}

			// 转换为 BSON 文档
			doc := convertCategoryTreeToBSON(rootTree)

			if err := h.storage.SaveRawDataWithFilter(ctx, collectionName, filter, doc); err != nil {
				h.logger.Error("failed to save category tree node",
					zap.Int64("cat_id", rootTree.Category.CatID),
					zap.Error(err),
				)
			} else {
				collections[collectionName] = collections[collectionName] + 1
			}
		}
	} else {
		h.logger.Info("skipping database save (test mode)")
	}

	response := CategoryTreeResponse{
		TaskID:      fmt.Sprintf("category_tree_%d", time.Now().Unix()),
		RootTrees:   rootTrees,
		TotalNodes:  totalNodes,
		Collections: collections,
	}

	h.JSONSuccess(c, response)
}

// buildCategoryTreeRecursive 递归构建分类树
func (h *CategoryTreeHandler) buildCategoryTreeRecursive(
	ctx context.Context,
	categoryID int64,
	domain int,
	visited map[int64]bool,
) (*model.CategoryTree, int, error) {
	// 防止重复访问
	if visited[categoryID] {
		return nil, 0, fmt.Errorf("category %d already visited (circular reference)", categoryID)
	}
	visited[categoryID] = true

	// 调用 API 获取分类信息
	params := category_lookup.RequestParams{
		Domain:   domain,
		Category: []int{int(categoryID)},
	}

	rawData, err := h.categorySvc.Fetch(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to fetch category %d: %w", categoryID, err)
	}

	// 解析 API 响应为 Category 对象
	category, err := h.parseCategoryResponse(rawData, categoryID, domain)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to parse category response: %w", err)
	}

	// 创建分类树节点
	tree := &model.CategoryTree{
		Category:  category,
		Children:  make([]*model.CategoryTree, 0),
		CreatedAt: time.Now().Unix(),
	}

	// 递归处理子分类
	// 当 category.Children 为空时，循环不会执行，递归自然停止
	totalCount := 1 // 当前节点
	for _, childID := range category.Children {
		childTree, count, err := h.buildCategoryTreeRecursive(ctx, childID, domain, visited)
		if err != nil {
			h.logger.Warn("failed to fetch child category",
				zap.Int64("parent_id", categoryID),
				zap.Int64("child_id", childID),
				zap.Error(err),
			)
			continue
		}
		tree.Children = append(tree.Children, childTree)
		totalCount += count
	}

	return tree, totalCount, nil
}

// parseCategoryResponse 解析 API 响应为 Category 对象
// API 返回格式: { "categories": { "2975221011": { ... }, "2975238011": { ... } }, ... }
func (h *CategoryTreeHandler) parseCategoryResponse(rawData []byte, expectedCatID int64, domain int) (*model.Category, error) {
	// 解析顶层响应对象
	var apiResponse map[string]interface{}
	if err := json.Unmarshal(rawData, &apiResponse); err != nil {
		return nil, fmt.Errorf("failed to parse API response: %w", err)
	}

	// 获取 categories 字段
	categoriesObj, ok := apiResponse["categories"]
	if !ok {
		return nil, fmt.Errorf("categories field not found in API response")
	}

	// categories 是一个对象，key 是字符串形式的 category ID，value 是分类对象
	categoriesMap, ok := categoriesObj.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("categories field is not an object")
	}

	// 查找匹配的分类（使用字符串形式的 ID）
	expectedCatIDStr := fmt.Sprintf("%d", expectedCatID)
	categoryObj, found := categoriesMap[expectedCatIDStr]
	if !found {
		// 如果没有找到精确匹配，尝试使用第一个可用的分类
		if len(categoriesMap) == 0 {
			return nil, fmt.Errorf("no category data returned for category %d", expectedCatID)
		}
		// 使用第一个分类
		for _, cat := range categoriesMap {
			categoryObj = cat
			break
		}
		h.logger.Warn("category ID mismatch, using first available category",
			zap.Int64("requested_id", expectedCatID),
		)
	}

	// 将分类对象转换为 JSON 字节，然后解析为 Category 结构
	categoryBytes, err := json.Marshal(categoryObj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal category object: %w", err)
	}

	var category model.Category
	if err := json.Unmarshal(categoryBytes, &category); err != nil {
		return nil, fmt.Errorf("failed to unmarshal category: %w", err)
	}

	// 确保 DomainID 设置正确
	if category.DomainID == 0 {
		category.DomainID = domain
	}

	// 验证 CatID 是否匹配
	if category.CatID != expectedCatID && found {
		h.logger.Warn("category ID mismatch",
			zap.Int64("requested_id", expectedCatID),
			zap.Int64("found_id", category.CatID),
		)
	}

	return &category, nil
}

// convertCategoryTreeToBSON 将分类树转换为 BSON 格式
func convertCategoryTreeToBSON(tree *model.CategoryTree) bson.M {
	doc := bson.M{
		"category":   tree.Category,
		"created_at": tree.CreatedAt,
	}

	if len(tree.Children) > 0 {
		children := make([]bson.M, len(tree.Children))
		for i, child := range tree.Children {
			children[i] = convertCategoryTreeToBSON(child)
		}
		doc["children"] = children
	}

	return doc
}
