package function

import (
	"keepa/internal/function/handlers"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Router 路由管理器
type Router struct {
	router       *gin.Engine
	logger       *zap.Logger
	dependencies *Dependencies
}

// NewRouter 创建路由管理器
func NewRouter(router *gin.Engine, logger *zap.Logger, deps *Dependencies) *Router {
	return &Router{
		router:       router,
		logger:       logger,
		dependencies: deps,
	}
}

// SetupRoutes 设置所有路由
func (r *Router) SetupRoutes() {
	// API v1 路由组
	api := r.router.Group("/api/v1")
	{
		// 一次性任务路由组
		functions := api.Group("/functions")
		{
			// 分类树构建处理器
			if r.dependencies != nil {
				categoryTreeHandler := handlers.NewCategoryTreeHandlerWithDeps(r.dependencies)
				if categoryTreeHandler != nil {
					functions.POST("/category-tree", categoryTreeHandler.BuildCategoryTree)
				}
			}
		}
	}
}

// RegisterCustomRoutes 注册自定义路由
// 这个方法允许外部注册额外的路由处理器
func (r *Router) RegisterCustomRoutes(registerFunc func(*gin.RouterGroup)) {
	api := r.router.Group("/api/v1")
	registerFunc(api)
}
