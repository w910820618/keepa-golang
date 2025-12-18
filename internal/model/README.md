# Model 包

这个包包含了 Keepa API 返回的数据结构模型。

## 文件说明

### category.go

定义了 Keepa API Category Object 的完整结构体，包含所有字段：

- **基础信息**: `DomainID`, `CatID`, `Name`, `ContextFreeName`, `WebsiteDisplayGroup`
- **层级关系**: `Children`, `Parent`
- **分类属性**: `IsBrowseNode`
- **排名和产品统计**: `HighestRank`, `LowestRank`, `ProductCount`
- **价格统计**: `AvgBuyBox`, `AvgBuyBox90`, `AvgBuyBox365`, `AvgBuyBoxDeviation`
- **评价统计**: `AvgReviewCount`, `AvgRating`
- **百分比统计**: `IsFBAPercent`, `SoldByAmazonPercent`, `HasCouponPercent`
- **报价统计**: `AvgOfferCountNew`, `AvgOfferCountUsed`
- **卖家统计**: `SellerCount`, `BrandCount`
- **价格变化**: `AvgDeltaPercent30BuyBox`, `AvgDeltaPercent90BuyBox`, `AvgDeltaPercent30Amazon`, `AvgDeltaPercent90Amazon`
- **关联信息**: `RelatedCategories`, `TopBrands`

### CategoryTree

分类树结构，用于存储递归的分类层级关系：

```go
type CategoryTree struct {
    Category  *Category       // 分类信息
    Children  []*CategoryTree  // 子分类树
    CreatedAt int64           // 创建时间戳
}
```

## 使用示例

```go
import "keepa/internal/model"

// 解析 API 响应
var category model.Category
json.Unmarshal(apiResponse, &category)

// 检查是否为根分类
if category.IsRoot() {
    // 处理根分类
}

// 检查是否有子分类
if category.HasChildren() {
    // 处理子分类
    for _, childID := range category.Children {
        // 递归处理子分类
    }
}
```

## 参考文档

- [Keepa API Category Object 文档](https://keepa.com/#!discuss/topic/category-object)

