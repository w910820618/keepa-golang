package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"keepa/internal/api"
	"keepa/internal/api/keepa/products"
	"keepa/internal/model"
	"keepa/internal/repository"

	"go.uber.org/zap"
)

const (
	// DefaultBatchSize 默认批次大小
	DefaultBatchSize = 20
	// DefaultStatsValue 默认统计天数
	DefaultStatsValue = 90
	// DefaultDomainID 默认 Amazon 域名 (US)
	DefaultDomainID = 1
)

// KeepaFetchProducts 从 Keepa Product API 获取 ASIN 详情数据的任务
type KeepaFetchProducts struct {
	client *api.Client
	repo   *repository.KeepaRepository
	logger *zap.Logger
}

// NewKeepaFetchProducts 创建新的 KeepaFetchProducts 实例
func NewKeepaFetchProducts(client *api.Client, repo *repository.KeepaRepository, logger *zap.Logger) *KeepaFetchProducts {
	return &KeepaFetchProducts{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

// FetchProductDetails 获取 ASIN 详情数据
// 流程：
// 1. 从 MongoDB 查询 detail_fetched=false 的 ASIN（每批 20 个）
// 2. 调用 Product Object API，参数：stats=90, history=1, buybox=1
// 3. 解析返回的 csv 数组和 stats 对象
// 4. 保存原始响应到 MongoDB
// 5. 存入 keepa_products collection
// 6. 更新对应 ASIN 的 detail_fetched=true
func (f *KeepaFetchProducts) FetchProductDetails(ctx context.Context) error {
	return f.FetchProductDetailsWithBatchSize(ctx, DefaultBatchSize)
}

// FetchProductDetailsWithBatchSize 使用指定批次大小获取 ASIN 详情数据
func (f *KeepaFetchProducts) FetchProductDetailsWithBatchSize(ctx context.Context, batchSize int) error {
	if batchSize <= 0 || batchSize > 100 {
		batchSize = DefaultBatchSize
	}

	if f.logger != nil {
		f.logger.Info("starting to fetch product details from Keepa Product API",
			zap.Int("batch_size", batchSize),
		)
	}

	// 1. 从 MongoDB 查询 detail_fetched=false 的 ASIN
	pendingASINs, err := f.repo.GetPendingASINs(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get pending ASINs: %w", err)
	}

	if len(pendingASINs) == 0 {
		if f.logger != nil {
			f.logger.Info("no pending ASINs found")
		}
		return nil
	}

	// 提取 ASIN 码列表
	asinCodes := extractASINCodes(pendingASINs)
	if f.logger != nil {
		f.logger.Info("found pending ASINs",
			zap.Int("count", len(asinCodes)),
			zap.Strings("asins", asinCodes),
		)
	}

	// 2. 调用 Product API（返回原始响应）
	productDetails, rawData, err := f.fetchProductsFromAPI(ctx, asinCodes)
	if err != nil {
		return fmt.Errorf("failed to fetch products from API: %w", err)
	}

	if f.logger != nil {
		f.logger.Info("fetched product details from API",
			zap.Int("count", len(productDetails)),
		)
	}

	// 3. 保存原始响应到 MongoDB
	if rawData != nil {
		if err := f.saveRawResponse(ctx, asinCodes, rawData); err != nil {
			if f.logger != nil {
				f.logger.Warn("failed to save raw response to MongoDB",
					zap.Error(err),
				)
			}
		}
	}

	// 4. 存入 keepa_products collection
	if err := f.saveProductDetails(ctx, productDetails); err != nil {
		return fmt.Errorf("failed to save product details: %w", err)
	}

	// 5. 更新对应 ASIN 的 detail_fetched=true
	if err := f.markAsinsAsFetched(ctx, asinCodes); err != nil {
		return fmt.Errorf("failed to mark ASINs as fetched: %w", err)
	}

	if f.logger != nil {
		f.logger.Info("successfully fetched and saved product details",
			zap.Int("product_count", len(productDetails)),
		)
	}

	return nil
}

// FetchAllProductDetails 循环获取所有未获取详情的 ASIN
// 每批处理 batchSize 个 ASIN，直到所有 ASIN 都已获取详情
func (f *KeepaFetchProducts) FetchAllProductDetails(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 || batchSize > 100 {
		batchSize = DefaultBatchSize
	}

	totalFetched := 0

	for {
		select {
		case <-ctx.Done():
			return totalFetched, ctx.Err()
		default:
		}

		// 查询未获取详情的 ASIN
		pendingASINs, err := f.repo.GetPendingASINs(ctx, batchSize)
		if err != nil {
			return totalFetched, fmt.Errorf("failed to get pending ASINs: %w", err)
		}

		if len(pendingASINs) == 0 {
			if f.logger != nil {
				f.logger.Info("all ASINs have been fetched",
					zap.Int("total_fetched", totalFetched),
				)
			}
			break
		}

		// 提取 ASIN 码
		asinCodes := extractASINCodes(pendingASINs)

		// 获取产品详情（返回原始响应）
		productDetails, rawData, err := f.fetchProductsFromAPI(ctx, asinCodes)
		if err != nil {
			return totalFetched, fmt.Errorf("failed to fetch products from API: %w", err)
		}

		// 保存原始响应
		if rawData != nil {
			if err := f.saveRawResponse(ctx, asinCodes, rawData); err != nil {
				if f.logger != nil {
					f.logger.Warn("failed to save raw response to MongoDB",
						zap.Error(err),
					)
				}
			}
		}

		// 保存产品详情
		if err := f.saveProductDetails(ctx, productDetails); err != nil {
			return totalFetched, fmt.Errorf("failed to save product details: %w", err)
		}

		// 标记为已获取
		if err := f.markAsinsAsFetched(ctx, asinCodes); err != nil {
			return totalFetched, fmt.Errorf("failed to mark ASINs as fetched: %w", err)
		}

		totalFetched += len(productDetails)

		if f.logger != nil {
			f.logger.Info("batch completed",
				zap.Int("batch_size", len(asinCodes)),
				zap.Int("products_fetched", len(productDetails)),
				zap.Int("total_fetched", totalFetched),
			)
		}
	}

	return totalFetched, nil
}

// fetchProductsFromAPI 从 Keepa Product API 获取产品详情，返回产品详情和原始响应
func (f *KeepaFetchProducts) fetchProductsFromAPI(ctx context.Context, asins []string) ([]model.ProductDetails, []byte, error) {
	if len(asins) == 0 {
		return nil, nil, nil
	}

	// 构建请求参数
	statsValue := DefaultStatsValue
	history := 1
	buybox := 1

	params := products.RequestParams{
		Domain: DefaultDomainID,
		ASINs:  asins,
		Stats: &products.StatsValue{
			Days: &statsValue,
		},
		History: &history,
		Buybox:  &buybox,
	}

	// 创建 Products 服务
	service := products.NewService(f.client, f.logger)

	// 调用 API
	rawData, err := service.Fetch(ctx, params)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}

	// 解析响应
	var response model.ProductAPIResponse
	if err := json.Unmarshal(rawData, &response); err != nil {
		return nil, rawData, fmt.Errorf("failed to parse API response: %w", err)
	}

	if f.logger != nil {
		f.logger.Info("Product API response",
			zap.Int("product_count", len(response.Products)),
			zap.Int("tokens_consumed", response.TokensConsumed),
			zap.Int("tokens_left", response.TokensLeft),
		)
	}

	return response.Products, rawData, nil
}

// saveRawResponse 保存原始 API 响应到 MongoDB
func (f *KeepaFetchProducts) saveRawResponse(ctx context.Context, asins []string, rawData []byte) error {
	// 解析响应获取 token 信息
	var response model.ProductAPIResponse
	tokensUsed := 0
	tokensLeft := 0
	if err := json.Unmarshal(rawData, &response); err == nil {
		tokensUsed = response.TokensConsumed
		tokensLeft = response.TokensLeft
	}

	rawResponse := &model.KeepaRawResponse{
		APIEndpoint:  "/product",
		RequestType:  "product",
		DomainID:     DefaultDomainID,
		RequestData:  map[string]interface{}{"asins": asins, "stats": DefaultStatsValue},
		RawResponse:  string(rawData),
		ResponseSize: len(rawData),
		TokensUsed:   tokensUsed,
		TokensLeft:   tokensLeft,
	}

	if err := f.repo.SaveRawResponse(ctx, rawResponse); err != nil {
		return fmt.Errorf("failed to save raw response: %w", err)
	}

	if f.logger != nil {
		f.logger.Info("saved raw API response to MongoDB",
			zap.String("api_endpoint", "/product"),
			zap.Int("response_size", len(rawData)),
			zap.Int("asin_count", len(asins)),
		)
	}

	return nil
}

// saveProductDetails 保存产品详情到 MongoDB（使用 Repository）
func (f *KeepaFetchProducts) saveProductDetails(ctx context.Context, productDetails []model.ProductDetails) error {
	if len(productDetails) == 0 {
		return nil
	}

	for _, product := range productDetails {
		// 转换为 KeepaProduct 模型
		keepaProduct := convertToKeepaProduct(&product)

		// 使用 Repository 保存
		if err := f.repo.SaveProduct(ctx, keepaProduct); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save product",
					zap.String("asin", product.ASIN),
					zap.Error(err),
				)
			}
			// 继续处理其他产品
			continue
		}

		if f.logger != nil {
			f.logger.Debug("saved product details",
				zap.String("asin", product.ASIN),
				zap.String("title", product.Title),
			)
		}
	}

	return nil
}

