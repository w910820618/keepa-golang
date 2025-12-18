# Keepa API 集成文档

本文档说明如何使用 Keepa API 接口框架。

## 概述

本项目为 Keepa API 的所有接口搭建了完整的框架结构。每个接口都有独立的目录，包含：
- 服务实现文件（`*.go`）
- 单元测试文件（`*_test.go`）

## 架构设计

### 三层架构

```
┌─────────────────────────────────────┐
│      Service Layer                  │  ← 业务逻辑层（各接口服务）
│  (internal/api/keepa/*/)           │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      API Client Layer               │  ← HTTP 请求层
│  (internal/api/client.go)          │
└──────────────┬──────────────────────┘
               │
┌──────────────▼──────────────────────┐
│      Storage Layer                  │  ← 数据存储层
│  (internal/api/storage.go)         │
└─────────────────────────────────────┘
```

### 组件说明

#### 1. API Client (`internal/api/client.go`)

负责与 Keepa API 服务器通信：
- `DoRequest()` - 执行 HTTP 请求
- `GetRawData()` - 获取原始 JSON 数据

#### 2. Storage (`internal/api/storage.go`)

负责将数据存储到 MongoDB：
- `SaveRawData()` - 保存原始数据
- `SaveRawDataWithFilter()` - 保存或更新数据（upsert）

#### 3. Service (`internal/api/keepa/*/`)

每个接口都有自己的服务：
- `NewService()` - 创建服务实例
- `FetchAndStore()` - 获取数据并存储

## 接口列表

| 接口名称 | 目录 | 说明 |
|---------|------|------|
| Best Sellers | `best_sellers/` | 获取畅销商品列表 |
| Browsing Deals | `browsing_deals/` | 获取浏览交易信息 |
| Category Lookup | `category_lookup/` | 类别查找 |
| Category Searches | `category_searches/` | 类别搜索 |
| Graph Image API | `graph_image_api/` | 获取价格图表图片 |
| Lightning Deals | `lightning_deals/` | 获取限时抢购信息 |
| Most Rated Sellers | `most_rated_sellers/` | 获取最受好评的卖家 |
| Product Finder | `product_finder/` | 商品查找器 |
| Product Searches | `product_searches/` | 商品搜索 |
| Products | `products/` | 获取商品信息 |
| Seller Information | `seller_information/` | 获取卖家信息 |
| Tracking Products | `tracking_products/` | 获取跟踪商品信息 |

## 配置

### 配置文件

在 `configs/config.yaml` 中添加 Keepa API 配置：

```yaml
keepa_api:
  access_key: your-access-key-here
  timeout: 30s
```

注意：Base URL (`https://api.keepa.com/`) 已在代码中定义为常量 `api.KeepaBaseURL`，无需在配置文件中指定。

### 环境变量

也可以通过环境变量配置：

```bash
export KEEPA_KEEPA_API_ACCESS_KEY=your-access-key
export KEEPA_KEEPA_API_TIMEOUT=30s
```

## 使用示例

### 基本使用

```go
package main

import (
    "context"
    "keepa/internal/api"
    "keepa/internal/api/keepa/products"
    "keepa/internal/config"
    "keepa/internal/database"
    "keepa/internal/logger"
)

func main() {
    // 加载配置
    cfg, _ := config.Load("")
    
    // 初始化日志
    zapLogger, _ := logger.NewLogger(logger.LoggerConfig{
        Level:  cfg.Logger.Level,
        Format: cfg.Logger.Format,
    })
    
    // 获取 MongoDB 连接
    mongoDB, _ := database.GetMongoDB()
    
    // 创建 API 客户端（BaseURL 使用代码中的常量，无需指定）
    timeout, _ := time.ParseDuration(cfg.KeepaAPI.Timeout)
    client := api.NewClient(api.Config{
        AccessKey: cfg.KeepaAPI.AccessKey,
        Timeout:   timeout,
        Logger:    zapLogger,
    })
    
    // 创建存储
    storage := api.NewStorage(mongoDB, zapLogger)
    
    // 创建 Products 服务
    productsService := products.NewService(client, storage, zapLogger)
    
    // 获取数据并存储
    ctx := context.Background()
    params := products.RequestParams{
        ASINs:  []string{"B000123456", "B000789012"},
        Domain: 1, // US
    }
    
    err := productsService.FetchAndStore(ctx, params)
    if err != nil {
        zapLogger.Error("failed to fetch products", zap.Error(err))
    }
}
```

### 在定时任务中使用

```go
package tasks

import (
    "context"
    "keepa/internal/api"
    "keepa/internal/api/keepa/best_sellers"
    "keepa/internal/database"
    "keepa/internal/task"
    "go.uber.org/zap"
)

type BestSellersTask struct {
    client  *api.Client
    storage *api.Storage
    logger  *zap.Logger
}

func (t *BestSellersTask) Run(ctx context.Context) error {
    mongoDB, err := database.GetMongoDB()
    if err != nil {
        return err
    }
    
    service := best_sellers.NewService(t.client, t.storage, t.logger)
    params := best_sellers.RequestParams{
        Domain: 1,
    }
    
    return service.FetchAndStore(ctx, params)
}
```

## 实现步骤

### 第一步：完善 RequestParams

根据 Keepa API 文档，为每个接口的 `RequestParams` 添加所有必要的字段：

```go
type RequestParams struct {
    Domain   int    `json:"domain"`
    Category int    `json:"category,omitempty"`
    // 添加其他字段...
}
```

### 第二步：实现 FetchAndStore

1. 将 `RequestParams` 转换为 `map[string]string`
2. 调用 `client.GetRawData()` 获取数据
3. 调用 `storage.SaveRawData()` 存储数据

```go
func (s *Service) FetchAndStore(ctx context.Context, params RequestParams) error {
    // 转换参数
    requestParams := map[string]string{
        "domain": strconv.Itoa(params.Domain),
        // ...
    }
    
    // 获取数据
    rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
    if err != nil {
        return err
    }
    
    // 存储数据
    return s.storage.SaveRawData(ctx, collectionName, rawData)
}
```

### 第三步：编写测试

为每个接口编写单元测试：

```go
func TestService_FetchAndStore(t *testing.T) {
    // 创建 mock
    // 测试各种场景
    // 验证结果
}
```

## 数据存储

### MongoDB 集合

每个接口的数据存储在对应的 MongoDB 集合中，集合名称与接口目录名称一致（使用下划线命名）。

### 数据格式

目前所有数据以原始 JSON 格式存储。后续可以：
1. 解析 JSON 数据
2. 创建结构化的数据模型
3. 进行数据验证和清洗

## 错误处理

所有接口都应该：
1. 记录详细的错误日志
2. 返回有意义的错误信息
3. 处理网络超时和重试
4. 验证 API 响应状态码

## 最佳实践

1. **参数验证**：在发送请求前验证所有必需参数
2. **错误处理**：捕获并记录所有可能的错误
3. **日志记录**：记录关键操作和错误信息
4. **测试覆盖**：为每个接口编写完整的单元测试
5. **文档更新**：及时更新接口文档和注释

## 后续扩展

1. 添加数据解析和结构化存储
2. 实现数据缓存机制
3. 添加 API 速率限制控制
4. 实现批量操作支持
5. 添加数据同步和增量更新

