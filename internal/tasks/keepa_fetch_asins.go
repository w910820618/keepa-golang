package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"keepa/internal/api"
	"keepa/internal/model"
	"keepa/internal/repository"

	"go.uber.org/zap"
)

// KeepaFetchAsins 从 Keepa Product Finder API 获取 ASIN 列表的任务
type KeepaFetchAsins struct {
	client *api.Client
	repo   *repository.KeepaRepository
	logger *zap.Logger
}

// NewKeepaFetchAsins 创建新的 KeepaFetchAsins 实例
func NewKeepaFetchAsins(client *api.Client, repo *repository.KeepaRepository, logger *zap.Logger) *KeepaFetchAsins {
	return &KeepaFetchAsins{
		client: client,
		repo:   repo,
		logger: logger,
	}
}

// FetchPetSuppliesAsins 获取宠物用品 ASIN 列表
// 筛选条件：
// - rootCategory: 2619533011 (US Pet Supplies)
// - salesRank: min=500, max=10000
// - current_AMAZON: -1 (排除亚马逊自营)
// - offerCount: min=3, max=20
// - current_NEW: min=1500, max=6000 (价格 $15-$60，单位分)
// - rating: min=38 (3.8星)
// - 限制获取 100 个 ASIN
func (f *KeepaFetchAsins) FetchPetSuppliesAsins(ctx context.Context) ([]string, error) {
	if f.logger != nil {
		f.logger.Info("starting to fetch pet supplies ASINs from Keepa Product Finder API")
	}

	// 构建查询参数
	salesRankMin := 500
	salesRankMax := 10000
	currentAmazon := -1
	offerCountMin := 3
	offerCountMax := 20
	currentNewMin := 1500 // $15.00
	currentNewMax := 6000 // $60.00
	ratingMin := 38       // 3.8 星
	perPage := 100        // 获取 100 个 ASIN

	query := &model.ProductFinderQuery{
		RootCategory:  2619533011, // US Pet Supplies
		SalesRankGTE:  &salesRankMin,
		SalesRankLTE:  &salesRankMax,
		CurrentAmazon: &currentAmazon,
		OfferCountGTE: &offerCountMin,
		OfferCountLTE: &offerCountMax,
		CurrentNewGTE: &currentNewMin,
		CurrentNewLTE: &currentNewMax,
		RatingGTE:     &ratingMin,
		PerPage:       &perPage,
	}

	return f.FetchAsinsWithQuery(ctx, 1, query) // domain=1 表示 US
}

