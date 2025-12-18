package tasks

import (
	"context"
	"testing"
	"time"

	"keepa/internal/api"
	"keepa/internal/database"
	"keepa/internal/model"
	"keepa/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.uber.org/zap"
)

// TestKeepaFetchProducts_RealAPI_FetchProductDetails 测试真实 API 获取产品详情
func TestKeepaFetchProducts_RealAPI_FetchProductDetails(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建真实的 API 客户端
	client := setupRealKeepaClient(t, logger)

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 先插入一些测试 ASIN（使用真实的 ASIN）
	testASINs := []*model.KeepaASIN{
		{
			ASIN:          "B08N5WRWNW", // 真实的 Amazon ASIN
			DomainID:      1,
			CategoryID:    2619533011,
			QuerySource:   "test",
			DetailFetched: false,
		},
		{
			ASIN:          "B07FZ8S74R", // 另一个真实的 ASIN
			DomainID:      1,
			CategoryID:    2619533011,
			QuerySource:   "test",
			DetailFetched: false,
		},
	}

	err := repo.SaveASINs(ctx, testASINs)
	if err != nil {
		t.Fatalf("插入测试 ASIN 失败: %v", err)
	}

	// 创建任务实例
	fetcher := NewKeepaFetchProducts(client, repo, logger)

	// 执行测试
	err = fetcher.FetchProductDetails(ctx)
	if err != nil {
		t.Fatalf("FetchProductDetails 失败: %v", err)
	}

	// 验证产品是否保存到数据库
	product1, err := repo.GetProductByASIN(ctx, "B08N5WRWNW", 1)
	if err != nil {
		t.Fatalf("获取产品失败: %v", err)
	}

	if product1 != nil {
		t.Logf("成功获取产品: %s - %s", product1.ASIN, product1.Title)
		t.Logf("  品牌: %s", product1.Brand)
		t.Logf("  BuyBox 价格: %d", product1.BuyBoxPrice)
		t.Logf("  销售排名: %d", product1.SalesRank)
	}

	// 验证 ASIN 是否标记为已获取
	asin1, _ := repo.GetASINByCode(ctx, "B08N5WRWNW", 1)
	if asin1 != nil && !asin1.DetailFetched {
		t.Error("ASIN 应该标记为已获取详情")
	}

	// 验证原始响应是否保存到 MongoDB
	rawCollection := db.Collection(model.CollectionKeepaRawResponses)
	rawCount, err := rawCollection.CountDocuments(ctx, bson.M{"api_endpoint": "/product"})
	if err != nil {
		t.Errorf("查询原始响应失败: %v", err)
	}

	if rawCount > 0 {
		t.Logf("MongoDB 中保存了 %d 条原始 API 响应", rawCount)

		// 查询并显示原始响应的基本信息
		var rawResponse model.KeepaRawResponse
		err = rawCollection.FindOne(ctx, bson.M{"api_endpoint": "/product"}).Decode(&rawResponse)
		if err == nil {
			t.Logf("原始响应信息:")
			t.Logf("  API 端点: %s", rawResponse.APIEndpoint)
			t.Logf("  请求类型: %s", rawResponse.RequestType)
			t.Logf("  响应大小: %d bytes", rawResponse.ResponseSize)
			t.Logf("  消耗 Token: %d", rawResponse.TokensUsed)
			t.Logf("  剩余 Token: %d", rawResponse.TokensLeft)
			t.Logf("  创建时间: %v", rawResponse.CreatedAt)
		}
	} else {
		t.Error("未找到保存的原始 API 响应")
	}

	t.Log("真实 API 产品详情获取测试完成")
}

