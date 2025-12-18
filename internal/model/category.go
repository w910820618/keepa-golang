package model

// Category Keepa API 返回的分类对象
// 参考: https://keepa.com/#!discuss/topic/category-object
type Category struct {
	// 基础信息
	DomainID            int    `json:"domainId" bson:"domain_id"`                                            // Amazon 域名 ID (1=.com, 2=.co.uk, 3=.de, 4=.fr, 5=.co.jp, 6=.ca, 8=.it, 9=.es, 10=.in, 11=.com.mx, 12=.com.br)
	CatID               int64  `json:"catId" bson:"cat_id"`                                                  // 分类节点 ID（Amazon 使用的标识符）
	Name                string `json:"name" bson:"name"`                                                     // 分类名称
	ContextFreeName     string `json:"contextFreeName,omitempty" bson:"context_free_name,omitempty"`         // 上下文无关的分类名称
	WebsiteDisplayGroup string `json:"websiteDisplayGroup,omitempty" bson:"website_display_group,omitempty"` // 网站显示组

	// 层级关系
	Children []int64 `json:"children,omitempty" bson:"children,omitempty"` // 所有子分类的 ID 列表，如果为空则为 null 或 []
	Parent   int64   `json:"parent" bson:"parent"`                         // 父分类的 ID，如果为 0 则表示是根分类

	// 分类属性
	IsBrowseNode bool `json:"isBrowseNode" bson:"is_browse_node"` // 是否为标准浏览节点（而非促销用途）

	// 排名和产品统计
	HighestRank  int `json:"highestRank,omitempty" bson:"highest_rank,omitempty"`   // 该分类中产品的最高（根分类）销售排名
	LowestRank   int `json:"lowestRank,omitempty" bson:"lowest_rank,omitempty"`     // 该分类中产品的最低（根分类）销售排名
	ProductCount int `json:"productCount,omitempty" bson:"product_count,omitempty"` // 该分类中估计的产品数量

	// 价格统计（单位：最小货币单位，如美分）
	AvgBuyBox          int `json:"avgBuyBox,omitempty" bson:"avg_buy_box,omitempty"`                    // 当前平均 Buy Box 价格
	AvgBuyBox90        int `json:"avgBuyBox90,omitempty" bson:"avg_buy_box_90,omitempty"`               // 90 天平均 Buy Box 价格
	AvgBuyBox365       int `json:"avgBuyBox365,omitempty" bson:"avg_buy_box_365,omitempty"`             // 365 天平均 Buy Box 价格
	AvgBuyBoxDeviation int `json:"avgBuyBoxDeviation,omitempty" bson:"avg_buy_box_deviation,omitempty"` // 30 天平均 Buy Box 价格偏差

	// 评价统计
	AvgReviewCount int `json:"avgReviewCount,omitempty" bson:"avg_review_count,omitempty"` // 平均评价数量
	AvgRating      int `json:"avgRating,omitempty" bson:"avg_rating,omitempty"`            // 平均评分（10-50 分制，例如 45 表示 4.5 星）

	// 百分比统计
	IsFBAPercent        float64 `json:"isFBAPercent,omitempty" bson:"is_fba_percent,omitempty"`                // FBA 配送的产品百分比
	SoldByAmazonPercent float64 `json:"soldByAmazonPercent,omitempty" bson:"sold_by_amazon_percent,omitempty"` // 由 Amazon 销售的产品百分比
	HasCouponPercent    float64 `json:"hasCouponPercent,omitempty" bson:"has_coupon_percent,omitempty"`        // 有优惠券的产品百分比

	// 报价统计
	AvgOfferCountNew  float64 `json:"avgOfferCountNew,omitempty" bson:"avg_offer_count_new,omitempty"`   // 平均新品报价数量（排除缺货）
	AvgOfferCountUsed float64 `json:"avgOfferCountUsed,omitempty" bson:"avg_offer_count_used,omitempty"` // 平均二手品报价数量（排除缺货）

	// 卖家统计
	SellerCount int `json:"sellerCount,omitempty" bson:"seller_count,omitempty"` // 该分类中至少有 1 个活跃报价的不同卖家总数
	BrandCount  int `json:"brandCount,omitempty" bson:"brand_count,omitempty"`   // 该分类中产品的不同品牌总数

	// 价格变化百分比
	AvgDeltaPercent30BuyBox float64 `json:"avgDeltaPercent30BuyBox,omitempty" bson:"avg_delta_percent_30_buy_box,omitempty"` // 过去 30 天 Buy Box 价格的平均百分比变化（正数表示更便宜）
	AvgDeltaPercent90BuyBox float64 `json:"avgDeltaPercent90BuyBox,omitempty" bson:"avg_delta_percent_90_buy_box,omitempty"` // 过去 90 天 Buy Box 价格的平均百分比变化
	AvgDeltaPercent30Amazon float64 `json:"avgDeltaPercent30Amazon,omitempty" bson:"avg_delta_percent_30_amazon,omitempty"`  // 过去 30 天 Amazon 报价价格的平均百分比变化
	AvgDeltaPercent90Amazon float64 `json:"avgDeltaPercent90Amazon,omitempty" bson:"avg_delta_percent_90_amazon,omitempty"`  // 过去 90 天 Amazon 报价价格的平均百分比变化

	// 关联信息
	RelatedCategories     []int64  `json:"relatedCategories,omitempty" bson:"related_categories,omitempty"`           // 相关分类 ID 列表（按共同列出频率排序）
	TopBrands             []string `json:"topBrands,omitempty" bson:"top_brands,omitempty"`                           // 最多前 3 个最常见的品牌（按出现频率降序）
	RelatedSellerNames    []string `json:"relatedSellerNames,omitempty" bson:"related_seller_names,omitempty"`        // 相关卖家名称列表
	RelatedSellerNamesAny []string `json:"relatedSellerNamesAny,omitempty" bson:"related_seller_names_any,omitempty"` // 相关卖家名称列表（任意条件）
	TopSellers            []string `json:"topSellers,omitempty" bson:"top_sellers,omitempty"`                         // 顶级卖家 ID 列表
	TopSellersAny         []string `json:"topSellersAny,omitempty" bson:"top_sellers_any,omitempty"`                  // 顶级卖家 ID 列表（任意条件）
}

// CategoryTree 分类树节点
type CategoryTree struct {
	Category  *Category       `json:"category" bson:"category"`                     // 分类信息
	Children  []*CategoryTree `json:"children,omitempty" bson:"children,omitempty"` // 子分类树
	CreatedAt int64           `json:"created_at" bson:"created_at"`                 // 创建时间戳（Unix 时间戳）
}

// IsRoot 判断是否为根分类
func (c *Category) IsRoot() bool {
	return c.Parent == 0
}

// HasChildren 判断是否有子分类
func (c *Category) HasChildren() bool {
	return len(c.Children) > 0
}