// FetchAsinsWithQuery 使用自定义查询参数获取 ASIN 列表
func (f *KeepaFetchAsins) FetchAsinsWithQuery(ctx context.Context, domain int, query *model.ProductFinderQuery) ([]string, error) {
	if f.logger != nil {
		f.logger.Info("starting to fetch ASINs with custom query",
			zap.Int("domain", domain),
		)
	}

	// 调用 API 获取数据（返回原始响应）
	asins, rawData, err := f.fetchFromProductFinder(ctx, domain, query)
	if err != nil {
		if f.logger != nil {
			f.logger.Error("failed to fetch ASINs from Product Finder API",
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to fetch ASINs: %w", err)
	}

	if f.logger != nil {
		f.logger.Info("successfully fetched ASINs from Product Finder API",
			zap.Int("asin_count", len(asins)),
		)
	}

	// 保存原始响应到 MongoDB
	if rawData != nil {
		if err := f.saveRawResponse(ctx, domain, query, rawData); err != nil {
			if f.logger != nil {
				f.logger.Warn("failed to save raw response to MongoDB",
					zap.Error(err),
				)
			}
		}
	}

	// 转换为 KeepaASIN 并存储到 MongoDB
	if err := f.saveAsinsToMongoDB(ctx, asins, domain, query); err != nil {
		if f.logger != nil {
			f.logger.Warn("failed to save ASINs to MongoDB, but returning results",
				zap.Error(err),
			)
		}
	} else {
		if f.logger != nil {
			f.logger.Info("successfully saved ASINs to MongoDB",
				zap.Int("asin_count", len(asins)),
			)
		}
	}

	return asins, nil
}

// fetchFromProductFinder 从 Product Finder API 获取数据，返回 ASIN 列表和原始响应
func (f *KeepaFetchAsins) fetchFromProductFinder(ctx context.Context, domain int, query *model.ProductFinderQuery) ([]string, []byte, error) {
	// 构建请求参数
	params := map[string]string{
		"domain": strconv.Itoa(domain),
	}

	// 构建查询 JSON
	queryMap := query.ToMap()
	queryJSON, err := json.Marshal(queryMap)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal query JSON: %w", err)
	}

	if f.logger != nil {
		f.logger.Debug("Product Finder query",
			zap.String("query_json", string(queryJSON)),
		)
	}

	// 使用 POST 方法调用 API
	rawData, err := f.client.PostRawDataWithParams(ctx, "/query", params, queryMap)
	if err != nil {
		return nil, nil, fmt.Errorf("API request failed: %w", err)
	}

	// 解析响应
	var response model.ProductFinderResponse
	if err := json.Unmarshal(rawData, &response); err != nil {
		return nil, rawData, fmt.Errorf("failed to parse API response: %w", err)
	}

	if f.logger != nil {
		f.logger.Info("Product Finder API response",
			zap.Int("total_results", response.TotalResults),
			zap.Int("asin_count", len(response.AsinList)),
			zap.Int("tokens_consumed", response.TokensConsumed),
			zap.Int("tokens_left", response.TokensLeft),
		)
	}

	return response.AsinList, rawData, nil
}

// saveRawResponse 保存原始 API 响应到 MongoDB
func (f *KeepaFetchAsins) saveRawResponse(ctx context.Context, domain int, query *model.ProductFinderQuery, rawData []byte) error {
	// 解析响应获取 token 信息
	var response model.ProductFinderResponse
	tokensUsed := 0
	tokensLeft := 0
	if err := json.Unmarshal(rawData, &response); err == nil {
		tokensUsed = response.TokensConsumed
		tokensLeft = response.TokensLeft
	}

	rawResponse := &model.KeepaRawResponse{
		APIEndpoint:  "/query",
		RequestType:  "product_finder",
		DomainID:     domain,
		RequestData:  query.ToMap(),
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
			zap.String("api_endpoint", "/query"),
			zap.Int("response_size", len(rawData)),
		)
	}

	return nil
}

// saveAsinsToMongoDB 将 ASIN 列表保存到 MongoDB（使用 Repository）
func (f *KeepaFetchAsins) saveAsinsToMongoDB(ctx context.Context, asins []string, domain int, query *model.ProductFinderQuery) error {
	if len(asins) == 0 {
		if f.logger != nil {
			f.logger.Warn("no ASINs to save")
		}
		return nil
	}

	// 转换为 KeepaASIN 模型
	keepaASINs := make([]*model.KeepaASIN, 0, len(asins))
	for _, asin := range asins {
		keepaASIN := &model.KeepaASIN{
			ASIN:        asin,
			DomainID:    domain,
			CategoryID:  query.RootCategory,
			QuerySource: "product_finder",
		}
		keepaASINs = append(keepaASINs, keepaASIN)
	}

	// 使用 Repository 批量保存
	if err := f.repo.SaveASINs(ctx, keepaASINs); err != nil {
		return fmt.Errorf("failed to save ASINs: %w", err)
	}

	return nil
}

// GetCollectionName 返回存储 ASIN 的 MongoDB 集合名称
func (f *KeepaFetchAsins) GetCollectionName() string {
	return model.CollectionKeepaASINs
}

// GetRawResponseCollectionName 返回存储原始响应的 MongoDB 集合名称
func (f *KeepaFetchAsins) GetRawResponseCollectionName() string {
	return model.CollectionKeepaRawResponses
}
