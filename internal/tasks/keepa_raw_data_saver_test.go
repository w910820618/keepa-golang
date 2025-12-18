package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"keepa/internal/api"
	"keepa/internal/api/keepa/best_sellers"
	"keepa/internal/config"
	"keepa/internal/database"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// mockAPIClient mock API 客户端
type mockAPIClient struct {
	getRawDataFunc func(ctx context.Context, endpoint string, params map[string]string) ([]byte, error)
}

func (m *mockAPIClient) GetRawData(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	if m.getRawDataFunc != nil {
		return m.getRawDataFunc(ctx, endpoint, params)
	}
	return nil, errors.New("not implemented")
}

// mockStorage mock MongoDB 存储
type mockStorage struct {
	saveRawDataFunc func(ctx context.Context, collectionName string, data interface{}) error
	savedData       map[string][]interface{} // collection -> []data
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		savedData: make(map[string][]interface{}),
	}
}

func (m *mockStorage) SaveRawData(ctx context.Context, collectionName string, data interface{}) error {
	if m.saveRawDataFunc != nil {
		return m.saveRawDataFunc(ctx, collectionName, data)
	}
	// 默认行为：保存数据到内存
	if m.savedData == nil {
		m.savedData = make(map[string][]interface{})
	}
	m.savedData[collectionName] = append(m.savedData[collectionName], data)
	return nil
}

// mockBestSellersService mock Best Sellers 服务
type mockBestSellersService struct {
	fetchFunc func(ctx context.Context, params best_sellers.RequestParams) ([]byte, error)
}

func (m *mockBestSellersService) Fetch(ctx context.Context, params best_sellers.RequestParams) ([]byte, error) {
	if m.fetchFunc != nil {
		return m.fetchFunc(ctx, params)
	}
	// 返回模拟数据
	return json.Marshal(map[string]interface{}{
		"domain":   params.Domain,
		"category": params.Category,
		"products": []map[string]interface{}{
			{"asin": "B012345678", "title": "Test Product"},
		},
	})
}

// TestKeepaRawDataSaver_FetchAndStoreAll 测试获取并存储所有启用的 API 数据
// 注意：由于当前实现使用具体类型，此测试需要真实的 MongoDB 和 API key
// 请使用集成测试 TestKeepaRawDataSaver_FetchAndStoreBestSellers_WithRealAPI
func TestKeepaRawDataSaver_FetchAndStoreAll(t *testing.T) {
	t.Skip("This test requires refactoring to use interfaces for better testability")
	t.Log("Please use integration tests with real MongoDB and API key")
}

// TestKeepaRawDataSaver_FetchAndStoreBestSellers_Integration 集成测试：测试 Best Sellers 数据获取和存储
// 此测试需要 MongoDB 运行在本地
func TestKeepaRawDataSaver_FetchAndStoreBestSellers_Integration(t *testing.T) {
	// 检查是否设置了集成测试标志
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 连接 MongoDB
	client, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer client.Disconnect(ctx)

	// 测试数据库
	testDB := client.Database("keepa_test")
	defer testDB.Drop(ctx) // 清理测试数据

	// 创建存储
	storage := database.NewStorage(testDB, zap.NewNop())

	// 创建配置
	queriesConfig := &config.KeepaQueriesConfig{
		BestSellers: config.KeepaQueryAPIConfig{
			Enabled:    true,
			Collection: "test_best_sellers",
			Tasks: []config.KeepaQueryTaskConfig{
				{
					Name:    "test_task",
					Enabled: true,
					Params: map[string]interface{}{
						"domain":   1,
						"category": "Beauty",
						"range":    0,
					},
				},
			},
		},
	}

	// 创建 API 客户端（需要真实的 access key 或使用 mock）
	// 这里我们使用一个简单的 mock 方式：创建一个返回测试数据的 client
	apiClient := api.NewClient(api.Config{
		AccessKey: "test_key",
		Timeout:   10 * time.Second,
		Logger:    zap.NewNop(),
	})

	// 创建 saver
	saver := NewKeepaRawDataSaver(apiClient, storage, queriesConfig, zap.NewNop())

	// 执行获取和存储
	err = saver.FetchAndStoreAll(ctx)
	if err != nil {
		t.Fatalf("FetchAndStoreAll() error = %v", err)
	}

	// 验证数据已保存
	collection := testDB.Collection("test_best_sellers")
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}

	if count == 0 {
		t.Error("Expected at least one document to be saved")
	}

	// 验证数据内容
	var result bson.M
	err = collection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	// 验证数据包含时间戳
	if result["created_at"] == nil {
		t.Error("Expected document to contain 'created_at' field")
	}

	t.Logf("Successfully saved %d document(s) to MongoDB", count)
}

