package repository

import (
	"context"
	"fmt"
	"time"

	"keepa/internal/database"
	"keepa/internal/model"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// KeepaRepository Keepa 数据存储层适配器
// 使用 database.Storage 作为底层存储实现
type KeepaRepository struct {
	storage *database.Storage
	logger  *zap.Logger
}

// NewKeepaRepository 创建新的 KeepaRepository 实例
func NewKeepaRepository(storage *database.Storage, logger *zap.Logger) *KeepaRepository {
	return &KeepaRepository{
		storage: storage,
		logger:  logger,
	}
}

// SaveCategory 保存分类到 MongoDB（upsert）
func (r *KeepaRepository) SaveCategory(ctx context.Context, category *model.KeepaCategory) error {
	if category == nil {
		return fmt.Errorf("category cannot be nil")
	}

	now := time.Now()

	// 设置时间戳
	if category.CreatedAt.IsZero() {
		category.CreatedAt = now
	}
	category.UpdatedAt = now

	// 使用 domain_id 和 cat_id 作为唯一标识进行 upsert
	filter := bson.M{
		"domain_id": category.DomainID,
		"cat_id":    category.CatID,
	}

	update := bson.M{
		"$set": bson.M{
			"name":       category.Name,
			"parent":     category.Parent,
			"children":   category.Children,
			"raw_data":   category.RawData,
			"updated_at": now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	result, err := r.storage.UpsertOne(ctx, model.CollectionKeepaCategories, filter, update)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to save category",
				zap.Int64("cat_id", category.CatID),
				zap.Int("domain_id", category.DomainID),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to save category: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("saved category",
			zap.Int64("cat_id", category.CatID),
			zap.String("name", category.Name),
			zap.Int64("upserted_count", result.UpsertedCount),
			zap.Int64("modified_count", result.ModifiedCount),
		)
	}

	return nil
}

// SaveASINs 批量保存 ASIN 到 MongoDB（批量 upsert）
func (r *KeepaRepository) SaveASINs(ctx context.Context, asins []*model.KeepaASIN) error {
	if len(asins) == 0 {
		return nil
	}

	now := time.Now()
	collection := r.storage.GetCollection(model.CollectionKeepaASINs)

	// 构建批量写入操作
	var models []mongo.WriteModel
	for _, asin := range asins {
		if asin == nil || asin.ASIN == "" {
			continue
		}

		filter := bson.M{
			"asin":      asin.ASIN,
			"domain_id": asin.DomainID,
		}

		update := bson.M{
			"$set": bson.M{
				"category_id":  asin.CategoryID,
				"query_source": asin.QuerySource,
				"updated_at":   now,
			},
			"$setOnInsert": bson.M{
				"detail_fetched": false,
				"created_at":     now,
			},
		}

		model := mongo.NewUpdateOneModel().
			SetFilter(filter).
			SetUpdate(update).
			SetUpsert(true)

		models = append(models, model)
	}

	if len(models) == 0 {
		return nil
	}

	// 执行批量写入
	opts := options.BulkWrite().SetOrdered(false) // 无序执行，提高性能
	result, err := collection.BulkWrite(ctx, models, opts)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to save ASINs",
				zap.Int("count", len(asins)),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to save ASINs: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("saved ASINs",
			zap.Int64("inserted_count", result.UpsertedCount),
			zap.Int64("modified_count", result.ModifiedCount),
			zap.Int64("matched_count", result.MatchedCount),
		)
	}

	return nil
}

// GetPendingASINs 获取未获取详情的 ASIN 列表
func (r *KeepaRepository) GetPendingASINs(ctx context.Context, limit int) ([]*model.KeepaASIN, error) {
	if limit <= 0 {
		limit = 100 // 默认限制
	}

	// 查询 detail_fetched=false 的记录
	filter := bson.M{
		"$or": []bson.M{
			{"detail_fetched": false},
			{"detail_fetched": bson.M{"$exists": false}},
		},
	}

	opts := options.Find().
		SetLimit(int64(limit)).
		SetSort(bson.M{"created_at": 1}) // 按创建时间升序，先处理旧的

	cursor, err := r.storage.FindDocuments(ctx, model.CollectionKeepaASINs, filter, opts)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to query pending ASINs",
				zap.Int("limit", limit),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to query pending ASINs: %w", err)
	}
	defer cursor.Close(ctx)

	var asins []*model.KeepaASIN
	if err := cursor.All(ctx, &asins); err != nil {
		if r.logger != nil {
			r.logger.Error("failed to decode pending ASINs",
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to decode pending ASINs: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("found pending ASINs",
			zap.Int("count", len(asins)),
		)
	}

	return asins, nil
}

// SaveProduct 保存产品到 MongoDB（upsert）
func (r *KeepaRepository) SaveProduct(ctx context.Context, product *model.KeepaProduct) error {
	if product == nil {
		return fmt.Errorf("product cannot be nil")
	}

	now := time.Now()

	// 设置时间戳
	if product.CreatedAt.IsZero() {
		product.CreatedAt = now
	}
	product.UpdatedAt = now
	product.FetchedAt = now

	// 使用 asin 和 domain_id 作为唯一标识进行 upsert
	filter := bson.M{
		"asin":      product.ASIN,
		"domain_id": product.DomainID,
	}

	update := bson.M{
		"$set": bson.M{
			// 基础信息
			"title":         product.Title,
			"brand":         product.Brand,
			"manufacturer":  product.Manufacturer,
			"product_group": product.ProductGroup,
			"root_category": product.RootCategory,
			"categories":    product.Categories,

			// 价格信息
			"current_price":     product.CurrentPrice,
			"buybox_price":      product.BuyBoxPrice,
			"amazon_price":      product.AmazonPrice,
			"new_price":         product.NewPrice,
			"used_price":        product.UsedPrice,
			"buybox_shipping":   product.BuyBoxShipping,
			"sales_rank":        product.SalesRank,
			"offer_count_new":   product.OfferCountNew,
			"offer_count_used":  product.OfferCountUsed,
			"total_offer_count": product.TotalOfferCount,

			// 评价信息
			"rating":       product.Rating,
			"review_count": product.ReviewCount,

			// 尺寸和重量
			"package_height": product.PackageHeight,
			"package_length": product.PackageLength,
			"package_width":  product.PackageWidth,
			"package_weight": product.PackageWeight,
			"item_height":    product.ItemHeight,
			"item_length":    product.ItemLength,
			"item_width":     product.ItemWidth,
			"item_weight":    product.ItemWeight,

			// 状态标识
			"is_fba":           product.IsFBA,
			"is_amazon":        product.IsAmazon,
			"is_prime":         product.IsPrime,
			"is_adult_product": product.IsAdultProduct,

			// 历史数据
			"csv":        product.CSV,
			"stats":      product.Stats,
			"images_csv": product.ImagesCSV,

			// 原始数据
			"raw_data": product.RawData,

			// 时间戳
			"last_update": product.LastUpdate,
			"fetched_at":  now,
			"updated_at":  now,
		},
		"$setOnInsert": bson.M{
			"created_at": now,
		},
	}

	result, err := r.storage.UpsertOne(ctx, model.CollectionKeepaProducts, filter, update)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to save product",
				zap.String("asin", product.ASIN),
				zap.Int("domain_id", product.DomainID),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to save product: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("saved product",
			zap.String("asin", product.ASIN),
			zap.String("title", product.Title),
			zap.Int64("upserted_count", result.UpsertedCount),
			zap.Int64("modified_count", result.ModifiedCount),
		)
	}

	return nil
}

// UpdateASINDetailStatus 更新 ASIN 的详情获取状态
func (r *KeepaRepository) UpdateASINDetailStatus(ctx context.Context, asin string, domainID int, fetched bool) error {
	if asin == "" {
		return fmt.Errorf("asin cannot be empty")
	}

	now := time.Now()

	filter := bson.M{
		"asin":      asin,
		"domain_id": domainID,
	}

	updateFields := bson.M{
		"detail_fetched": fetched,
		"updated_at":     now,
	}

	// 如果标记为已获取，设置获取时间
	if fetched {
		updateFields["fetched_at"] = now
	}

	update := bson.M{"$set": updateFields}

	result, err := r.storage.UpsertOne(ctx, model.CollectionKeepaASINs, filter, update)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to update ASIN detail status",
				zap.String("asin", asin),
				zap.Int("domain_id", domainID),
				zap.Bool("fetched", fetched),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to update ASIN detail status: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("updated ASIN detail status",
			zap.String("asin", asin),
			zap.Bool("fetched", fetched),
			zap.Int64("matched_count", result.MatchedCount),
			zap.Int64("modified_count", result.ModifiedCount),
		)
	}

	return nil
}

