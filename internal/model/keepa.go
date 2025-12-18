package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// KeepaCategory Keepa 分类 MongoDB 存储模型
type KeepaCategory struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	DomainID  int                `bson:"domain_id"` // Amazon 域名 ID
	CatID     int64              `bson:"cat_id"`    // 分类 ID
	Name      string             `bson:"name"`      // 分类名称
	Parent    int64              `bson:"parent"`    // 父分类 ID
	Children  []int64            `bson:"children,omitempty"`
	CreatedAt time.Time          `bson:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at"`

	// 完整分类数据（JSON 原始数据）
	RawData *Category `bson:"raw_data,omitempty"`
}

// KeepaASIN Keepa ASIN MongoDB 存储模型
type KeepaASIN struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	ASIN          string             `bson:"asin"`                   // ASIN 码
	DomainID      int                `bson:"domain_id"`              // Amazon 域名 ID
	CategoryID    int64              `bson:"category_id,omitempty"`  // 分类 ID
	DetailFetched bool               `bson:"detail_fetched"`         // 是否已获取详情
	FetchedAt     *time.Time         `bson:"fetched_at,omitempty"`   // 详情获取时间
	QuerySource   string             `bson:"query_source,omitempty"` // 来源查询（如 product_finder）
	CreatedAt     time.Time          `bson:"created_at"`
	UpdatedAt     time.Time          `bson:"updated_at"`
}

// KeepaProduct Keepa 产品 MongoDB 存储模型
type KeepaProduct struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	ASIN     string             `bson:"asin"`      // ASIN 码
	DomainID int                `bson:"domain_id"` // Amazon 域名 ID

	// 基础信息
	Title        string  `bson:"title,omitempty"`
	Brand        string  `bson:"brand,omitempty"`
	Manufacturer string  `bson:"manufacturer,omitempty"`
	ProductGroup string  `bson:"product_group,omitempty"`
	RootCategory int64   `bson:"root_category,omitempty"`
	Categories   []int64 `bson:"categories,omitempty"`

	// 价格信息（单位：分）
	CurrentPrice    int `bson:"current_price,omitempty"`     // 当前价格
	BuyBoxPrice     int `bson:"buybox_price,omitempty"`      // BuyBox 价格
	AmazonPrice     int `bson:"amazon_price,omitempty"`      // 亚马逊价格
	NewPrice        int `bson:"new_price,omitempty"`         // 新品最低价
	UsedPrice       int `bson:"used_price,omitempty"`        // 二手最低价
	BuyBoxShipping  int `bson:"buybox_shipping,omitempty"`   // BuyBox 运费
	SalesRank       int `bson:"sales_rank,omitempty"`        // 销售排名
	OfferCountNew   int `bson:"offer_count_new,omitempty"`   // 新品 offer 数量
	OfferCountUsed  int `bson:"offer_count_used,omitempty"`  // 二手 offer 数量
	TotalOfferCount int `bson:"total_offer_count,omitempty"` // 总 offer 数量

	// 评价信息
	Rating      int `bson:"rating,omitempty"`       // 评分（10-50 分制）
	ReviewCount int `bson:"review_count,omitempty"` // 评价数量

	// 尺寸和重量
	PackageHeight int `bson:"package_height,omitempty"`
	PackageLength int `bson:"package_length,omitempty"`
	PackageWidth  int `bson:"package_width,omitempty"`
	PackageWeight int `bson:"package_weight,omitempty"`
	ItemHeight    int `bson:"item_height,omitempty"`
	ItemLength    int `bson:"item_length,omitempty"`
	ItemWidth     int `bson:"item_width,omitempty"`
	ItemWeight    int `bson:"item_weight,omitempty"`

	// 状态标识
	IsFBA          bool `bson:"is_fba,omitempty"`
	IsAmazon       bool `bson:"is_amazon,omitempty"`
	IsPrime        bool `bson:"is_prime,omitempty"`
	IsAdultProduct bool `bson:"is_adult_product,omitempty"`

	// 历史数据
	CSV   [][]int       `bson:"csv,omitempty"`   // 价格历史
	Stats *ProductStats `bson:"stats,omitempty"` // 统计数据

	// 图片
	ImagesCSV string `bson:"images_csv,omitempty"`

	// 完整产品数据（JSON 原始数据）
	RawData *ProductDetails `bson:"raw_data,omitempty"`

	// 时间戳
	LastUpdate int64     `bson:"last_update,omitempty"` // Keepa 最后更新时间
	FetchedAt  time.Time `bson:"fetched_at"`            // 本地获取时间
	CreatedAt  time.Time `bson:"created_at"`
	UpdatedAt  time.Time `bson:"updated_at"`
}

// CollectionNames 定义 MongoDB 集合名称
const (
	CollectionKeepaCategories   = "keepa_categories"
	CollectionKeepaASINs        = "keepa_asins"
	CollectionKeepaProducts     = "keepa_products"
	CollectionKeepaRawResponses = "keepa_raw_responses"
)

// KeepaRawResponse 存储 Keepa API 原始响应
type KeepaRawResponse struct {
	ID           primitive.ObjectID `bson:"_id,omitempty"`
	APIEndpoint  string             `bson:"api_endpoint"`           // API 端点（如 /query, /product）
	RequestType  string             `bson:"request_type"`           // 请求类型（如 product_finder, product）
	DomainID     int                `bson:"domain_id"`              // Amazon 域名 ID
	RequestData  interface{}        `bson:"request_data,omitempty"` // 请求参数
	RawResponse  string             `bson:"raw_response"`           // 原始 JSON 响应字符串
	ResponseSize int                `bson:"response_size"`          // 响应大小（字节）
	StatusCode   int                `bson:"status_code,omitempty"`  // HTTP 状态码
	TokensUsed   int                `bson:"tokens_used,omitempty"`  // 消耗的 token 数量
	TokensLeft   int                `bson:"tokens_left,omitempty"`  // 剩余 token 数量
	CreatedAt    time.Time          `bson:"created_at"`             // 创建时间
}