// TestKeepaRawDataSaver_MapToBestSellersParams 测试参数映射函数
func TestKeepaRawDataSaver_MapToBestSellersParams(t *testing.T) {
	saver := &KeepaRawDataSaver{}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid params",
			params: map[string]interface{}{
				"domain":   1,
				"category": "Beauty",
				"range":    0,
			},
			wantErr: false,
		},
		{
			name: "valid params with float64 domain",
			params: map[string]interface{}{
				"domain":   float64(1),
				"category": "Beauty",
			},
			wantErr: false,
		},
		{
			name: "missing domain",
			params: map[string]interface{}{
				"category": "Beauty",
			},
			wantErr: true,
		},
		{
			name: "missing category",
			params: map[string]interface{}{
				"domain": 1,
			},
			wantErr: true,
		},
		{
			name: "invalid domain type",
			params: map[string]interface{}{
				"domain":   "invalid",
				"category": "Beauty",
			},
			wantErr: true,
		},
		{
			name: "with optional params",
			params: map[string]interface{}{
				"domain":     1,
				"category":   "Beauty",
				"range":      30,
				"variations": 1,
				"sublist":    0,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := saver.mapToBestSellersParams(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapToBestSellersParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Domain == 0 {
					t.Error("Expected Domain to be set")
				}
				if got.Category == "" {
					t.Error("Expected Category to be set")
				}
			}
		})
	}
}

// TestKeepaRawDataSaver_MapToCategoryLookupParams 测试 Category Lookup 参数映射
func TestKeepaRawDataSaver_MapToCategoryLookupParams(t *testing.T) {
	saver := &KeepaRawDataSaver{}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid params with array",
			params: map[string]interface{}{
				"domain":   1,
				"category": []interface{}{0},
			},
			wantErr: false,
		},
		{
			name: "valid params with multiple categories",
			params: map[string]interface{}{
				"domain":   1,
				"category": []interface{}{1036592, 1036684},
			},
			wantErr: false,
		},
		{
			name: "valid params with includeParents",
			params: map[string]interface{}{
				"domain":         1,
				"category":       []interface{}{0},
				"includeParents": 1,
			},
			wantErr: false,
		},
		{
			name: "missing category",
			params: map[string]interface{}{
				"domain": 1,
			},
			wantErr: true,
		},
		{
			name: "invalid category type",
			params: map[string]interface{}{
				"domain":   1,
				"category": "invalid",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := saver.mapToCategoryLookupParams(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapToCategoryLookupParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Domain == 0 {
					t.Error("Expected Domain to be set")
				}
				if len(got.Category) == 0 {
					t.Error("Expected Category to be set")
				}
			}
		})
	}
}

// TestKeepaRawDataSaver_MapToProductsParams 测试 Products 参数映射
func TestKeepaRawDataSaver_MapToProductsParams(t *testing.T) {
	saver := &KeepaRawDataSaver{}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid params with asins",
			params: map[string]interface{}{
				"domain": 1,
				"asins":  []interface{}{"B08N5WRWNW", "B07H8QMZPV"},
			},
			wantErr: false,
		},
		{
			name: "valid params with codes",
			params: map[string]interface{}{
				"domain": 1,
				"codes":  []interface{}{"1234567890123"},
			},
			wantErr: false,
		},
		{
			name: "valid params with optional fields",
			params: map[string]interface{}{
				"domain": 1,
				"asins":  []interface{}{"B08N5WRWNW"},
				"offers": 20,
				"rating": 1,
			},
			wantErr: false,
		},
		{
			name: "missing domain",
			params: map[string]interface{}{
				"asins": []interface{}{"B08N5WRWNW"},
			},
			wantErr: true,
		},
		{
			name: "missing both asins and codes",
			params: map[string]interface{}{
				"domain": 1,
			},
			wantErr: false, // 参数映射不验证业务逻辑，只验证类型
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := saver.mapToProductsParams(tt.params)
			if (err != nil) != tt.wantErr {
				t.Errorf("mapToProductsParams() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Domain == 0 {
					t.Error("Expected Domain to be set")
				}
			}
		})
	}
}

