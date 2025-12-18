package browsing_deals

import (
	"context"
	"fmt"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Browsing Deals 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Browsing Deals 服务
func NewService(client *api.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// RequestParams 请求参数
type RequestParams struct {
	// 必需参数
	DomainID   int   `json:"domainId"`   // Amazon 域名代码 (1=com, 2=co.uk, 3=de, 4=fr, 5=co.jp, 6=ca, 8=it, 9=es, 10=in, 11=com.mx)
	PriceTypes []int `json:"priceTypes"` // 价格类型数组，必须只包含一个值
	DateRange  int   `json:"dateRange"`  // 时间范围: 0=Day, 1=Week, 2=Month, 3=3 Months

	// 可选参数 - 分页
	Page *int `json:"page,omitempty"` // 分页参数，从 0 开始

	// 可选参数 - 过滤选项
	IsFilterEnabled *bool `json:"isFilterEnabled,omitempty"` // 是否启用过滤选项

	// 类别过滤
	ExcludeCategories []int `json:"excludeCategories,omitempty"` // 排除的类别节点 ID 数组（最多 500 个）
	IncludeCategories []int `json:"includeCategories,omitempty"` // 包含的类别节点 ID 数组（最多 500 个）

	// 评分过滤
	MinRating *int `json:"minRating,omitempty"` // 最小评分 (0-50，-1 表示不启用)

	// 价格过滤
	IsLowest      *bool `json:"isLowest,omitempty"`      // 是否是最低价格（自跟踪开始）
	IsLowest90    *bool `json:"isLowest90,omitempty"`    // 是否是最低价格（过去 90 天）
	IsLowestOffer *bool `json:"isLowestOffer,omitempty"` // 是否是最低报价（New 类型）
	IsHighest     *bool `json:"isHighest,omitempty"`     // 是否是最高价格（自跟踪开始）

	// 库存过滤
	IsOutOfStock  *bool `json:"isOutOfStock,omitempty"`  // 是否缺货（过去 24 小时）
	IsBackInStock *bool `json:"isBackInStock,omitempty"` // 是否重新入库（过去 24 小时）

	// 其他过滤
	HasReviews             *bool `json:"hasReviews,omitempty"`             // 是否有评论 (true=排除无评论产品)
	FilterErotic           *bool `json:"filterErotic,omitempty"`           // 是否排除成人产品
	SingleVariation        *bool `json:"singleVariation,omitempty"`        // 是否只返回单个变体
	IsRisers               *bool `json:"isRisers,omitempty"`               // 是否价格上涨
	IsPrimeExclusive       *bool `json:"isPrimeExclusive,omitempty"`       // 是否 Prime Exclusive
	MustHaveAmazonOffer    *bool `json:"mustHaveAmazonOffer,omitempty"`    // 是否必须有 Amazon 报价
	MustNotHaveAmazonOffer *bool `json:"mustNotHaveAmazonOffer,omitempty"` // 是否必须没有 Amazon 报价

	// Warehouse 条件
	WarehouseConditions []int `json:"warehouseConditions,omitempty"` // Amazon Warehouse 条件代码数组

	// 产品属性过滤
	Material                []string `json:"material,omitempty"`                // 材质（如 "cotton"）
	Type                    []string `json:"type,omitempty"`                    // 类型（如 "shirt", "dress"）
	Manufacturer            []string `json:"manufacturer,omitempty"`            // 制造商
	Brand                   []string `json:"brand,omitempty"`                   // 品牌
	ProductGroup            []string `json:"productGroup,omitempty"`            // 产品组（如 "home", "book"）
	Model                   []string `json:"model,omitempty"`                   // 型号
	Color                   []string `json:"color,omitempty"`                   // 颜色
	Size                    []string `json:"size,omitempty"`                    // 尺寸（如 "small", "one size"）
	UnitType                []string `json:"unitType,omitempty"`                // 单位类型（如 "count", "ounce"）
	Scent                   []string `json:"scent,omitempty"`                   // 香味（如 "lavender", "citrus"）
	ItemForm                []string `json:"itemForm,omitempty"`                // 物品形式（如 "liquid", "sheet"）
	Pattern                 []string `json:"pattern,omitempty"`                 // 图案（如 "striped", "solid"）
	Style                   []string `json:"style,omitempty"`                   // 风格（如 "modern", "vintage"）
	ItemTypeKeyword         []string `json:"itemTypeKeyword,omitempty"`         // 物品类型关键词（如 "books", "prints"）
	TargetAudienceKeyword   []string `json:"targetAudienceKeyword,omitempty"`   // 目标受众关键词（如 "kids", "professional"）
	Edition                 []string `json:"edition,omitempty"`                 // 版本（如 "first edition", "standard edition"）
	Format                  []string `json:"format,omitempty"`                  // 格式（如 "kindle ebook", "import", "dvd"）
	Author                  []string `json:"author,omitempty"`                  // 作者（适用于书籍、音乐等）
	Binding                 []string `json:"binding,omitempty"`                 // 装订类型（如 "paperback"）
	Languages               []string `json:"languages,omitempty"`               // 语言数组
	BrandStoreName          []string `json:"brandStoreName,omitempty"`          // 品牌商店名称
	BrandStoreUrlName       []string `json:"brandStoreUrlName,omitempty"`       // 品牌商店 URL 名称
	WebsiteDisplayGroup     []string `json:"websiteDisplayGroup,omitempty"`     // 网站显示组
	WebsiteDisplayGroupName []string `json:"websiteDisplayGroupName,omitempty"` // 网站显示组名称
	SalesRankDisplayGroup   []string `json:"salesRankDisplayGroup,omitempty"`   // 销售排名显示组（如 "fashion_display_on_website"）
}

// Validate 验证请求参数
func (p *RequestParams) Validate() error {
	// 验证 domainId (有效值: 1,2,3,4,5,6,8,9,10,11)
	validDomains := map[int]bool{
		1:  true, // com
		2:  true, // co.uk
		3:  true, // de
		4:  true, // fr
		5:  true, // co.jp
		6:  true, // ca
		8:  true, // it
		9:  true, // es
		10: true, // in
		11: true, // com.mx
	}
	if !validDomains[p.DomainID] {
		return fmt.Errorf("invalid domainId: %d (valid domains: 1,2,3,4,5,6,8,9,10,11)", p.DomainID)
	}

	// 验证 priceTypes 必须存在且只包含一个值
	if len(p.PriceTypes) == 0 {
		return fmt.Errorf("priceTypes is required and must contain exactly one value")
	}
	if len(p.PriceTypes) > 1 {
		return fmt.Errorf("priceTypes must contain exactly one value, got %d", len(p.PriceTypes))
	}

	// 验证 priceTypes 的值
	validPriceTypes := map[int]bool{
		0:  true, // AMAZON
		1:  true, // NEW
		2:  true, // USED
		3:  true, // SALES
		5:  true, // COLLECTIBLE
		6:  true, // REFURBISHED
		7:  true, // NEW_FBM_SHIPPING
		8:  true, // LIGHTNING_DEAL
		9:  true, // WAREHOUSE
		10: true, // NEW_FBA
		18: true, // BUY_BOX_SHIPPING
		19: true, // USED_NEW_SHIPPING
		20: true, // USED_VERY_GOOD_SHIPPING
		21: true, // USED_GOOD_SHIPPING
		22: true, // USED_ACCEPTABLE_SHIPPING
		32: true, // BUY_BOX_USED_SHIPPING
		33: true, // PRIME_EXCL
	}
	if !validPriceTypes[p.PriceTypes[0]] {
		return fmt.Errorf("invalid priceType: %d", p.PriceTypes[0])
	}

	// 验证 dateRange
	validDateRanges := map[int]bool{
		0: true, // Day (last 24 hours)
		1: true, // Week (last 7 days)
		2: true, // Month (last 31 days)
		3: true, // 3 Months (last 90 days)
	}
	if !validDateRanges[p.DateRange] {
		return fmt.Errorf("invalid dateRange: %d (valid values: 0, 1, 2, 3)", p.DateRange)
	}

	// 验证 page（如果提供）
	if p.Page != nil && *p.Page < 0 {
		return fmt.Errorf("page must be >= 0, got %d", *p.Page)
	}

	// 验证 minRating（如果提供）
	if p.MinRating != nil {
		if *p.MinRating < -1 || *p.MinRating > 50 {
			return fmt.Errorf("minRating must be between -1 and 50, got %d", *p.MinRating)
		}
	}

	// 验证 excludeCategories（最多 500 个）
	if len(p.ExcludeCategories) > 500 {
		return fmt.Errorf("excludeCategories can contain at most 500 category node IDs, got %d", len(p.ExcludeCategories))
	}

	// 验证 includeCategories（最多 500 个）
	if len(p.IncludeCategories) > 500 {
		return fmt.Errorf("includeCategories can contain at most 500 category node IDs, got %d", len(p.IncludeCategories))
	}

	return nil
}

// ToQueryJSON 将 RequestParams 转换为 queryJSON 格式（map[string]interface{}）
func (p *RequestParams) ToQueryJSON() map[string]interface{} {
	queryJSON := make(map[string]interface{})

	// 必需参数
	queryJSON["domainId"] = p.DomainID
	queryJSON["priceTypes"] = p.PriceTypes
	queryJSON["dateRange"] = p.DateRange

	// 可选参数 - 分页
	if p.Page != nil {
		queryJSON["page"] = *p.Page
	}

	// 可选参数 - 过滤选项
	if p.IsFilterEnabled != nil {
		queryJSON["isFilterEnabled"] = *p.IsFilterEnabled
	}

	// 类别过滤
	if len(p.ExcludeCategories) > 0 {
		queryJSON["excludeCategories"] = p.ExcludeCategories
	}
	if len(p.IncludeCategories) > 0 {
		queryJSON["includeCategories"] = p.IncludeCategories
	}

	// 评分过滤
	if p.MinRating != nil {
		queryJSON["minRating"] = *p.MinRating
	}

	// 价格过滤
	if p.IsLowest != nil {
		queryJSON["isLowest"] = *p.IsLowest
	}
	if p.IsLowest90 != nil {
		queryJSON["isLowest90"] = *p.IsLowest90
	}
	if p.IsLowestOffer != nil {
		queryJSON["isLowestOffer"] = *p.IsLowestOffer
	}
	if p.IsHighest != nil {
		queryJSON["isHighest"] = *p.IsHighest
	}

	// 库存过滤
	if p.IsOutOfStock != nil {
		queryJSON["isOutOfStock"] = *p.IsOutOfStock
	}
	if p.IsBackInStock != nil {
		queryJSON["isBackInStock"] = *p.IsBackInStock
	}

	// 其他过滤
	if p.HasReviews != nil {
		queryJSON["hasReviews"] = *p.HasReviews
	}
	if p.FilterErotic != nil {
		queryJSON["filterErotic"] = *p.FilterErotic
	}
	if p.SingleVariation != nil {
		queryJSON["singleVariation"] = *p.SingleVariation
	}
	if p.IsRisers != nil {
		queryJSON["isRisers"] = *p.IsRisers
	}
	if p.IsPrimeExclusive != nil {
		queryJSON["isPrimeExclusive"] = *p.IsPrimeExclusive
	}
	if p.MustHaveAmazonOffer != nil {
		queryJSON["mustHaveAmazonOffer"] = *p.MustHaveAmazonOffer
	}
	if p.MustNotHaveAmazonOffer != nil {
		queryJSON["mustNotHaveAmazonOffer"] = *p.MustNotHaveAmazonOffer
	}

	// Warehouse 条件
	if len(p.WarehouseConditions) > 0 {
		queryJSON["warehouseConditions"] = p.WarehouseConditions
	}

	// 产品属性过滤
	if len(p.Material) > 0 {
		queryJSON["material"] = p.Material
	}
	if len(p.Type) > 0 {
		queryJSON["type"] = p.Type
	}
	if len(p.Manufacturer) > 0 {
		queryJSON["manufacturer"] = p.Manufacturer
	}
	if len(p.Brand) > 0 {
		queryJSON["brand"] = p.Brand
	}
	if len(p.ProductGroup) > 0 {
		queryJSON["productGroup"] = p.ProductGroup
	}
	if len(p.Model) > 0 {
		queryJSON["model"] = p.Model
	}
	if len(p.Color) > 0 {
		queryJSON["color"] = p.Color
	}
	if len(p.Size) > 0 {
		queryJSON["size"] = p.Size
	}
	if len(p.UnitType) > 0 {
		queryJSON["unitType"] = p.UnitType
	}
	if len(p.Scent) > 0 {
		queryJSON["scent"] = p.Scent
	}
	if len(p.ItemForm) > 0 {
		queryJSON["itemForm"] = p.ItemForm
	}
	if len(p.Pattern) > 0 {
		queryJSON["pattern"] = p.Pattern
	}
	if len(p.Style) > 0 {
		queryJSON["style"] = p.Style
	}
	if len(p.ItemTypeKeyword) > 0 {
		queryJSON["itemTypeKeyword"] = p.ItemTypeKeyword
	}
	if len(p.TargetAudienceKeyword) > 0 {
		queryJSON["targetAudienceKeyword"] = p.TargetAudienceKeyword
	}
	if len(p.Edition) > 0 {
		queryJSON["edition"] = p.Edition
	}
	if len(p.Format) > 0 {
		queryJSON["format"] = p.Format
	}
	if len(p.Author) > 0 {
		queryJSON["author"] = p.Author
	}
	if len(p.Binding) > 0 {
		queryJSON["binding"] = p.Binding
	}
	if len(p.Languages) > 0 {
		queryJSON["languages"] = p.Languages
	}
	if len(p.BrandStoreName) > 0 {
		queryJSON["brandStoreName"] = p.BrandStoreName
	}
	if len(p.BrandStoreUrlName) > 0 {
		queryJSON["brandStoreUrlName"] = p.BrandStoreUrlName
	}
	if len(p.WebsiteDisplayGroup) > 0 {
		queryJSON["websiteDisplayGroup"] = p.WebsiteDisplayGroup
	}
	if len(p.WebsiteDisplayGroupName) > 0 {
		queryJSON["websiteDisplayGroupName"] = p.WebsiteDisplayGroupName
	}
	if len(p.SalesRankDisplayGroup) > 0 {
		queryJSON["salesRankDisplayGroup"] = p.SalesRankDisplayGroup
	}

	return queryJSON
}

// Fetch 获取数据
func (s *Service) Fetch(ctx context.Context, params RequestParams) ([]byte, error) {
	// 验证参数
	if err := params.Validate(); err != nil {
		if s.logger != nil {
			s.logger.Error("invalid request parameters", zap.Error(err))
		}
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 记录请求日志
	if s.logger != nil {
		logFields := []zap.Field{
			zap.Int("domainId", params.DomainID),
			zap.Ints("priceTypes", params.PriceTypes),
			zap.Int("dateRange", params.DateRange),
		}
		if params.Page != nil {
			logFields = append(logFields, zap.Int("page", *params.Page))
		}

		s.logger.Info("fetching browsing deals data", logFields...)
	}

	// 构建 API 请求
	endpoint := "/deal"
	queryJSON := params.ToQueryJSON()

	// 调用 API 获取原始数据（使用 POST 方法）
	rawData, err := s.client.PostRawData(ctx, endpoint, queryJSON)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch browsing deals data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("browsing deals data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
