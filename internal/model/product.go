package model

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// ProductAPIResponse Keepa Product API 响应
type ProductAPIResponse struct {
	Timestamp          int64            `json:"timestamp"`
	TokensLeft         int              `json:"tokensLeft"`
	RefillIn           int              `json:"refillIn"`
	RefillRate         int              `json:"refillRate"`
	TokenFlowReduction float64          `json:"tokenFlowReduction"`
	TokensConsumed     int              `json:"tokensConsumed"`
	ProcessingTimeInMs int              `json:"processingTimeInMs"`
	Products           []ProductDetails `json:"products"`
}

// ProductDetails 产品详情
type ProductDetails struct {
	ASIN                            string                 `json:"asin" bson:"asin"`
	DomainID                        int                    `json:"domainId" bson:"domain_id"`
	Title                           string                 `json:"title" bson:"title"`
	Type                            string                 `json:"type,omitempty" bson:"type,omitempty"`
	RootCategory                    int64                  `json:"rootCategory" bson:"root_category"`
	Categories                      []int64                `json:"categories,omitempty" bson:"categories,omitempty"`
	Brand                           string                 `json:"brand,omitempty" bson:"brand,omitempty"`
	Manufacturer                    string                 `json:"manufacturer,omitempty" bson:"manufacturer,omitempty"`
	ProductGroup                    string                 `json:"productGroup,omitempty" bson:"product_group,omitempty"`
	PartNumber                      string                 `json:"partNumber,omitempty" bson:"part_number,omitempty"`
	Model                           string                 `json:"model,omitempty" bson:"model,omitempty"`
	Color                           string                 `json:"color,omitempty" bson:"color,omitempty"`
	Size                            string                 `json:"size,omitempty" bson:"size,omitempty"`
	PackageHeight                   int                    `json:"packageHeight,omitempty" bson:"package_height,omitempty"`
	PackageLength                   int                    `json:"packageLength,omitempty" bson:"package_length,omitempty"`
	PackageWidth                    int                    `json:"packageWidth,omitempty" bson:"package_width,omitempty"`
	PackageWeight                   int                    `json:"packageWeight,omitempty" bson:"package_weight,omitempty"`
	PackageQuantity                 int                    `json:"packageQuantity,omitempty" bson:"package_quantity,omitempty"`
	ItemHeight                      int                    `json:"itemHeight,omitempty" bson:"item_height,omitempty"`
	ItemLength                      int                    `json:"itemLength,omitempty" bson:"item_length,omitempty"`
	ItemWidth                       int                    `json:"itemWidth,omitempty" bson:"item_width,omitempty"`
	ItemWeight                      int                    `json:"itemWeight,omitempty" bson:"item_weight,omitempty"`
	NumberOfItems                   int                    `json:"numberOfItems,omitempty" bson:"number_of_items,omitempty"`
	NumberOfPages                   int                    `json:"numberOfPages,omitempty" bson:"number_of_pages,omitempty"`
	Description                     string                 `json:"description,omitempty" bson:"description,omitempty"`
	Features                        []string               `json:"features,omitempty" bson:"features,omitempty"`
	HasReviews                      bool                   `json:"hasReviews,omitempty" bson:"has_reviews,omitempty"`
	HasAplus                        *bool                  `json:"hasAplus,omitempty" bson:"has_aplus,omitempty"`
	IsEligibleForSuperSaverShipping bool                   `json:"isEligibleForSuperSaverShipping,omitempty" bson:"is_eligible_for_super_saver_shipping,omitempty"`
	IsEligibleForPrime              bool                   `json:"isEligibleForPrime,omitempty" bson:"is_eligible_for_prime,omitempty"`
	IsAdultProduct                  bool                   `json:"isAdultProduct,omitempty" bson:"is_adult_product,omitempty"`
	NewPriceIsMAP                   bool                   `json:"newPriceIsMAP,omitempty" bson:"new_price_is_map,omitempty"`
	FBAFees                         map[string]interface{} `json:"fbaFees,omitempty" bson:"fba_fees,omitempty"`
	VariationCSV                    []interface{}          `json:"variationCSV,omitempty" bson:"variation_csv,omitempty"`
	FrequentlyBoughtTogether        []string               `json:"frequentlyBoughtTogether,omitempty" bson:"frequently_bought_together,omitempty"`
	// CSV 数组 - 价格历史数据
	CSV [][]int `json:"csv,omitempty" bson:"csv,omitempty"`
	// Stats 对象 - 统计数据
	Stats *ProductStats `json:"stats,omitempty" bson:"stats,omitempty"`
	// BuyBox 数据
	BuyBoxSellerIdHistory []interface{} `json:"buyBoxSellerIdHistory,omitempty" bson:"buybox_seller_id_history,omitempty"`
	// 其他字段
	LastUpdate       int64 `json:"lastUpdate,omitempty" bson:"last_update,omitempty"`
	LastPriceChange  int64 `json:"lastPriceChange,omitempty" bson:"last_price_change,omitempty"`
	LastRatingUpdate int64 `json:"lastRatingUpdate,omitempty" bson:"last_rating_update,omitempty"`
	// 图片 URL
	ImagesCSV string `json:"imagesCSV,omitempty" bson:"images_csv,omitempty"`
}