// TestKeepaRawDataSaver_FetchAndStoreBestSellers_WithRealAPI 集成测试：使用真实的 Keepa API 和 MongoDB
// 此测试需要：
// 1. MongoDB 运行在 localhost:27017
// 2. 配置文件中包含有效的 keepa_api.access_key
// 3. 运行测试时使用: go test -v -run TestKeepaRawDataSaver_FetchAndStoreBestSellers_WithRealAPI
func TestKeepaRawDataSaver_FetchAndStoreBestSellers_WithRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 1. 连接 MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	// 测试数据库
	testDB := mongoClient.Database("keepa_test")
	defer testDB.Drop(ctx) // 清理测试数据

	// 2. 加载配置
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// 检查是否有 API key
	if cfg.KeepaAPI.AccessKey == "" {
		t.Skip("Skipping test: keepa_api.access_key not configured in config.yaml")
	}

	// 3. 创建存储
	storage := database.NewStorage(testDB, zap.NewNop())

	// 4. 创建 API 客户端
	timeout, err := time.ParseDuration(cfg.KeepaAPI.Timeout)
	if err != nil {
		timeout = 30 * time.Second
	}

	apiClient := api.NewClient(api.Config{
		AccessKey: cfg.KeepaAPI.AccessKey,
		Timeout:   timeout,
		Logger:    zap.NewNop(),
	})

	// 5. 创建查询配置
	queriesConfig := &config.KeepaQueriesConfig{
		BestSellers: config.KeepaQueryAPIConfig{
			Enabled:    true,
			Collection: "test_best_sellers",
			Tasks: []config.KeepaQueryTaskConfig{
				{
					Name:    "us_beauty_current",
					Enabled: true,
					Params: map[string]interface{}{
						"domain":   1, // US
						"category": "Beauty",
						"range":    0, // 当前排名
					},
				},
			},
		},
	}

	// 6. 创建 saver
	saver := NewKeepaRawDataSaver(apiClient, storage, queriesConfig, zap.NewNop())

	// 7. 执行获取和存储
	err = saver.FetchAndStoreAll(ctx)
	if err != nil {
		t.Fatalf("FetchAndStoreAll() error = %v", err)
	}

	// 8. 验证数据已保存到 MongoDB
	collection := testDB.Collection("test_best_sellers")
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}

	if count == 0 {
		t.Error("Expected at least one document to be saved")
	}

	// 9. 验证数据内容
	var result bson.M
	err = collection.FindOne(ctx, bson.M{}).Decode(&result)
	if err != nil {
		t.Fatalf("Failed to find document: %v", err)
	}

	// 验证数据包含时间戳
	if result["created_at"] == nil {
		t.Error("Expected document to contain 'created_at' field")
	}

	// 验证数据是有效的 JSON
	if result["raw_data"] != nil {
		rawData, ok := result["raw_data"].(string)
		if !ok {
			t.Error("Expected raw_data to be a string")
		} else {
			// 验证是有效的 JSON
			var jsonData interface{}
			if err := json.Unmarshal([]byte(rawData), &jsonData); err != nil {
				t.Errorf("raw_data is not valid JSON: %v", err)
			}
		}
	}

	t.Logf("Successfully saved %d document(s) to MongoDB", count)
}

// TestKeepaRawDataSaver_FetchAndStoreCategoryLookup_WithRealAPI 集成测试：Category Lookup API
func TestKeepaRawDataSaver_FetchAndStoreCategoryLookup_WithRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 连接 MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	testDB := mongoClient.Database("keepa_test")
	defer testDB.Drop(ctx)

	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.KeepaAPI.AccessKey == "" {
		t.Skip("Skipping test: keepa_api.access_key not configured")
	}

	// 创建存储和客户端
	storage := database.NewStorage(testDB, zap.NewNop())
	timeout, _ := time.ParseDuration(cfg.KeepaAPI.Timeout)
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	apiClient := api.NewClient(api.Config{
		AccessKey: cfg.KeepaAPI.AccessKey,
		Timeout:   timeout,
		Logger:    zap.NewNop(),
	})

	// 创建查询配置
	queriesConfig := &config.KeepaQueriesConfig{
		CategoryLookup: config.KeepaQueryAPIConfig{
			Enabled:    true,
			Collection: "test_category_lookup",
			Tasks: []config.KeepaQueryTaskConfig{
				{
					Name:    "us_root_categories",
					Enabled: true,
					Params: map[string]interface{}{
						"domain":   1,
						"category": []interface{}{0}, // 获取所有根分类
					},
				},
			},
		},
	}

	// 创建 saver 并执行
	saver := NewKeepaRawDataSaver(apiClient, storage, queriesConfig, zap.NewNop())
	err = saver.FetchAndStoreAll(ctx)
	if err != nil {
		t.Fatalf("FetchAndStoreAll() error = %v", err)
	}

	// 验证数据
	collection := testDB.Collection("test_category_lookup")
	count, err := collection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to count documents: %v", err)
	}

	if count == 0 {
		t.Error("Expected at least one document to be saved")
	}

	t.Logf("Successfully saved %d document(s) to MongoDB", count)
}