// TestKeepaFetchProducts_RealAPI_FetchWithCustomBatchSize 测试自定义批次大小
func TestKeepaFetchProducts_RealAPI_FetchWithCustomBatchSize(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建真实的 API 客户端
	client := setupRealKeepaClient(t, logger)

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 插入多个测试 ASIN
	testASINs := []*model.KeepaASIN{
		{ASIN: "B08N5WRWNW", DomainID: 1, DetailFetched: false},
		{ASIN: "B07FZ8S74R", DomainID: 1, DetailFetched: false},
		{ASIN: "B07H4M4NG3", DomainID: 1, DetailFetched: false},
	}

	err := repo.SaveASINs(ctx, testASINs)
	if err != nil {
		t.Fatalf("插入测试 ASIN 失败: %v", err)
	}

	// 创建任务实例
	fetcher := NewKeepaFetchProducts(client, repo, logger)

	// 使用较小的批次大小（节省 token）
	err = fetcher.FetchProductDetailsWithBatchSize(ctx, 2)
	if err != nil {
		t.Fatalf("FetchProductDetailsWithBatchSize 失败: %v", err)
	}

	// 统计已获取的产品数量
	collection := db.Collection(model.CollectionKeepaProducts)
	count, _ := collection.CountDocuments(ctx, bson.M{})
	t.Logf("已获取 %d 个产品详情", count)

	// 验证原始响应
	rawCollection := db.Collection(model.CollectionKeepaRawResponses)
	rawCount, _ := rawCollection.CountDocuments(ctx, bson.M{"api_endpoint": "/product"})
	t.Logf("保存了 %d 条原始 API 响应", rawCount)
}

// TestKeepaFetchProducts_GetPendingASINs 测试获取待处理的 ASIN
func TestKeepaFetchProducts_GetPendingASINs(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 插入混合状态的 ASIN
	collection := db.Collection(model.CollectionKeepaASINs)
	docs := []interface{}{
		bson.M{"asin": "B001", "domain_id": 1, "detail_fetched": false, "created_at": time.Now()},
		bson.M{"asin": "B002", "domain_id": 1, "detail_fetched": false, "created_at": time.Now()},
		bson.M{"asin": "B003", "domain_id": 1, "detail_fetched": true, "created_at": time.Now()}, // 已获取
	}

	_, err := collection.InsertMany(ctx, docs)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 获取待处理的 ASIN
	pendingASINs, err := repo.GetPendingASINs(ctx, 10)
	if err != nil {
		t.Fatalf("获取待处理 ASIN 失败: %v", err)
	}

	// 应该只有 2 个待处理的
	if len(pendingASINs) != 2 {
		t.Errorf("期望获取 2 个待处理 ASIN，实际获取 %d 个", len(pendingASINs))
	}

	// 验证获取的都是未处理的
	for _, asin := range pendingASINs {
		if asin.DetailFetched {
			t.Errorf("ASIN %s 不应该在待处理列表中", asin.ASIN)
		}
	}

	t.Log("获取待处理 ASIN 测试通过")
}

// TestKeepaFetchProducts_BatchUpdateStatus 测试批量更新状态
func TestKeepaFetchProducts_BatchUpdateStatus(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 插入测试 ASIN
	testASINs := []*model.KeepaASIN{
		{ASIN: "B001", DomainID: 1, DetailFetched: false},
		{ASIN: "B002", DomainID: 1, DetailFetched: false},
		{ASIN: "B003", DomainID: 1, DetailFetched: false},
	}

	err := repo.SaveASINs(ctx, testASINs)
	if err != nil {
		t.Fatalf("插入测试 ASIN 失败: %v", err)
	}

	// 批量更新状态
	asinCodes := []string{"B001", "B002"}
	err = repo.BatchUpdateASINDetailStatus(ctx, asinCodes, 1, true)
	if err != nil {
		t.Fatalf("批量更新状态失败: %v", err)
	}

	// 验证更新结果
	asin1, _ := repo.GetASINByCode(ctx, "B001", 1)
	asin2, _ := repo.GetASINByCode(ctx, "B002", 1)
	asin3, _ := repo.GetASINByCode(ctx, "B003", 1)

	if asin1 == nil || !asin1.DetailFetched {
		t.Error("B001 应该标记为已获取")
	}

	if asin2 == nil || !asin2.DetailFetched {
		t.Error("B002 应该标记为已获取")
	}

	if asin3 == nil || asin3.DetailFetched {
		t.Error("B003 不应该标记为已获取")
	}

	t.Log("批量更新状态测试通过")
}