// ProductStats 产品统计数据
type ProductStats struct {
	Current                        []int    `json:"current,omitempty" bson:"current,omitempty"`
	Avg                            []int    `json:"avg,omitempty" bson:"avg,omitempty"`
	Avg30                          []int    `json:"avg30,omitempty" bson:"avg30,omitempty"`
	Avg90                          []int    `json:"avg90,omitempty" bson:"avg90,omitempty"`
	Avg180                         []int    `json:"avg180,omitempty" bson:"avg180,omitempty"`
	AtIntervalStart                []int    `json:"atIntervalStart,omitempty" bson:"at_interval_start,omitempty"`
	Min                            [][]int  `json:"min,omitempty" bson:"min,omitempty"`
	Max                            [][]int  `json:"max,omitempty" bson:"max,omitempty"`
	MinInInterval                  [][]int  `json:"minInInterval,omitempty" bson:"min_in_interval,omitempty"`
	MaxInInterval                  [][]int  `json:"maxInInterval,omitempty" bson:"max_in_interval,omitempty"`
	OutOfStockPercentageInInterval []int    `json:"outOfStockPercentageInInterval,omitempty" bson:"out_of_stock_percentage_in_interval,omitempty"`
	OutOfStockPercentage30         []int    `json:"outOfStockPercentage30,omitempty" bson:"out_of_stock_percentage30,omitempty"`
	OutOfStockPercentage90         []int    `json:"outOfStockPercentage90,omitempty" bson:"out_of_stock_percentage90,omitempty"`
	LastOffersUpdate               int64    `json:"lastOffersUpdate,omitempty" bson:"last_offers_update,omitempty"`
	TotalOfferCount                int      `json:"totalOfferCount,omitempty" bson:"total_offer_count,omitempty"`
	LightningDealInfo              []int    `json:"lightningDealInfo,omitempty" bson:"lightning_deal_info,omitempty"`
	SellerId                       string   `json:"sellerId,omitempty" bson:"seller_id,omitempty"`
	BuyBoxPrice                    int      `json:"buyBoxPrice,omitempty" bson:"buybox_price,omitempty"`
	BuyBoxShipping                 int      `json:"buyBoxShipping,omitempty" bson:"buybox_shipping,omitempty"`
	BuyBoxIsUnqualified            bool     `json:"buyBoxIsUnqualified,omitempty" bson:"buybox_is_unqualified,omitempty"`
	BuyBoxIsShippable              bool     `json:"buyBoxIsShippable,omitempty" bson:"buybox_is_shippable,omitempty"`
	BuyBoxIsPreorder               bool     `json:"buyBoxIsPreorder,omitempty" bson:"buybox_is_preorder,omitempty"`
	BuyBoxIsFBA                    bool     `json:"buyBoxIsFBA,omitempty" bson:"buybox_is_fba,omitempty"`
	BuyBoxIsAmazon                 bool     `json:"buyBoxIsAmazon,omitempty" bson:"buybox_is_amazon,omitempty"`
	BuyBoxIsMAP                    bool     `json:"buyBoxIsMAP,omitempty" bson:"buybox_is_map,omitempty"`
	BuyBoxIsUsed                   bool     `json:"buyBoxIsUsed,omitempty" bson:"buybox_is_used,omitempty"`
	SellerIdsLowestFBA             []string `json:"sellerIdsLowestFBA,omitempty" bson:"seller_ids_lowest_fba,omitempty"`
	SellerIdsLowestFBM             []string `json:"sellerIdsLowestFBM,omitempty" bson:"seller_ids_lowest_fbm,omitempty"`
	OfferCountFBA                  int      `json:"offerCountFBA,omitempty" bson:"offer_count_fba,omitempty"`
	OfferCountFBM                  int      `json:"offerCountFBM,omitempty" bson:"offer_count_fbm,omitempty"`
	SalesRankDrops30               int      `json:"salesRankDrops30,omitempty" bson:"sales_rank_drops30,omitempty"`
	SalesRankDrops90               int      `json:"salesRankDrops90,omitempty" bson:"sales_rank_drops90,omitempty"`
	SalesRankDrops180              int      `json:"salesRankDrops180,omitempty" bson:"sales_rank_drops180,omitempty"`
	SalesRankDrops365              int      `json:"salesRankDrops365,omitempty" bson:"sales_rank_drops365,omitempty"`
}

// ASINRecord 存储在 product_finder_asins 集合中的 ASIN 记录
type ASINRecord struct {
	ID            primitive.ObjectID `bson:"_id,omitempty"`
	ASIN          string             `bson:"asin,omitempty"`
	ASINs         []string           `bson:"asins,omitempty"`
	Domain        int                `bson:"domain"`
	DetailFetched bool               `bson:"detail_fetched"`
	FetchedAt     time.Time          `bson:"fetched_at,omitempty"`
	CreatedAt     time.Time          `bson:"created_at,omitempty"`
	UpdatedAt     time.Time          `bson:"updated_at,omitempty"`
}
