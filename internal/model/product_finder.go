package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson"
)

// ProductFinderQuery Product Finder API 查询参数
type ProductFinderQuery struct {
	// 分类相关
	RootCategory int64 `json:"rootCategory,omitempty"` // 根分类 ID

	// 销量排名范围
	SalesRankGTE *int `json:"salesRank_gte,omitempty"` // 最小销量排名
	SalesRankLTE *int `json:"salesRank_lte,omitempty"` // 最大销量排名

	// 亚马逊自营价格 (-1 表示排除亚马逊自营)
	CurrentAmazon *int `json:"current_AMAZON,omitempty"`

	// Offer 数量范围
	OfferCountGTE *int `json:"offerCount_gte,omitempty"` // 最小 offer 数量
	OfferCountLTE *int `json:"offerCount_lte,omitempty"` // 最大 offer 数量

	// 新品价格范围（单位：分）
	CurrentNewGTE *int `json:"current_NEW_gte,omitempty"` // 最低新品价格
	CurrentNewLTE *int `json:"current_NEW_lte,omitempty"` // 最高新品价格

	// 评分范围（38 = 3.8 星）
	RatingGTE *int `json:"rating_gte,omitempty"` // 最低评分

	// 分页参数
	Page    *int `json:"page,omitempty"`    // 页码，从 0 开始
	PerPage *int `json:"perPage,omitempty"` // 每页数量，最小 50
}

// ToMap 将查询参数转换为 map[string]interface{}
func (q *ProductFinderQuery) ToMap() map[string]interface{} {
	result := make(map[string]interface{})

	if q.RootCategory != 0 {
		result["rootCategory"] = q.RootCategory
	}

	if q.SalesRankGTE != nil {
		result["salesRank_gte"] = *q.SalesRankGTE
	}
	if q.SalesRankLTE != nil {
		result["salesRank_lte"] = *q.SalesRankLTE
	}

	if q.CurrentAmazon != nil {
		result["current_AMAZON"] = *q.CurrentAmazon
	}

	if q.OfferCountGTE != nil {
		result["offerCount_gte"] = *q.OfferCountGTE
	}
	if q.OfferCountLTE != nil {
		result["offerCount_lte"] = *q.OfferCountLTE
	}

	if q.CurrentNewGTE != nil {
		result["current_NEW_gte"] = *q.CurrentNewGTE
	}
	if q.CurrentNewLTE != nil {
		result["current_NEW_lte"] = *q.CurrentNewLTE
	}

	if q.RatingGTE != nil {
		result["rating_gte"] = *q.RatingGTE
	}

	if q.Page != nil {
		result["page"] = *q.Page
	}
	if q.PerPage != nil {
		result["perPage"] = *q.PerPage
	}

	return result
}

// ProductFinderResponse Product Finder API 响应
type ProductFinderResponse struct {
	Timestamp          int64    `json:"timestamp"`
	TokensLeft         int      `json:"tokensLeft"`
	RefillIn           int      `json:"refillIn"`
	RefillRate         int      `json:"refillRate"`
	TokenFlowReduction float64  `json:"tokenFlowReduction"`
	TokensConsumed     int      `json:"tokensConsumed"`
	ProcessingTimeInMs int      `json:"processingTimeInMs"`
	AsinList           []string `json:"asinList"` // ASIN 列表
	TotalResults       int      `json:"totalResults"`
}

// ASINDocument MongoDB 中存储的 ASIN 文档
type ASINDocument struct {
	ASIN        string    `bson:"asin"`
	Domain      int       `bson:"domain"`
	Category    int64     `bson:"category"`
	QueryParams bson.M    `bson:"query_params"`
	FetchedAt   time.Time `bson:"fetched_at"`
	CreatedAt   time.Time `bson:"created_at"`
	UpdatedAt   time.Time `bson:"updated_at"`
}
