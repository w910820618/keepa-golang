# KeepaRawDataSaver 单元测试说明

本文档说明如何运行和使用 `keepa_raw_data_saver_test.go` 中的测试函数。

## 测试分类

### 1. 单元测试（无需外部依赖）

这些测试只测试参数映射和配置解析，不需要 MongoDB 或 Keepa API：

- `TestKeepaRawDataSaver_MapToBestSellersParams` - 测试 Best Sellers 参数映射
- `TestKeepaRawDataSaver_MapToCategoryLookupParams` - 测试 Category Lookup 参数映射
- `TestKeepaRawDataSaver_MapToProductsParams` - 测试 Products 参数映射

**运行方式：**
```bash
go test ./internal/tasks -v -run TestKeepaRawDataSaver_MapTo
```

### 2. 集成测试（需要 MongoDB）

这些测试需要 MongoDB 运行在 `localhost:27017`，用于验证数据存储功能：

- `TestKeepaRawDataSaver_FetchAndStoreBestSellers_WithRealAPI` - 使用真实 Keepa API 和 MongoDB 测试 Best Sellers
- `TestKeepaRawDataSaver_FetchAndStoreCategoryLookup_WithRealAPI` - 测试 Category Lookup API
- `TestKeepaRawDataSaver_MultipleAPIs_WithRealAPI` - 测试多个 API 同时获取和存储
- `TestKeepaRawDataSaver_ErrorHandling` - 测试错误处理
- `TestKeepaRawDataSaver_DisabledAPIs` - 测试禁用的 API 不会被调用
- `TestKeepaRawDataSaver_DisabledTasks` - 测试禁用的任务不会被执行

**运行方式：**

1. 确保 MongoDB 运行在本地：
```bash
# 检查 MongoDB 是否运行
mongosh --eval "db.version()"
```

2. 确保配置文件中有有效的 Keepa API key：
```yaml
# configs/config.yaml
keepa_api:
  access_key: "your-valid-api-key"
  timeout: 30s
```

3. 运行集成测试：
```bash
# 运行所有集成测试
go test ./internal/tasks -v -run TestKeepaRawDataSaver_.*_WithRealAPI

# 运行特定测试
go test ./internal/tasks -v -run TestKeepaRawDataSaver_FetchAndStoreBestSellers_WithRealAPI

# 跳过集成测试（只运行单元测试）
go test ./internal/tasks -short
```

## 测试验证内容

### 数据获取验证
- ✅ API 调用成功
- ✅ 返回的数据格式正确
- ✅ 错误处理正确

### 数据存储验证
- ✅ 数据成功保存到 MongoDB
- ✅ 集合名称正确
- ✅ 数据包含时间戳（`created_at`）
- ✅ 数据格式有效（JSON）

### 配置验证
- ✅ 启用的 API 被调用
- ✅ 禁用的 API 不被调用
- ✅ 启用的任务被执行
- ✅ 禁用的任务被跳过
- ✅ 参数映射正确

## 测试数据库

所有集成测试使用 `keepa_test` 数据库，测试结束后会自动清理（`defer testDB.Drop(ctx)`）。

**注意：** 如果测试中断，可能需要手动清理测试数据库：
```bash
mongosh keepa_test --eval "db.dropDatabase()"
```

## 测试示例输出

成功运行的测试输出示例：

```
=== RUN   TestKeepaRawDataSaver_FetchAndStoreBestSellers_WithRealAPI
    keepa_raw_data_saver_test.go:XXX: Successfully saved 1 document(s) to MongoDB
--- PASS: TestKeepaRawDataSaver_FetchAndStoreBestSellers_WithRealAPI (2.34s)
PASS
ok      keepa/internal/tasks    2.345s
```

## 故障排除

### MongoDB 连接失败
```
MongoDB not available: connection refused
```
**解决方案：** 确保 MongoDB 服务正在运行

### API Key 未配置
```
Skipping test: keepa_api.access_key not configured in config.yaml
```
**解决方案：** 在 `configs/config.yaml` 中配置有效的 API key

### 测试超时
```
context deadline exceeded
```
**解决方案：** 增加测试超时时间或检查网络连接

## 注意事项

1. **API 配额：** 集成测试会消耗 Keepa API 的 token 配额，请谨慎使用
2. **测试数据：** 测试使用真实的 API，会产生真实的 API 调用
3. **MongoDB 清理：** 测试会自动清理测试数据库，但建议在测试前确认没有重要数据
4. **并发测试：** 多个测试同时运行可能会产生冲突，建议串行运行

## 最佳实践

1. 开发时使用单元测试快速验证逻辑
2. 提交代码前运行所有测试确保功能正常
3. 定期运行集成测试验证与真实 API 和数据库的集成
4. 使用 `-short` 标志跳过集成测试以加快开发速度

