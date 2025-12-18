package tasks

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"keepa/internal/api"
	"keepa/internal/api/keepa/best_sellers"
	"keepa/internal/api/keepa/browsing_deals"
	"keepa/internal/api/keepa/category_lookup"
	"keepa/internal/api/keepa/category_searches"
	"keepa/internal/api/keepa/lightning_deals"
	"keepa/internal/api/keepa/most_rated_sellers"
	"keepa/internal/api/keepa/product_finder"
	"keepa/internal/api/keepa/product_searches"
	"keepa/internal/api/keepa/products"
	"keepa/internal/api/keepa/seller_information"
	"keepa/internal/config"

	"go.uber.org/zap"
)

// KeepaRawDataSaver Keepa API 原始数据获取和存储服务
type KeepaRawDataSaver struct {
	client        *api.Client
	storage       *api.Storage
	logger        *zap.Logger
	queriesConfig *config.KeepaQueriesConfig
}

// NewKeepaRawDataSaver 创建新的 KeepaRawDataSaver 实例
func NewKeepaRawDataSaver(client *api.Client, storage *api.Storage, queriesConfig *config.KeepaQueriesConfig, logger *zap.Logger) *KeepaRawDataSaver {
	return &KeepaRawDataSaver{
		client:        client,
		storage:       storage,
		logger:        logger,
		queriesConfig: queriesConfig,
	}
}

// FetchAndStoreAll 获取并存储所有启用的 API 数据
func (f *KeepaRawDataSaver) FetchAndStoreAll(ctx context.Context) error {
	if f.logger != nil {
		f.logger.Info("starting to fetch and store all enabled keepa API data")
	}

	// Best Sellers
	if f.queriesConfig.BestSellers.Enabled {
		if err := f.fetchAndStoreBestSellers(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store best sellers", zap.Error(err))
			}
			// 继续处理其他 API，不中断
		}
	}

	// Category Lookup
	if f.queriesConfig.CategoryLookup.Enabled {
		if err := f.fetchAndStoreCategoryLookup(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store category lookup", zap.Error(err))
			}
		}
	}

	// Browsing Deals
	if f.queriesConfig.BrowsingDeals.Enabled {
		if err := f.fetchAndStoreBrowsingDeals(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store browsing deals", zap.Error(err))
			}
		}
	}

	// Products
	if f.queriesConfig.Products.Enabled {
		if err := f.fetchAndStoreProducts(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store products", zap.Error(err))
			}
		}
	}

	// Category Searches
	if f.queriesConfig.CategorySearches.Enabled {
		if err := f.fetchAndStoreCategorySearches(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store category searches", zap.Error(err))
			}
		}
	}

	// Lightning Deals
	if f.queriesConfig.LightningDeals.Enabled {
		if err := f.fetchAndStoreLightningDeals(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store lightning deals", zap.Error(err))
			}
		}
	}

	// Most Rated Sellers
	if f.queriesConfig.MostRatedSellers.Enabled {
		if err := f.fetchAndStoreMostRatedSellers(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store most rated sellers", zap.Error(err))
			}
		}
	}

	// Product Finder
	if f.queriesConfig.ProductFinder.Enabled {
		if err := f.fetchAndStoreProductFinder(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store product finder", zap.Error(err))
			}
		}
	}

	// Product Searches
	if f.queriesConfig.ProductSearches.Enabled {
		if err := f.fetchAndStoreProductSearches(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store product searches", zap.Error(err))
			}
		}
	}

	// Seller Information
	if f.queriesConfig.SellerInfo.Enabled {
		if err := f.fetchAndStoreSellerInfo(ctx); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch and store seller information", zap.Error(err))
			}
		}
	}

	if f.logger != nil {
		f.logger.Info("completed fetching and storing all enabled keepa API data")
	}

	return nil
}

