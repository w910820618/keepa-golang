package tasks

import (
	"context"
	"os"
	"testing"
	"time"

	"keepa/internal/api"
	"keepa/internal/config"
	"keepa/internal/database"
	"keepa/internal/model"
	"keepa/internal/repository"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// getKeepaAPIKey 获取 Keepa API Key
// 优先从环境变量获取，其次从配置文件获取
func getKeepaAPIKey(t *testing.T) string {
	// 1. 从环境变量获取
	apiKey := os.Getenv("KEEPA_API_KEY")
	if apiKey != "" {
		t.Logf("从环境变量获取 API Key")
		return apiKey
	}

	// 2. 从配置文件获取（尝试多个路径）
	configPaths := []string{"", "configs", "../../configs", "../../../configs"}
	for _, path := range configPaths {
		cfg, err := config.Load(path)
		if err == nil && cfg.KeepaAPI.AccessKey != "" {
			t.Logf("从配置文件获取 API Key (路径: %s)", path)
			return cfg.KeepaAPI.AccessKey
		}
	}

	return ""
}

// setupTestMongoDB 创建测试用的 MongoDB 连接
func setupTestMongoDB(t *testing.T) (*mongo.Database, func()) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 尝试从配置文件获取 MongoDB URI
	mongoURI := "mongodb://localhost:27017"
	cfg, err := config.Load("")
	if err == nil && cfg.Database.MongoDB.URI != "" {
		mongoURI = cfg.Database.MongoDB.URI
	}

	// 连接到 MongoDB
	clientOpts := options.Client().ApplyURI(mongoURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		t.Skipf("跳过测试：无法连接到 MongoDB: %v", err)
	}

	// 测试连接
	if err := client.Ping(ctx, nil); err != nil {
		t.Skipf("跳过测试：MongoDB ping 失败: %v", err)
	}

	// 使用测试数据库
	db := client.Database("keepa_test")

	// 返回清理函数
	cleanup := func() {
		// 清理测试集合
		_ = db.Collection(model.CollectionKeepaASINs).Drop(context.Background())
		_ = db.Collection(model.CollectionKeepaProducts).Drop(context.Background())
		_ = db.Collection(model.CollectionKeepaCategories).Drop(context.Background())
		_ = db.Collection(model.CollectionKeepaRawResponses).Drop(context.Background())
		_ = client.Disconnect(context.Background())
	}

	return db, cleanup
}

// setupRealKeepaClient 创建连接真实 Keepa API 的客户端
func setupRealKeepaClient(t *testing.T, logger *zap.Logger) *api.Client {
	apiKey := getKeepaAPIKey(t)
	if apiKey == "" {
		t.Skip("跳过测试：未配置 KEEPA_API_KEY 环境变量或配置文件中的 access_key")
	}

	// 从配置文件获取其他配置
	cfg, _ := config.Load("")

	timeout := 60 * time.Second
	printCurl := false
	printResponse := false

	if cfg != nil {
		if t, err := time.ParseDuration(cfg.KeepaAPI.Timeout); err == nil {
			timeout = t
		}
		printCurl = cfg.KeepaAPI.PrintCurlCommand
		printResponse = cfg.KeepaAPI.PrintResponseBody
	}

	return api.NewClient(api.Config{
		AccessKey:         apiKey,
		Timeout:           timeout,
		Logger:            logger,
		PrintCurlCommand:  printCurl,
		PrintResponseBody: printResponse,
	})
}