// TestKeepaFetchProducts_CountPending 测试统计待处理数量
func TestKeepaFetchProducts_CountPending(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 插入混合状态的 ASIN
	collection := db.Collection(model.CollectionKeepaASINs)
	docs := []interface{}{
		bson.M{"asin": "B001", "domain_id": 1, "detail_fetched": false, "created_at": time.Now()},
		bson.M{"asin": "B002", "domain_id": 1, "detail_fetched": false, "created_at": time.Now()},
		bson.M{"asin": "B003", "domain_id": 1, "detail_fetched": true, "created_at": time.Now()},
		bson.M{"asin": "B004", "domain_id": 1, "detail_fetched": false, "created_at": time.Now()},
	}

	_, err := collection.InsertMany(ctx, docs)
	if err != nil {
		t.Fatalf("插入测试数据失败: %v", err)
	}

	// 统计待处理数量
	count, err := repo.CountPendingASINs(ctx)
	if err != nil {
		t.Fatalf("统计待处理数量失败: %v", err)
	}

	if count != 3 {
		t.Errorf("期望待处理数量为 3，实际为 %d", count)
	}

	t.Logf("待处理 ASIN 数量: %d", count)
}

// TestKeepaFetchProducts_SaveProduct 测试保存产品
func TestKeepaFetchProducts_SaveProduct(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 创建测试产品
	product := &model.KeepaProduct{
		ASIN:         "B08N5WRWNW",
		DomainID:     1,
		Title:        "测试产品",
		Brand:        "测试品牌",
		Manufacturer: "测试制造商",
		RootCategory: 2619533011,
		CurrentPrice: 1999,
		BuyBoxPrice:  1999,
		SalesRank:    5000,
		IsFBA:        true,
		IsPrime:      true,
	}

	// 保存产品
	err := repo.SaveProduct(ctx, product)
	if err != nil {
		t.Fatalf("保存产品失败: %v", err)
	}

	// 获取保存的产品
	savedProduct, err := repo.GetProductByASIN(ctx, "B08N5WRWNW", 1)
	if err != nil {
		t.Fatalf("获取产品失败: %v", err)
	}

	if savedProduct == nil {
		t.Fatal("产品未保存到数据库")
	}

	// 验证字段
	if savedProduct.Title != "测试产品" {
		t.Errorf("期望标题为 '测试产品'，实际为 '%s'", savedProduct.Title)
	}

	if savedProduct.BuyBoxPrice != 1999 {
		t.Errorf("期望价格为 1999，实际为 %d", savedProduct.BuyBoxPrice)
	}

	if !savedProduct.IsFBA {
		t.Error("产品应该标记为 FBA")
	}

	t.Log("保存产品测试通过")
}

// TestKeepaFetchProducts_ProductUpsert 测试产品 upsert 行为
func TestKeepaFetchProducts_ProductUpsert(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 第一次保存
	product := &model.KeepaProduct{
		ASIN:        "B08N5WRWNW",
		DomainID:    1,
		Title:       "原标题",
		BuyBoxPrice: 1999,
	}

	err := repo.SaveProduct(ctx, product)
	if err != nil {
		t.Fatalf("第一次保存产品失败: %v", err)
	}

	// 第二次保存（更新）
	product.Title = "新标题"
	product.BuyBoxPrice = 2999
	err = repo.SaveProduct(ctx, product)
	if err != nil {
		t.Fatalf("第二次保存产品失败: %v", err)
	}

	// 验证数据库中只有一条记录
	collection := db.Collection(model.CollectionKeepaProducts)
	count, _ := collection.CountDocuments(ctx, bson.M{"asin": "B08N5WRWNW"})
	if count != 1 {
		t.Errorf("期望数据库中只有 1 条记录，实际有 %d 条", count)
	}

	// 验证数据被更新
	savedProduct, _ := repo.GetProductByASIN(ctx, "B08N5WRWNW", 1)
	if savedProduct.Title != "新标题" {
		t.Errorf("期望标题为 '新标题'，实际为 '%s'", savedProduct.Title)
	}

	if savedProduct.BuyBoxPrice != 2999 {
		t.Errorf("期望价格为 2999，实际为 %d", savedProduct.BuyBoxPrice)
	}

	t.Log("产品 upsert 测试通过")
}

