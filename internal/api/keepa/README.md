# Keepa API 接口框架

本目录包含了 Keepa API 的所有接口实现框架。每个接口都有独立的目录结构，便于维护和扩展。

## 目录结构

```
internal/api/keepa/
├── best_sellers/           # Best Sellers 接口
│   ├── best_sellers.go
│   └── best_sellers_test.go
├── browsing_deals/         # Browsing Deals 接口
│   ├── browsing_deals.go
│   └── browsing_deals_test.go
├── category_lookup/        # Category Lookup 接口
│   ├── category_lookup.go
│   └── category_lookup_test.go
├── category_searches/      # Category Searches 接口
│   ├── category_searches.go
│   └── category_searches_test.go
├── graph_image_api/        # Graph Image API 接口
│   ├── graph_image_api.go
│   └── graph_image_api_test.go
├── lightning_deals/        # Lightning Deals 接口
│   ├── lightning_deals.go
│   └── lightning_deals_test.go
├── most_rated_sellers/     # Most Rated Sellers 接口
│   ├── most_rated_sellers.go
│   └── most_rated_sellers_test.go
├── product_finder/         # Product Finder 接口
│   ├── product_finder.go
│   └── product_finder_test.go
├── product_searches/       # Product Searches 接口
│   ├── product_searches.go
│   └── product_searches_test.go
├── products/               # Products 接口
│   ├── products.go
│   └── products_test.go
├── seller_information/     # Seller Information 接口
│   ├── seller_information.go
│   └── seller_information_test.go
└── tracking_products/      # Tracking Products 接口
    ├── tracking_products.go
    └── tracking_products_test.go
```

## 接口列表

1. **Best Sellers** - 获取畅销商品列表
2. **Browsing Deals** - 获取浏览交易信息
3. **Category Lookup** - 类别查找
4. **Category Searches** - 类别搜索
5. **Graph Image API** - 获取价格图表图片
6. **Lightning Deals** - 获取限时抢购信息
7. **Most Rated Sellers** - 获取最受好评的卖家
8. **Product Finder** - 商品查找器
9. **Product Searches** - 商品搜索
10. **Products** - 获取商品信息
11. **Seller Information** - 获取卖家信息
12. **Tracking Products** - 获取跟踪商品信息

## 通用结构

每个接口都遵循相同的结构：

### Service 结构

```go
type Service struct {
    client *api.Client    // HTTP 客户端
    logger *zap.Logger    // 日志记录器
}
```

### 主要方法

- `NewService()` - 创建服务实例
- `Fetch()` - 获取数据（返回原始字节数据，不进行存储）

### RequestParams

每个接口都有自己的 `RequestParams` 结构体，用于定义请求参数。

## 使用示例

### 基本使用（只获取数据）

```go
import (
    "context"
    "keepa/internal/api"
    "keepa/internal/api/keepa/best_sellers"
    "go.uber.org/zap"
)

// 初始化（BaseURL 使用代码中的常量，无需指定）
client := api.NewClient(api.Config{
    AccessKey: "your-access-key",
    Logger:    logger,
})

// 创建服务
service := best_sellers.NewService(client, logger)

// 获取数据
params := best_sellers.RequestParams{
    Domain: 1, // US
}
data, err := service.Fetch(context.Background(), params)
if err != nil {
    // 处理错误
}
// data 是 []byte 类型的原始数据
```

### 在任务中使用（获取并存储）

```go
import (
    "context"
    "keepa/internal/api"
    "keepa/internal/api/keepa/best_sellers"
    "keepa/internal/database"
    "go.uber.org/zap"
)

// 在任务中
func (t *BestSellersTask) Run(ctx context.Context) error {
    // 创建服务
    service := best_sellers.NewService(t.client, t.logger)
    
    // 获取数据
    params := best_sellers.RequestParams{
        Domain: 1,
    }
    data, err := service.Fetch(ctx, params)
    if err != nil {
        return err
    }
    
    // 在上层进行存储
    mongoDB, _ := database.GetMongoDB()
    storage := api.NewStorage(mongoDB, t.logger)
    collectionName := "best_sellers"
    err = storage.SaveRawData(ctx, collectionName, data)
    if err != nil {
        return err
    }
    
    return nil
}
```

## 架构说明

### 职责分离

API 接口层只负责：
- 请求参数验证
- 调用 Keepa API 获取数据
- 返回原始数据（[]byte）

存储功能由上层（tasks 层）负责：
- 调用 API 接口获取数据
- 决定如何存储数据
- 处理存储错误和重试逻辑

### MongoDB 集合命名建议

如果在上层使用 MongoDB 存储，建议使用以下集合名称：

- `best_sellers` - Best Sellers 数据
- `browsing_deals` - Browsing Deals 数据
- `category_lookup` - Category Lookup 数据
- `category_searches` - Category Searches 数据
- `graph_images` - Graph Image 数据
- `lightning_deals` - Lightning Deals 数据
- `most_rated_sellers` - Most Rated Sellers 数据
- `product_finder` - Product Finder 数据
- `product_searches` - Product Searches 数据
- `products` - Products 数据
- `seller_information` - Seller Information 数据
- `tracking_products` - Tracking Products 数据

## 开发指南

### 实现接口

1. 完善 `RequestParams` 结构体，添加所有需要的字段
2. 在 `Fetch` 方法中实现：
   - 构建请求参数（转换为 `map[string]string` 或 JSON body）
   - 调用 `client.GetRawData()` 或 `client.PostRawData()` 获取原始数据
   - 返回原始数据（[]byte）和错误
3. 添加适当的错误处理和日志记录

### 编写测试

1. 创建 mock 客户端
2. 测试各种参数组合
3. 测试错误情况
4. 验证返回的数据格式

## 配置

在 `configs/config.yaml` 中配置 Keepa API：

```yaml
keepa_api:
  access_key: your-access-key
  timeout: 30s
```

注意：Base URL (`https://api.keepa.com/`) 已在代码中定义为常量，无需在配置文件中指定。

## 注意事项

1. 所有接口目前都是框架代码，需要根据 Keepa API 文档完善实现
2. 每个接口都应该有相应的单元测试
3. API 接口只负责获取数据，存储功能由上层实现
4. 返回的数据为原始 JSON 格式（[]byte），上层可以根据需要进行解析和结构化存储
5. 注意 API 的速率限制，必要时在上层添加重试机制
6. 存储逻辑应该在上层（tasks）统一管理，便于统一处理错误、重试和监控