// BatchUpdateASINDetailStatus 批量更新 ASIN 的详情获取状态
func (r *KeepaRepository) BatchUpdateASINDetailStatus(ctx context.Context, asins []string, domainID int, fetched bool) error {
	if len(asins) == 0 {
		return nil
	}

	now := time.Now()

	filter := bson.M{
		"asin":      bson.M{"$in": asins},
		"domain_id": domainID,
	}

	updateFields := bson.M{
		"detail_fetched": fetched,
		"updated_at":     now,
	}

	// 如果标记为已获取，设置获取时间
	if fetched {
		updateFields["fetched_at"] = now
	}

	update := bson.M{"$set": updateFields}

	result, err := r.storage.UpdateMany(ctx, model.CollectionKeepaASINs, filter, update)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to batch update ASIN detail status",
				zap.Int("count", len(asins)),
				zap.Int("domain_id", domainID),
				zap.Bool("fetched", fetched),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to batch update ASIN detail status: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("batch updated ASIN detail status",
			zap.Int("count", len(asins)),
			zap.Bool("fetched", fetched),
			zap.Int64("matched_count", result.MatchedCount),
			zap.Int64("modified_count", result.ModifiedCount),
		)
	}

	return nil
}

// GetASINByCode 根据 ASIN 码获取 ASIN 记录
func (r *KeepaRepository) GetASINByCode(ctx context.Context, asin string, domainID int) (*model.KeepaASIN, error) {
	filter := bson.M{
		"asin":      asin,
		"domain_id": domainID,
	}

	opts := options.Find().SetLimit(1)
	cursor, err := r.storage.FindDocuments(ctx, model.CollectionKeepaASINs, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query ASIN: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*model.KeepaASIN
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode ASIN: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results[0], nil
}

// GetProductByASIN 根据 ASIN 码获取产品
func (r *KeepaRepository) GetProductByASIN(ctx context.Context, asin string, domainID int) (*model.KeepaProduct, error) {
	filter := bson.M{
		"asin":      asin,
		"domain_id": domainID,
	}

	opts := options.Find().SetLimit(1)
	cursor, err := r.storage.FindDocuments(ctx, model.CollectionKeepaProducts, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query product: %w", err)
	}
	defer cursor.Close(ctx)

	var results []*model.KeepaProduct
	if err := cursor.All(ctx, &results); err != nil {
		return nil, fmt.Errorf("failed to decode product: %w", err)
	}

	if len(results) == 0 {
		return nil, nil
	}

	return results[0], nil
}

// CountPendingASINs 统计未获取详情的 ASIN 数量
func (r *KeepaRepository) CountPendingASINs(ctx context.Context) (int64, error) {
	collection := r.storage.GetCollection(model.CollectionKeepaASINs)

	filter := bson.M{
		"$or": []bson.M{
			{"detail_fetched": false},
			{"detail_fetched": bson.M{"$exists": false}},
		},
	}

	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count pending ASINs: %w", err)
	}

	return count, nil
}