// markAsinsAsFetched 标记 ASIN 记录为已获取详情（使用 Repository）
func (f *KeepaFetchProducts) markAsinsAsFetched(ctx context.Context, asins []string) error {
	if len(asins) == 0 {
		return nil
	}

	// 使用 Repository 批量更新
	if err := f.repo.BatchUpdateASINDetailStatus(ctx, asins, DefaultDomainID, true); err != nil {
		return fmt.Errorf("failed to update ASIN status: %w", err)
	}

	return nil
}

// GetPendingCount 获取待处理的 ASIN 数量
func (f *KeepaFetchProducts) GetPendingCount(ctx context.Context) (int64, error) {
	return f.repo.CountPendingASINs(ctx)
}

// GetCollectionName 返回存储产品详情的 MongoDB 集合名称
func (f *KeepaFetchProducts) GetCollectionName() string {
	return model.CollectionKeepaProducts
}

// GetSourceCollectionName 返回 ASIN 来源的 MongoDB 集合名称
func (f *KeepaFetchProducts) GetSourceCollectionName() string {
	return model.CollectionKeepaASINs
}

// GetRawResponseCollectionName 返回存储原始响应的 MongoDB 集合名称
func (f *KeepaFetchProducts) GetRawResponseCollectionName() string {
	return model.CollectionKeepaRawResponses
}