// TestKeepaFetchProducts_ExtractASINCodes 测试提取 ASIN 码
func TestKeepaFetchProducts_ExtractASINCodes(t *testing.T) {
	testCases := []struct {
		name     string
		input    []*model.KeepaASIN
		expected []string
	}{
		{
			name:     "正常情况",
			input:    []*model.KeepaASIN{{ASIN: "B001"}, {ASIN: "B002"}, {ASIN: "B003"}},
			expected: []string{"B001", "B002", "B003"},
		},
		{
			name:     "包含重复",
			input:    []*model.KeepaASIN{{ASIN: "B001"}, {ASIN: "B001"}, {ASIN: "B002"}},
			expected: []string{"B001", "B002"},
		},
		{
			name:     "包含空值",
			input:    []*model.KeepaASIN{{ASIN: "B001"}, {ASIN: ""}, nil, {ASIN: "B002"}},
			expected: []string{"B001", "B002"},
		},
		{
			name:     "空列表",
			input:    []*model.KeepaASIN{},
			expected: []string{},
		},
		{
			name:     "nil 列表",
			input:    nil,
			expected: []string{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := extractASINCodes(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("期望 %d 个结果，实际 %d 个", len(tc.expected), len(result))
				return
			}

			// 验证内容
			for i, asin := range result {
				if asin != tc.expected[i] {
					t.Errorf("索引 %d: 期望 %s，实际 %s", i, tc.expected[i], asin)
				}
			}
		})
	}
}

// TestKeepaFetchProducts_ConvertToKeepaProduct 测试产品转换
func TestKeepaFetchProducts_ConvertToKeepaProduct(t *testing.T) {
	// 测试 nil 输入
	result := convertToKeepaProduct(nil)
	if result != nil {
		t.Error("nil 输入应该返回 nil")
	}

	// 测试正常转换
	details := &model.ProductDetails{
		ASIN:               "B08N5WRWNW",
		DomainID:           1,
		Title:              "测试产品",
		Brand:              "测试品牌",
		RootCategory:       2619533011,
		IsEligibleForPrime: true,
		IsAdultProduct:     false,
		Stats: &model.ProductStats{
			BuyBoxPrice:     1999,
			BuyBoxShipping:  0,
			TotalOfferCount: 10,
			BuyBoxIsFBA:     true,
			BuyBoxIsAmazon:  false,
			OfferCountFBA:   5,
			OfferCountFBM:   5,
			Current:         []int{1999, 1899, 1599, 5000},
		},
	}

	product := convertToKeepaProduct(details)

	if product == nil {
		t.Fatal("转换结果不应为 nil")
	}

	if product.ASIN != "B08N5WRWNW" {
		t.Errorf("ASIN 不匹配: %s", product.ASIN)
	}

	if product.Title != "测试产品" {
		t.Errorf("标题不匹配: %s", product.Title)
	}

	if product.BuyBoxPrice != 1999 {
		t.Errorf("BuyBox 价格不匹配: %d", product.BuyBoxPrice)
	}

	if !product.IsFBA {
		t.Error("应该标记为 FBA")
	}

	if !product.IsPrime {
		t.Error("应该标记为 Prime")
	}

	if product.CurrentPrice != 1999 {
		t.Errorf("当前价格不匹配: %d", product.CurrentPrice)
	}

	if product.SalesRank != 5000 {
		t.Errorf("销售排名不匹配: %d", product.SalesRank)
	}

	if product.RawData == nil {
		t.Error("原始数据不应为 nil")
	}

	t.Log("产品转换测试通过")
}

// TestKeepaFetchProducts_GetCollectionName 测试获取集合名称
func TestKeepaFetchProducts_GetCollectionName(t *testing.T) {
	fetcher := &KeepaFetchProducts{}

	if fetcher.GetCollectionName() != model.CollectionKeepaProducts {
		t.Errorf("期望集合名称为 %s，实际为 %s",
			model.CollectionKeepaProducts, fetcher.GetCollectionName())
	}

	if fetcher.GetSourceCollectionName() != model.CollectionKeepaASINs {
		t.Errorf("期望源集合名称为 %s，实际为 %s",
			model.CollectionKeepaASINs, fetcher.GetSourceCollectionName())
	}
}