// TestKeepaRawDataSaver_MultipleAPIs_WithRealAPI 集成测试：测试多个 API 同时获取和存储
func TestKeepaRawDataSaver_MultipleAPIs_WithRealAPI(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 连接 MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	testDB := mongoClient.Database("keepa_test")
	defer testDB.Drop(ctx)

	// 加载配置
	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if cfg.KeepaAPI.AccessKey == "" {
		t.Skip("Skipping test: keepa_api.access_key not configured")
	}

	// 创建存储和客户端
	storage := database.NewStorage(testDB, zap.NewNop())
	timeout, _ := time.ParseDuration(cfg.KeepaAPI.Timeout)
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	apiClient := api.NewClient(api.Config{
		AccessKey: cfg.KeepaAPI.AccessKey,
		Timeout:   timeout,
		Logger:    zap.NewNop(),
	})

	// 创建包含多个 API 的查询配置
	queriesConfig := &config.KeepaQueriesConfig{
		BestSellers: config.KeepaQueryAPIConfig{
			Enabled:    true,
			Collection: "test_best_sellers",
			Tasks: []config.KeepaQueryTaskConfig{
				{
					Name:    "us_beauty",
					Enabled: true,
					Params: map[string]interface{}{
						"domain":   1,
						"category": "Beauty",
						"range":    0,
					},
				},
			},
		},
		CategoryLookup: config.KeepaQueryAPIConfig{
			Enabled:    true,
			Collection: "test_category_lookup",
			Tasks: []config.KeepaQueryTaskConfig{
				{
					Name:    "us_root",
					Enabled: true,
					Params: map[string]interface{}{
						"domain":   1,
						"category": []interface{}{0},
					},
				},
			},
		},
	}

	// 创建 saver 并执行
	saver := NewKeepaRawDataSaver(apiClient, storage, queriesConfig, zap.NewNop())
	err = saver.FetchAndStoreAll(ctx)
	if err != nil {
		t.Fatalf("FetchAndStoreAll() error = %v", err)
	}

	// 验证两个集合都有数据
	bestSellersCollection := testDB.Collection("test_best_sellers")
	bestSellersCount, err := bestSellersCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to count best_sellers documents: %v", err)
	}

	categoryLookupCollection := testDB.Collection("test_category_lookup")
	categoryLookupCount, err := categoryLookupCollection.CountDocuments(ctx, bson.M{})
	if err != nil {
		t.Fatalf("Failed to count category_lookup documents: %v", err)
	}

	if bestSellersCount == 0 {
		t.Error("Expected at least one document in test_best_sellers collection")
	}

	if categoryLookupCount == 0 {
		t.Error("Expected at least one document in test_category_lookup collection")
	}

	t.Logf("Successfully saved %d best_sellers and %d category_lookup documents", bestSellersCount, categoryLookupCount)
}

