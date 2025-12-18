package function

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"keepa/internal/config"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Dependencies 服务器依赖
type Dependencies struct {
	Config        *config.Config
	QueriesConfig *config.KeepaQueriesConfig
	Logger        *zap.Logger
}

// GetConfig 获取配置
func (d *Dependencies) GetConfig() *config.Config {
	return d.Config
}

// GetQueriesConfig 获取查询配置
func (d *Dependencies) GetQueriesConfig() *config.KeepaQueriesConfig {
	return d.QueriesConfig
}

// GetLogger 获取日志器
func (d *Dependencies) GetLogger() *zap.Logger {
	return d.Logger
}

// Server HTTP 服务器
type Server struct {
	config       *config.ServerConfig
	router       *gin.Engine
	logger       *zap.Logger
	httpServer   *http.Server
	dependencies *Dependencies
}

// NewServer 创建新的 HTTP 服务器
func NewServer(cfg *config.ServerConfig, logger *zap.Logger) *Server {
	return NewServerWithDeps(cfg, logger, nil)
}

// NewServerWithDeps 创建新的 HTTP 服务器（带依赖）
func NewServerWithDeps(cfg *config.ServerConfig, logger *zap.Logger, deps *Dependencies) *Server {
	// 设置 gin 模式
	gin.SetMode(cfg.Mode)

	// 创建 gin 引擎
	router := gin.New()

	// 添加中间件
	router.Use(ginLogger(logger))
	router.Use(gin.Recovery())

	// 创建服务器实例
	server := &Server{
		config:       cfg,
		router:       router,
		logger:       logger,
		dependencies: deps,
	}

	// 设置路由
	routerManager := NewRouter(router, logger, deps)
	routerManager.SetupRoutes()

	// 创建 HTTP 服务器
	server.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return server
}

// Start 启动服务器
func (s *Server) Start() error {
	if !s.config.Enabled {
		s.logger.Info("HTTP server is disabled, skipping startup")
		return nil
	}

	s.logger.Info("starting HTTP server",
		zap.String("host", s.config.Host),
		zap.Int("port", s.config.Port),
		zap.String("mode", s.config.Mode),
	)

	// 在 goroutine 中启动服务器
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.logger.Fatal("failed to start HTTP server", zap.Error(err))
		}
	}()

	return nil
}

// Stop 停止服务器
func (s *Server) Stop(ctx context.Context) error {
	if !s.config.Enabled {
		return nil
	}

	s.logger.Info("shutting down HTTP server")

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
		s.logger.Error("error shutting down HTTP server", zap.Error(err))
		return err
	}

	s.logger.Info("HTTP server stopped")
	return nil
}

// RegisterCustomRoutes 注册自定义路由
// 这个方法允许外部注册额外的路由处理器
func (s *Server) RegisterCustomRoutes(registerFunc func(*gin.RouterGroup)) {
	routerManager := NewRouter(s.router, s.logger, s.dependencies)
	routerManager.RegisterCustomRoutes(registerFunc)
}

// ginLogger 自定义 gin 日志中间件
func ginLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 记录日志
		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		errorMessage := c.Errors.ByType(gin.ErrorTypePrivate).String()

		if raw != "" {
			path = path + "?" + raw
		}

		logger.Info("HTTP request",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("error", errorMessage),
		)
	}
}