// fetchAndStoreBestSellers 获取并存储 Best Sellers 数据
func (f *KeepaRawDataSaver) fetchAndStoreBestSellers(ctx context.Context) error {
	service := best_sellers.NewService(f.client, f.logger)
	collection := f.queriesConfig.BestSellers.Collection
	if collection == "" {
		collection = "best_sellers"
	}

	for _, task := range f.queriesConfig.BestSellers.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToBestSellersParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse best sellers params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.Fetch(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch best sellers data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		// 存储数据
		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save best sellers data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("best sellers data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// fetchAndStoreCategoryLookup 获取并存储 Category Lookup 数据
func (f *KeepaRawDataSaver) fetchAndStoreCategoryLookup(ctx context.Context) error {
	service := category_lookup.NewService(f.client, f.logger)
	collection := f.queriesConfig.CategoryLookup.Collection
	if collection == "" {
		collection = "category_lookup"
	}

	for _, task := range f.queriesConfig.CategoryLookup.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToCategoryLookupParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse category lookup params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.Fetch(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch category lookup data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save category lookup data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("category lookup data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// fetchAndStoreBrowsingDeals 获取并存储 Browsing Deals 数据
func (f *KeepaRawDataSaver) fetchAndStoreBrowsingDeals(ctx context.Context) error {
	service := browsing_deals.NewService(f.client, f.logger)
	collection := f.queriesConfig.BrowsingDeals.Collection
	if collection == "" {
		collection = "browsing_deals"
	}

	for _, task := range f.queriesConfig.BrowsingDeals.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToBrowsingDealsParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse browsing deals params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.Fetch(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch browsing deals data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save browsing deals data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("browsing deals data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// fetchAndStoreProducts 获取并存储 Products 数据
func (f *KeepaRawDataSaver) fetchAndStoreProducts(ctx context.Context) error {
	service := products.NewService(f.client, f.logger)
	collection := f.queriesConfig.Products.Collection
	if collection == "" {
		collection = "products"
	}

	for _, task := range f.queriesConfig.Products.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToProductsParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse products params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.Fetch(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch products data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save products data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("products data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// fetchAndStoreCategorySearches 获取并存储 Category Searches 数据
func (f *KeepaRawDataSaver) fetchAndStoreCategorySearches(ctx context.Context) error {
	service := category_searches.NewService(f.client, f.logger)
	collection := f.queriesConfig.CategorySearches.Collection
	if collection == "" {
		collection = "category_searches"
	}

	for _, task := range f.queriesConfig.CategorySearches.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToCategorySearchesParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse category searches params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.Fetch(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch category searches data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save category searches data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("category searches data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// fetchAndStoreLightningDeals 获取并存储 Lightning Deals 数据
func (f *KeepaRawDataSaver) fetchAndStoreLightningDeals(ctx context.Context) error {
	service := lightning_deals.NewService(f.client, f.logger)
	collection := f.queriesConfig.LightningDeals.Collection
	if collection == "" {
		collection = "lightning_deals"
	}

	for _, task := range f.queriesConfig.LightningDeals.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToLightningDealsParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse lightning deals params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.Fetch(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch lightning deals data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save lightning deals data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("lightning deals data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// fetchAndStoreMostRatedSellers 获取并存储 Most Rated Sellers 数据
func (f *KeepaRawDataSaver) fetchAndStoreMostRatedSellers(ctx context.Context) error {
	service := most_rated_sellers.NewService(f.client, f.logger)
	collection := f.queriesConfig.MostRatedSellers.Collection
	if collection == "" {
		collection = "most_rated_sellers"
	}

	for _, task := range f.queriesConfig.MostRatedSellers.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToMostRatedSellersParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse most rated sellers params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.FetchRaw(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch most rated sellers data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save most rated sellers data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("most rated sellers data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// fetchAndStoreProductFinder 获取并存储 Product Finder 数据
func (f *KeepaRawDataSaver) fetchAndStoreProductFinder(ctx context.Context) error {
	service := product_finder.NewService(f.client, f.logger)
	collection := f.queriesConfig.ProductFinder.Collection
	if collection == "" {
		collection = "product_finder"
	}

	for _, task := range f.queriesConfig.ProductFinder.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToProductFinderParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse product finder params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.Fetch(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch product finder data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save product finder data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("product finder data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// fetchAndStoreProductSearches 获取并存储 Product Searches 数据
func (f *KeepaRawDataSaver) fetchAndStoreProductSearches(ctx context.Context) error {
	service := product_searches.NewService(f.client, f.logger)
	collection := f.queriesConfig.ProductSearches.Collection
	if collection == "" {
		collection = "product_searches"
	}

	for _, task := range f.queriesConfig.ProductSearches.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToProductSearchesParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse product searches params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.Fetch(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch product searches data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save product searches data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("product searches data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// fetchAndStoreSellerInfo 获取并存储 Seller Information 数据
func (f *KeepaRawDataSaver) fetchAndStoreSellerInfo(ctx context.Context) error {
	service := seller_information.NewService(f.client, f.logger)
	collection := f.queriesConfig.SellerInfo.Collection
	if collection == "" {
		collection = "seller_information"
	}

	for _, task := range f.queriesConfig.SellerInfo.Tasks {
		if !task.Enabled {
			continue
		}

		params, err := f.mapToSellerInfoParams(task.Params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to parse seller info params",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		data, err := service.Fetch(ctx, params)
		if err != nil {
			if f.logger != nil {
				f.logger.Error("failed to fetch seller info data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if err := f.storage.SaveRawData(ctx, collection, data); err != nil {
			if f.logger != nil {
				f.logger.Error("failed to save seller info data",
					zap.String("task", task.Name),
					zap.Error(err),
				)
			}
			continue
		}

		if f.logger != nil {
			f.logger.Info("seller info data fetched and stored successfully",
				zap.String("task", task.Name),
			)
		}
	}

	return nil
}

// 参数转换辅助函数

func (f *KeepaRawDataSaver) mapToBestSellersParams(m map[string]interface{}) (best_sellers.RequestParams, error) {
	var params best_sellers.RequestParams

	if domain, ok := m["domain"].(int); ok {
		params.Domain = domain
	} else if domain, ok := m["domain"].(float64); ok {
		params.Domain = int(domain)
	} else {
		return params, fmt.Errorf("domain must be an integer")
	}

	if category, ok := m["category"].(string); ok {
		params.Category = category
	} else {
		return params, fmt.Errorf("category must be a string")
	}

	if rangeVal, ok := m["range"]; ok {
		if r, ok := rangeVal.(int); ok {
			params.Range = &r
		} else if r, ok := rangeVal.(float64); ok {
			rInt := int(r)
			params.Range = &rInt
		}
	}

	if month, ok := m["month"]; ok {
		if m, ok := month.(int); ok {
			params.Month = &m
		} else if m, ok := month.(float64); ok {
			mInt := int(m)
			params.Month = &mInt
		}
	}

	if year, ok := m["year"]; ok {
		if y, ok := year.(int); ok {
			params.Year = &y
		} else if y, ok := year.(float64); ok {
			yInt := int(y)
			params.Year = &yInt
		}
	}

	if variations, ok := m["variations"]; ok {
		if v, ok := variations.(int); ok {
			params.Variations = &v
		} else if v, ok := variations.(float64); ok {
			vInt := int(v)
			params.Variations = &vInt
		}
	}

	if sublist, ok := m["sublist"]; ok {
		if s, ok := sublist.(int); ok {
			params.Sublist = &s
		} else if s, ok := sublist.(float64); ok {
			sInt := int(s)
			params.Sublist = &sInt
		}
	}

	return params, nil
}

func (f *KeepaRawDataSaver) mapToCategoryLookupParams(m map[string]interface{}) (category_lookup.RequestParams, error) {
	var params category_lookup.RequestParams

	if domain, ok := m["domain"].(int); ok {
		params.Domain = domain
	} else if domain, ok := m["domain"].(float64); ok {
		params.Domain = int(domain)
	} else {
		return params, fmt.Errorf("domain must be an integer")
	}

	if category, ok := m["category"]; ok {
		switch v := category.(type) {
		case []interface{}:
			params.Category = make([]int, len(v))
			for i, item := range v {
				if val, ok := item.(int); ok {
					params.Category[i] = val
				} else if val, ok := item.(float64); ok {
					params.Category[i] = int(val)
				} else {
					return params, fmt.Errorf("category[%d] must be an integer", i)
				}
			}
		case []int:
			params.Category = v
		default:
			return params, fmt.Errorf("category must be an array of integers")
		}
	} else {
		return params, fmt.Errorf("category is required")
	}

	if includeParents, ok := m["includeParents"]; ok {
		if ip, ok := includeParents.(int); ok {
			params.IncludeParents = &ip
		} else if ip, ok := includeParents.(float64); ok {
			ipInt := int(ip)
			params.IncludeParents = &ipInt
		}
	}

	return params, nil
}

func (f *KeepaRawDataSaver) mapToBrowsingDealsParams(m map[string]interface{}) (browsing_deals.RequestParams, error) {
	var params browsing_deals.RequestParams

	if domainId, ok := m["domainId"].(int); ok {
		params.DomainID = domainId
	} else if domainId, ok := m["domainId"].(float64); ok {
		params.DomainID = int(domainId)
	} else {
		return params, fmt.Errorf("domainId must be an integer")
	}

	if priceTypes, ok := m["priceTypes"]; ok {
		switch v := priceTypes.(type) {
		case []interface{}:
			params.PriceTypes = make([]int, len(v))
			for i, item := range v {
				if val, ok := item.(int); ok {
					params.PriceTypes[i] = val
				} else if val, ok := item.(float64); ok {
					params.PriceTypes[i] = int(val)
				} else {
					return params, fmt.Errorf("priceTypes[%d] must be an integer", i)
				}
			}
		case []int:
			params.PriceTypes = v
		default:
			return params, fmt.Errorf("priceTypes must be an array of integers")
		}
	} else {
		return params, fmt.Errorf("priceTypes is required")
	}

	if dateRange, ok := m["dateRange"].(int); ok {
		params.DateRange = dateRange
	} else if dateRange, ok := m["dateRange"].(float64); ok {
		params.DateRange = int(dateRange)
	} else {
		return params, fmt.Errorf("dateRange must be an integer")
	}

	// 处理可选参数
	if page, ok := m["page"]; ok {
		if p, ok := page.(int); ok {
			params.Page = &p
		} else if p, ok := page.(float64); ok {
			pInt := int(p)
			params.Page = &pInt
		}
	}

	// 处理布尔值参数
	boolFields := []string{"isFilterEnabled", "isLowest", "isLowest90", "isLowestOffer", "isHighest",
		"isOutOfStock", "isBackInStock", "hasReviews", "filterErotic", "singleVariation",
		"isRisers", "isPrimeExclusive", "mustHaveAmazonOffer", "mustNotHaveAmazonOffer"}

	for _, field := range boolFields {
		if val, ok := m[field]; ok {
			if b, ok := val.(bool); ok {
				reflect.ValueOf(&params).Elem().FieldByName(field).Set(reflect.ValueOf(&b))
			}
		}
	}

	return params, nil
}

func (f *KeepaRawDataSaver) mapToProductsParams(m map[string]interface{}) (products.RequestParams, error) {
	var params products.RequestParams

	if domain, ok := m["domain"].(int); ok {
		params.Domain = domain
	} else if domain, ok := m["domain"].(float64); ok {
		params.Domain = int(domain)
	} else {
		return params, fmt.Errorf("domain must be an integer")
	}

	if asins, ok := m["asins"]; ok {
		switch v := asins.(type) {
		case []interface{}:
			params.ASINs = make([]string, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					params.ASINs[i] = str
				} else {
					return params, fmt.Errorf("asins[%d] must be a string", i)
				}
			}
		case []string:
			params.ASINs = v
		default:
			return params, fmt.Errorf("asins must be an array of strings")
		}
	}

	if codes, ok := m["codes"]; ok {
		switch v := codes.(type) {
		case []interface{}:
			params.Codes = make([]string, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					params.Codes[i] = str
				} else {
					return params, fmt.Errorf("codes[%d] must be a string", i)
				}
			}
		case []string:
			params.Codes = v
		default:
			return params, fmt.Errorf("codes must be an array of strings")
		}
	}

	// 处理可选整数参数
	intFields := []string{"update", "history", "days", "code-limit", "offers",
		"only-live-offers", "rental", "videos", "aplus", "rating", "buybox", "stock", "historical-variations"}

	for _, field := range intFields {
		fieldName := field
		if field == "code-limit" {
			fieldName = "CodeLimit"
		} else if field == "only-live-offers" {
			fieldName = "OnlyLiveOffers"
		} else if field == "historical-variations" {
			fieldName = "HistoricalVariations"
		} else {
			fieldName = field[:1] + field[1:]
		}

		if val, ok := m[field]; ok {
			if i, ok := val.(int); ok {
				reflect.ValueOf(&params).Elem().FieldByName(fieldName).Set(reflect.ValueOf(&i))
			} else if i, ok := val.(float64); ok {
				iInt := int(i)
				reflect.ValueOf(&params).Elem().FieldByName(fieldName).Set(reflect.ValueOf(&iInt))
			}
		}
	}

	return params, nil
}

// 其他 API 的参数转换函数（简化实现，可根据需要扩展）
func (f *KeepaRawDataSaver) mapToCategorySearchesParams(m map[string]interface{}) (category_searches.RequestParams, error) {
	var params category_searches.RequestParams

	if domain, ok := m["domain"].(int); ok {
		params.Domain = domain
	} else if domain, ok := m["domain"].(float64); ok {
		params.Domain = int(domain)
	} else {
		return params, fmt.Errorf("domain must be an integer")
	}

	if term, ok := m["term"].(string); ok {
		params.Term = term
	} else {
		return params, fmt.Errorf("term must be a string")
	}

	return params, nil
}

func (f *KeepaRawDataSaver) mapToLightningDealsParams(m map[string]interface{}) (lightning_deals.RequestParams, error) {
	var params lightning_deals.RequestParams

	if domain, ok := m["domain"].(int); ok {
		params.Domain = domain
	} else if domain, ok := m["domain"].(float64); ok {
		params.Domain = int(domain)
	} else {
		return params, fmt.Errorf("domain must be an integer")
	}

	if asin, ok := m["asin"].(string); ok {
		params.ASIN = asin
	}

	if state, ok := m["state"].(string); ok {
		params.State = lightning_deals.LightningDealState(state)
	}

	return params, nil
}

func (f *KeepaRawDataSaver) mapToMostRatedSellersParams(m map[string]interface{}) (most_rated_sellers.RequestParams, error) {
	var params most_rated_sellers.RequestParams

	if domain, ok := m["domain"].(int); ok {
		params.Domain = domain
	} else if domain, ok := m["domain"].(float64); ok {
		params.Domain = int(domain)
	} else {
		return params, fmt.Errorf("domain must be an integer")
	}

	return params, nil
}

func (f *KeepaRawDataSaver) mapToProductFinderParams(m map[string]interface{}) (product_finder.RequestParams, error) {
	var params product_finder.RequestParams

	if domain, ok := m["domain"].(int); ok {
		params.Domain = domain
	} else if domain, ok := m["domain"].(float64); ok {
		params.Domain = int(domain)
	} else {
		return params, fmt.Errorf("domain must be an integer")
	}

	// 处理 query 字段（需要根据实际结构实现）
	if query, ok := m["query"].(map[string]interface{}); ok {
		// 这里需要根据 product_finder.QueryJSON 的实际结构进行转换
		// 简化实现，实际使用时需要完善
		queryBytes, _ := json.Marshal(query)
		json.Unmarshal(queryBytes, &params.Query)
	}

	return params, nil
}

func (f *KeepaRawDataSaver) mapToProductSearchesParams(m map[string]interface{}) (product_searches.RequestParams, error) {
	var params product_searches.RequestParams

	if domain, ok := m["domain"].(int); ok {
		params.Domain = domain
	} else if domain, ok := m["domain"].(float64); ok {
		params.Domain = int(domain)
	} else {
		return params, fmt.Errorf("domain must be an integer")
	}

	if term, ok := m["term"].(string); ok {
		params.Term = term
	} else {
		return params, fmt.Errorf("term must be a string")
	}

	if page, ok := m["page"]; ok {
		if p, ok := page.(int); ok {
			params.Page = &p
		} else if p, ok := page.(float64); ok {
			pInt := int(p)
			params.Page = &pInt
		}
	}

	if asinsOnly, ok := m["asins-only"]; ok {
		if ao, ok := asinsOnly.(int); ok {
			params.AsinsOnly = &ao
		} else if ao, ok := asinsOnly.(float64); ok {
			aoInt := int(ao)
			params.AsinsOnly = &aoInt
		}
	}

	return params, nil
}

func (f *KeepaRawDataSaver) mapToSellerInfoParams(m map[string]interface{}) (seller_information.RequestParams, error) {
	var params seller_information.RequestParams

	if domain, ok := m["domain"].(int); ok {
		params.Domain = domain
	} else if domain, ok := m["domain"].(float64); ok {
		params.Domain = int(domain)
	} else {
		return params, fmt.Errorf("domain must be an integer")
	}

	if seller, ok := m["seller"].(string); ok {
		params.SellerIDs = []string{seller}
	} else if sellers, ok := m["seller_ids"]; ok {
		switch v := sellers.(type) {
		case []interface{}:
			params.SellerIDs = make([]string, len(v))
			for i, item := range v {
				if str, ok := item.(string); ok {
					params.SellerIDs[i] = str
				} else {
					return params, fmt.Errorf("seller_ids[%d] must be a string", i)
				}
			}
		case []string:
			params.SellerIDs = v
		default:
			return params, fmt.Errorf("seller_ids must be an array of strings")
		}
	} else {
		return params, fmt.Errorf("seller or seller_ids must be provided")
	}

	if storefront, ok := m["storefront"]; ok {
		if sf, ok := storefront.(int); ok {
			params.Storefront = &sf
		} else if sf, ok := storefront.(float64); ok {
			sfInt := int(sf)
			params.Storefront = &sfInt
		}
	}

	if update, ok := m["update"]; ok {
		if u, ok := update.(int); ok {
			params.Update = &u
		} else if u, ok := update.(float64); ok {
			uInt := int(u)
			params.Update = &uInt
		}
	}

	return params, nil
}