// EnsureIndexes 创建必要的索引
func (r *KeepaRepository) EnsureIndexes(ctx context.Context) error {
	// ASIN 集合索引
	asinCollection := r.storage.GetCollection(model.CollectionKeepaASINs)
	asinIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "asin", Value: 1}, {Key: "domain_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "detail_fetched", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "created_at", Value: 1}},
		},
	}

	_, err := asinCollection.Indexes().CreateMany(ctx, asinIndexes)
	if err != nil {
		return fmt.Errorf("failed to create ASIN indexes: %w", err)
	}

	// 产品集合索引
	productCollection := r.storage.GetCollection(model.CollectionKeepaProducts)
	productIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "asin", Value: 1}, {Key: "domain_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "root_category", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "brand", Value: 1}},
		},
		{
			Keys: bson.D{{Key: "sales_rank", Value: 1}},
		},
	}

	_, err = productCollection.Indexes().CreateMany(ctx, productIndexes)
	if err != nil {
		return fmt.Errorf("failed to create product indexes: %w", err)
	}

	// 分类集合索引
	categoryCollection := r.storage.GetCollection(model.CollectionKeepaCategories)
	categoryIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "domain_id", Value: 1}, {Key: "cat_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "parent", Value: 1}},
		},
	}

	_, err = categoryCollection.Indexes().CreateMany(ctx, categoryIndexes)
	if err != nil {
		return fmt.Errorf("failed to create category indexes: %w", err)
	}

	if r.logger != nil {
		r.logger.Info("ensured all Keepa repository indexes")
	}

	return nil
}

// GetStorage 获取底层的 Storage 实例
func (r *KeepaRepository) GetStorage() *database.Storage {
	return r.storage
}

// SaveRawResponse 保存 API 原始响应到 MongoDB
func (r *KeepaRepository) SaveRawResponse(ctx context.Context, rawResponse *model.KeepaRawResponse) error {
	if rawResponse == nil {
		return fmt.Errorf("rawResponse cannot be nil")
	}

	now := time.Now()
	rawResponse.CreatedAt = now

	collection := r.storage.GetCollection(model.CollectionKeepaRawResponses)

	result, err := collection.InsertOne(ctx, rawResponse)
	if err != nil {
		if r.logger != nil {
			r.logger.Error("failed to save raw response",
				zap.String("api_endpoint", rawResponse.APIEndpoint),
				zap.String("request_type", rawResponse.RequestType),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to save raw response: %w", err)
	}

	if r.logger != nil {
		r.logger.Debug("saved raw response",
			zap.String("api_endpoint", rawResponse.APIEndpoint),
			zap.Int("response_size", rawResponse.ResponseSize),
			zap.Any("inserted_id", result.InsertedID),
		)
	}

	return nil
}