// extractASINCodes 从 KeepaASIN 列表中提取 ASIN 码
func extractASINCodes(asins []*model.KeepaASIN) []string {
	codes := make([]string, 0, len(asins))
	seen := make(map[string]bool)

	for _, asin := range asins {
		if asin != nil && asin.ASIN != "" && !seen[asin.ASIN] {
			seen[asin.ASIN] = true
			codes = append(codes, asin.ASIN)
		}
	}

	return codes
}

// convertToKeepaProduct 将 ProductDetails 转换为 KeepaProduct
func convertToKeepaProduct(details *model.ProductDetails) *model.KeepaProduct {
	if details == nil {
		return nil
	}

	product := &model.KeepaProduct{
		ASIN:         details.ASIN,
		DomainID:     details.DomainID,
		Title:        details.Title,
		Brand:        details.Brand,
		Manufacturer: details.Manufacturer,
		ProductGroup: details.ProductGroup,
		RootCategory: details.RootCategory,
		Categories:   details.Categories,

		// 尺寸和重量
		PackageHeight: details.PackageHeight,
		PackageLength: details.PackageLength,
		PackageWidth:  details.PackageWidth,
		PackageWeight: details.PackageWeight,
		ItemHeight:    details.ItemHeight,
		ItemLength:    details.ItemLength,
		ItemWidth:     details.ItemWidth,
		ItemWeight:    details.ItemWeight,

		// 状态标识
		IsPrime:        details.IsEligibleForPrime,
		IsAdultProduct: details.IsAdultProduct,

		// 历史数据
		CSV:   details.CSV,
		Stats: details.Stats,

		// 图片
		ImagesCSV: details.ImagesCSV,

		// 原始数据
		RawData: details,

		// Keepa 时间戳
		LastUpdate: details.LastUpdate,
	}

	// 从 Stats 中提取价格和排名信息
	if details.Stats != nil {
		// 提取 BuyBox 价格
		product.BuyBoxPrice = details.Stats.BuyBoxPrice
		product.BuyBoxShipping = details.Stats.BuyBoxShipping

		// 提取 offer 数量
		product.OfferCountNew = details.Stats.OfferCountFBA + details.Stats.OfferCountFBM
		product.TotalOfferCount = details.Stats.TotalOfferCount

		// 提取 FBA/Amazon 状态
		product.IsFBA = details.Stats.BuyBoxIsFBA
		product.IsAmazon = details.Stats.BuyBoxIsAmazon

		// 从 current 数组中提取价格（如果存在）
		if len(details.Stats.Current) > 0 {
			product.CurrentPrice = details.Stats.Current[0]
		}
		if len(details.Stats.Current) > 1 {
			product.NewPrice = details.Stats.Current[1]
		}
		if len(details.Stats.Current) > 2 {
			product.UsedPrice = details.Stats.Current[2]
		}
		if len(details.Stats.Current) > 3 {
			product.SalesRank = details.Stats.Current[3]
		}
	}

	return product
}