// TestKeepaFetchAsins_RealAPI_FetchPetSuppliesAsins 测试真实 API 获取宠物用品 ASIN
func TestKeepaFetchAsins_RealAPI_FetchPetSuppliesAsins(t *testing.T) {
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
	fetcher := NewKeepaFetchAsins(client, repo, logger)

	// 执行测试
	ctx := context.Background()
	asins, err := fetcher.FetchPetSuppliesAsins(ctx)

	// 验证结果
	if err != nil {
		t.Fatalf("FetchPetSuppliesAsins 失败: %v", err)
	}

	t.Logf("成功从真实 API 获取 %d 个 ASIN", len(asins))

	if len(asins) == 0 {
		t.Log("警告：没有获取到 ASIN，可能是查询条件过于严格")
	} else {
		// 打印所有 ASIN
		t.Log("========== 获取到的所有 ASIN ==========")
		for i, asin := range asins {
			t.Logf("[%3d] %s", i+1, asin)
		}
		t.Log("========================================")
	}

	// 验证数据是否保存到 MongoDB
	pendingASINs, err := repo.GetPendingASINs(ctx, 10)
	if err != nil {
		t.Errorf("获取待处理 ASIN 失败: %v", err)
	}

	t.Logf("MongoDB 中保存了 %d 个待处理 ASIN", len(pendingASINs))

	// 验证原始响应是否保存到 MongoDB
	rawCollection := db.Collection(model.CollectionKeepaRawResponses)
	rawCount, err := rawCollection.CountDocuments(ctx, bson.M{"api_endpoint": "/query"})
	if err != nil {
		t.Errorf("查询原始响应失败: %v", err)
	}

	if rawCount > 0 {
		t.Logf("MongoDB 中保存了 %d 条原始 API 响应", rawCount)

		// 查询并显示原始响应的基本信息
		var rawResponse model.KeepaRawResponse
		err = rawCollection.FindOne(ctx, bson.M{"api_endpoint": "/query"}).Decode(&rawResponse)
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
}

// TestKeepaFetchAsins_RealAPI_CustomQuery 测试真实 API 使用自定义查询
func TestKeepaFetchAsins_RealAPI_CustomQuery(t *testing.T) {
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
	fetcher := NewKeepaFetchAsins(client, repo, logger)

	// 构建自定义查询参数（更宽松的条件）
	salesRankMin := 1000
	salesRankMax := 50000
	perPage := 50 // 只获取 50 个以节省 token

	query := &model.ProductFinderQuery{
		RootCategory: 2619533011, // US Pet Supplies
		SalesRankGTE: &salesRankMin,
		SalesRankLTE: &salesRankMax,
		PerPage:      &perPage,
	}

	// 执行测试
	ctx := context.Background()
	asins, err := fetcher.FetchAsinsWithQuery(ctx, 1, query)

	// 验证结果
	if err != nil {
		t.Fatalf("FetchAsinsWithQuery 失败: %v", err)
	}

	t.Logf("成功获取 %d 个 ASIN", len(asins))

	// 验证 ASIN 格式（应该是 10 位字母数字组合）
	for i, asin := range asins {
		if len(asin) != 10 {
			t.Errorf("ASIN[%d] 格式不正确: %s (长度: %d)", i, asin, len(asin))
		}
	}

	// 验证原始响应是否保存
	rawCollection := db.Collection(model.CollectionKeepaRawResponses)
	rawCount, _ := rawCollection.CountDocuments(ctx, bson.M{"api_endpoint": "/query"})
	t.Logf("保存了 %d 条原始 API 响应", rawCount)
}

// TestKeepaFetchAsins_SaveASINs 测试保存 ASIN 到 MongoDB
func TestKeepaFetchAsins_SaveASINs(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 准备测试数据
	testASINs := []*model.KeepaASIN{
		{
			ASIN:        "B08N5WRWNW",
			DomainID:    1,
			CategoryID:  2619533011,
			QuerySource: "product_finder",
		},
		{
			ASIN:        "B09ABC1234",
			DomainID:    1,
			CategoryID:  2619533011,
			QuerySource: "product_finder",
		},
	}

	// 保存 ASIN
	err := repo.SaveASINs(ctx, testASINs)
	if err != nil {
		t.Fatalf("保存 ASIN 失败: %v", err)
	}

	// 验证保存结果
	savedASIN, err := repo.GetASINByCode(ctx, "B08N5WRWNW", 1)
	if err != nil {
		t.Fatalf("获取 ASIN 失败: %v", err)
	}

	if savedASIN == nil {
		t.Fatal("ASIN 未保存到数据库")
	}

	if savedASIN.ASIN != "B08N5WRWNW" {
		t.Errorf("期望 ASIN 为 B08N5WRWNW，实际为 %s", savedASIN.ASIN)
	}

	if savedASIN.DetailFetched {
		t.Error("新保存的 ASIN 应该标记为未获取详情")
	}

	t.Logf("成功保存 ASIN: %s", savedASIN.ASIN)
}

// TestKeepaFetchAsins_DuplicateASINs 测试重复 ASIN 的 upsert 行为
func TestKeepaFetchAsins_DuplicateASINs(t *testing.T) {
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
	testASINs := []*model.KeepaASIN{
		{
			ASIN:        "B08N5WRWNW",
			DomainID:    1,
			CategoryID:  2619533011,
			QuerySource: "product_finder",
		},
	}

	err := repo.SaveASINs(ctx, testASINs)
	if err != nil {
		t.Fatalf("第一次保存 ASIN 失败: %v", err)
	}

	// 第二次保存（相同 ASIN，不同来源）
	testASINs[0].QuerySource = "best_sellers"
	err = repo.SaveASINs(ctx, testASINs)
	if err != nil {
		t.Fatalf("第二次保存 ASIN 失败: %v", err)
	}

	// 验证数据库中只有一条记录
	collection := db.Collection(model.CollectionKeepaASINs)
	count, err := collection.CountDocuments(ctx, bson.M{"asin": "B08N5WRWNW"})
	if err != nil {
		t.Fatalf("统计 ASIN 数量失败: %v", err)
	}

	if count != 1 {
		t.Errorf("期望数据库中只有 1 条记录，实际有 %d 条", count)
	}

	// 验证 query_source 被更新
	savedASIN, _ := repo.GetASINByCode(ctx, "B08N5WRWNW", 1)
	if savedASIN.QuerySource != "best_sellers" {
		t.Errorf("期望 query_source 为 best_sellers，实际为 %s", savedASIN.QuerySource)
	}

	t.Log("重复 ASIN upsert 测试通过")
}

// TestKeepaFetchAsins_EmptyASINList 测试空 ASIN 列表
func TestKeepaFetchAsins_EmptyASINList(t *testing.T) {
	// 设置 MongoDB
	db, cleanup := setupTestMongoDB(t)
	defer cleanup()

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 保存空列表
	err := repo.SaveASINs(ctx, []*model.KeepaASIN{})
	if err != nil {
		t.Errorf("保存空 ASIN 列表应该成功，但返回错误: %v", err)
	}

	// 保存 nil
	err = repo.SaveASINs(ctx, nil)
	if err != nil {
		t.Errorf("保存 nil ASIN 列表应该成功，但返回错误: %v", err)
	}

	t.Log("空 ASIN 列表测试通过")
}

// TestKeepaFetchAsins_GetCollectionName 测试获取集合名称
func TestKeepaFetchAsins_GetCollectionName(t *testing.T) {
	fetcher := &KeepaFetchAsins{}
	collectionName := fetcher.GetCollectionName()

	if collectionName != model.CollectionKeepaASINs {
		t.Errorf("期望集合名称为 %s，实际为 %s", model.CollectionKeepaASINs, collectionName)
	}
}

// TestKeepaFetchAsins_GetRawResponseCollectionName 测试获取原始响应集合名称
func TestKeepaFetchAsins_GetRawResponseCollectionName(t *testing.T) {
	fetcher := &KeepaFetchAsins{}
	collectionName := fetcher.GetRawResponseCollectionName()

	if collectionName != model.CollectionKeepaRawResponses {
		t.Errorf("期望集合名称为 %s，实际为 %s", model.CollectionKeepaRawResponses, collectionName)
	}
}

// TestKeepaFetchAsins_SaveRawResponse 测试保存原始响应
func TestKeepaFetchAsins_SaveRawResponse(t *testing.T) {
	// 设置 MongoDB
	db, _ := setupTestMongoDB(t)
	// defer cleanup()

	// 打印数据库信息
	t.Logf("========== 调试信息 ==========")
	t.Logf("数据库名称: %s", db.Name())
	t.Logf("集合名称: %s", model.CollectionKeepaRawResponses)

	// 创建 logger
	logger, _ := zap.NewDevelopment()

	// 创建 Storage 和 Repository
	storage := database.NewStorage(db, logger)
	repo := repository.NewKeepaRepository(storage, logger)

	ctx := context.Background()

	// 查询保存前的文档数量
	collection := db.Collection(model.CollectionKeepaRawResponses)
	countBefore, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Logf("查询保存前文档数量失败: %v", err)
	} else {
		t.Logf("保存前文档数量: %d", countBefore)
	}

	// 创建测试原始响应
	rawResponse := &model.KeepaRawResponse{
		APIEndpoint:  "/query",
		RequestType:  "product_finder",
		DomainID:     1,
		RequestData:  map[string]interface{}{"rootCategory": 2619533011},
		RawResponse:  `{"totalResults": 100, "asinList": ["B08N5WRWNW"]}`,
		ResponseSize: 50,
		TokensUsed:   10,
		TokensLeft:   990,
	}

	t.Logf("准备保存的数据: APIEndpoint=%s, RequestType=%s, DomainID=%d",
		rawResponse.APIEndpoint, rawResponse.RequestType, rawResponse.DomainID)

	// 保存原始响应
	err = repo.SaveRawResponse(ctx, rawResponse)
	if err != nil {
		t.Fatalf("保存原始响应失败: %v", err)
	}
	t.Logf("SaveRawResponse 调用成功，无错误返回")

	// 查询保存后的文档数量
	countAfter, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Logf("查询保存后文档数量失败: %v", err)
	} else {
		t.Logf("保存后文档数量: %d", countAfter)
	}

	// 列出集合中所有文档
	cursor, err := collection.Find(ctx, bson.M{})
	if err != nil {
		t.Logf("查询所有文档失败: %v", err)
	} else {
		var allDocs []bson.M
		if err := cursor.All(ctx, &allDocs); err != nil {
			t.Logf("解析文档失败: %v", err)
		} else {
			t.Logf("集合中的所有文档 (%d 条):", len(allDocs))
			for i, doc := range allDocs {
				t.Logf("  文档[%d]: _id=%v, api_endpoint=%v, request_type=%v",
					i, doc["_id"], doc["api_endpoint"], doc["request_type"])
			}
		}
	}

	// 验证保存结果
	var savedResponse model.KeepaRawResponse
	err = collection.FindOne(ctx, bson.M{"api_endpoint": "/query"}).Decode(&savedResponse)
	if err != nil {
		t.Logf("FindOne 查询条件: api_endpoint=/query")
		t.Fatalf("获取保存的原始响应失败: %v", err)
	}

	t.Logf("成功获取到保存的文档: ID=%v", savedResponse.ID)

	if savedResponse.APIEndpoint != "/query" {
		t.Errorf("期望 API 端点为 /query，实际为 %s", savedResponse.APIEndpoint)
	}

	if savedResponse.RequestType != "product_finder" {
		t.Errorf("期望请求类型为 product_finder，实际为 %s", savedResponse.RequestType)
	}

	if savedResponse.ResponseSize != 50 {
		t.Errorf("期望响应大小为 50，实际为 %d", savedResponse.ResponseSize)
	}

	if savedResponse.TokensUsed != 10 {
		t.Errorf("期望消耗 Token 为 10，实际为 %d", savedResponse.TokensUsed)
	}

	if savedResponse.CreatedAt.IsZero() {
		t.Error("创建时间不应为空")
	}

	t.Logf("========== 调试信息结束 ==========")
	t.Log("保存原始响应测试通过")
}