// TestKeepaFetchProducts_GetRawResponseCollectionName 测试获取原始响应集合名称
func TestKeepaFetchProducts_GetRawResponseCollectionName(t *testing.T) {
	fetcher := &KeepaFetchProducts{}
	collectionName := fetcher.GetRawResponseCollectionName()

	if collectionName != model.CollectionKeepaRawResponses {
		t.Errorf("期望集合名称为 %s，实际为 %s", model.CollectionKeepaRawResponses, collectionName)
	}
}

// TestKeepaFetchProducts_NoPendingASINs 测试没有待处理 ASIN 的情况
func TestKeepaFetchProducts_NoPendingASINs(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	// 创建一个简单的 API 客户端（不会被调用）
	client := api.NewClient(api.Config{
		AccessKey: "test-key",
		Timeout:   30 * time.Second,
		Logger:    logger,
	})

	// 创建任务实例
	fetcher := NewKeepaFetchProducts(client, repo, logger)

	// 执行测试（数据库为空）
	ctx := context.Background()
	err := fetcher.FetchProductDetails(ctx)

	if err != nil {
		t.Errorf("没有待处理 ASIN 时不应该返回错误: %v", err)
	}

	t.Log("无待处理 ASIN 测试通过")
}

// TestKeepaFetchProducts_RealAPI_GetPendingCount 测试获取待处理数量
func TestKeepaFetchProducts_RealAPI_GetPendingCount(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建真实的 API 客户端
	client := setupRealKeepaClient(t, logger)

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	// 创建任务实例
	fetcher := NewKeepaFetchProducts(client, repo, logger)

	ctx := context.Background()

	// 插入一些测试 ASIN
	testASINs := []*model.KeepaASIN{
		{ASIN: "B001", DomainID: 1, DetailFetched: false},
		{ASIN: "B002", DomainID: 1, DetailFetched: false},
		{ASIN: "B003", DomainID: 1, DetailFetched: true},
	}
	_ = repo.SaveASINs(ctx, testASINs)

	// 获取待处理数量
	count, err := fetcher.GetPendingCount(ctx)
	if err != nil {
		t.Fatalf("获取待处理数量失败: %v", err)
	}

	if count != 2 {
		t.Errorf("期望待处理数量为 2，实际为 %d", count)
	}

	t.Logf("待处理 ASIN 数量: %d", count)
}

// TestKeepaFetchProducts_SaveRawResponse 测试保存原始响应
func TestKeepaFetchProducts_SaveRawResponse(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 创建测试原始响应
	rawResponse := &model.KeepaRawResponse{
		APIEndpoint:  "/product",
		RequestType:  "product",
		DomainID:     1,
		RequestData:  map[string]interface{}{"asins": []string{"B08N5WRWNW"}, "stats": 90},
		RawResponse:  `{"products": [{"asin": "B08N5WRWNW", "title": "Test Product"}]}`,
		ResponseSize: 100,
		TokensUsed:   5,
		TokensLeft:   995,
	}

	// 保存原始响应
	err := repo.SaveRawResponse(ctx, rawResponse)
	if err != nil {
		t.Fatalf("保存原始响应失败: %v", err)
	}

	// 验证保存结果
	collection := db.Collection(model.CollectionKeepaRawResponses)
	var savedResponse model.KeepaRawResponse
	err = collection.FindOne(ctx, bson.M{"api_endpoint": "/product"}).Decode(&savedResponse)
	if err != nil {
		t.Fatalf("获取保存的原始响应失败: %v", err)
	}

	if savedResponse.APIEndpoint != "/product" {
		t.Errorf("期望 API 端点为 /product，实际为 %s", savedResponse.APIEndpoint)
	}

	if savedResponse.RequestType != "product" {
		t.Errorf("期望请求类型为 product，实际为 %s", savedResponse.RequestType)
	}

	if savedResponse.ResponseSize != 100 {
		t.Errorf("期望响应大小为 100，实际为 %d", savedResponse.ResponseSize)
	}

	if savedResponse.TokensUsed != 5 {
		t.Errorf("期望消耗 Token 为 5，实际为 %d", savedResponse.TokensUsed)
	}

	if savedResponse.CreatedAt.IsZero() {
		t.Error("创建时间不应为空")
	}

	t.Log("保存原始响应测试通过")
}