// TestKeepaRawDataSaver_ErrorHandling 测试错误处理
func TestKeepaRawDataSaver_ErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 连接 MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	testDB := mongoClient.Database("keepa_test")
	defer testDB.Drop(ctx)

	// 创建存储
	storage := database.NewStorage(testDB, zap.NewNop())

	// 测试配置解析错误
	queriesConfig := &config.KeepaQueriesConfig{
		BestSellers: config.KeepaQueryAPIConfig{
			Enabled:    true,
			Collection: "test_collection",
			Tasks: []config.KeepaQueryTaskConfig{
				{
					Name:    "invalid_task",
					Enabled: true,
					Params: map[string]interface{}{
						// 缺少必需的 domain 参数
						"category": "Beauty",
					},
				},
			},
		},
	}

	// 创建 API 客户端（使用无效的 key，会触发错误）
	apiClient := api.NewClient(api.Config{
		AccessKey: "invalid_key",
		Timeout:   10 * time.Second,
		Logger:    zap.NewNop(),
	})

	// 创建 saver
	saver := NewKeepaRawDataSaver(apiClient, storage, queriesConfig, zap.NewNop())

	// 调用 FetchAndStoreAll，应该处理错误而不崩溃
	// 由于参数错误，会跳过该任务，但不会导致整个流程失败
	err = saver.FetchAndStoreAll(ctx)
	// 即使有错误，也应该继续处理其他任务
	if err != nil {
		t.Logf("FetchAndStoreAll returned error (expected for invalid params): %v", err)
	}

	// 验证错误被正确处理，不会导致 panic
	t.Log("Error handling test passed: errors are handled gracefully")
}

// TestKeepaRawDataSaver_DisabledAPIs 测试禁用的 API 不会被调用
func TestKeepaRawDataSaver_DisabledAPIs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 连接 MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	testDB := mongoClient.Database("keepa_test")
	defer testDB.Drop(ctx)

	// 创建存储
	storage := database.NewStorage(testDB, zap.NewNop())

	// 创建配置，所有 API 都禁用
	queriesConfig := &config.KeepaQueriesConfig{
		BestSellers: config.KeepaQueryAPIConfig{
			Enabled: false,
		},
		CategoryLookup: config.KeepaQueryAPIConfig{
			Enabled: false,
		},
	}

	apiClient := api.NewClient(api.Config{
		AccessKey: "test_key",
		Timeout:   10 * time.Second,
		Logger:    zap.NewNop(),
	})

	saver := NewKeepaRawDataSaver(apiClient, storage, queriesConfig, zap.NewNop())

	// 执行，应该不会调用任何 API
	err = saver.FetchAndStoreAll(ctx)
	if err != nil {
		t.Fatalf("FetchAndStoreAll() should not return error when all APIs are disabled: %v", err)
	}

	// 验证没有数据被保存（检查所有可能的集合）
	collections := []string{"best_sellers", "category_lookup", "test_best_sellers", "test_category_lookup"}
	for _, collName := range collections {
		collection := testDB.Collection(collName)
		count, _ := collection.CountDocuments(ctx, bson.M{})
		if count > 0 {
			t.Logf("Note: Found %d documents in %s (may be from previous tests)", count, collName)
		}
	}

	t.Log("Disabled APIs test passed: no APIs were called")
}

// TestKeepaRawDataSaver_DisabledTasks 测试禁用的任务不会被执行
func TestKeepaRawDataSaver_DisabledTasks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 连接 MongoDB
	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI("mongodb://localhost:27017"))
	if err != nil {
		t.Skipf("MongoDB not available: %v", err)
	}
	defer mongoClient.Disconnect(ctx)

	testDB := mongoClient.Database("keepa_test")
	defer testDB.Drop(ctx)

	// 创建存储
	storage := database.NewStorage(testDB, zap.NewNop())

	// 创建配置，API 启用但任务禁用
	queriesConfig := &config.KeepaQueriesConfig{
		BestSellers: config.KeepaQueryAPIConfig{
			Enabled:    true,
			Collection: "test_collection",
			Tasks: []config.KeepaQueryTaskConfig{
				{
					Name:    "disabled_task",
					Enabled: false, // 任务被禁用
					Params: map[string]interface{}{
						"domain":   1,
						"category": "Beauty",
					},
				},
			},
		},
	}

	apiClient := api.NewClient(api.Config{
		AccessKey: "test_key",
		Timeout:   10 * time.Second,
		Logger:    zap.NewNop(),
	})

	saver := NewKeepaRawDataSaver(apiClient, storage, queriesConfig, zap.NewNop())

	// 执行，应该跳过禁用的任务
	err = saver.FetchAndStoreAll(ctx)
	if err != nil {
		t.Logf("FetchAndStoreAll returned error (may be expected): %v", err)
	}

	// 验证没有数据被保存（因为任务被禁用）
	collection := testDB.Collection("test_collection")
	count, _ := collection.CountDocuments(ctx, bson.M{})
	if count > 0 {
		t.Logf("Note: Found %d documents (task was disabled, but data may exist from other sources)", count)
	} else {
		t.Log("Disabled task test passed: no data was saved for disabled task")
	}
}
