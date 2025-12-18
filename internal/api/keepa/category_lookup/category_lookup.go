package category_lookup

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Category Lookup 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Category Lookup 服务
func NewService(client *api.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// RequestParams 请求参数
type RequestParams struct {
	// 必需参数
	Domain   int   `json:"domain"`   // Amazon 域名代码 (1=com, 2=co.uk, 3=de, 4=fr, 5=co.jp, 6=ca, 8=it, 9=es, 10=in, 11=com.mx)
	Category []int `json:"category"` // 类别节点 ID 数组（最多10个）。或者使用单个值 0 来获取所有根分类

	// 可选参数
	IncludeParents *int `json:"includeParents,omitempty"` // 是否包含父分类树: 0=不包含, 1=包含
}

// Validate 验证请求参数
func (p *RequestParams) Validate() error {
	// 验证 domain (有效值: 1,2,3,4,5,6,8,9,10,11)
	validDomains := map[int]bool{
		1:  true, // com
		2:  true, // co.uk
		3:  true, // de
		4:  true, // fr
		5:  true, // co.jp
		6:  true, // ca
		8:  true, // it
		9:  true, // es
		10: true, // in
		11: true, // com.mx
	}
	if !validDomains[p.Domain] {
		return fmt.Errorf("invalid domain: %d (valid domains: 1,2,3,4,5,6,8,9,10,11)", p.Domain)
	}

	// 验证 category 必须存在
	if len(p.Category) == 0 {
		return fmt.Errorf("category is required (use [0] to get all root categories)")
	}

	// 验证 category 数量（最多10个）
	if len(p.Category) > 10 {
		return fmt.Errorf("category can contain at most 10 IDs, got %d", len(p.Category))
	}

	// 检查是否包含 0（用于获取所有根分类）
	hasZero := false
	for _, catID := range p.Category {
		if catID == 0 {
			hasZero = true
			break
		}
	}

	// 如果包含 0，必须是唯一的（不能与其他 ID 混合）
	if hasZero {
		if len(p.Category) != 1 {
			return fmt.Errorf("category ID 0 (root categories) cannot be used with other category IDs")
		}
		// 使用 0 获取所有根分类，这是合法的
	} else {
		// 验证所有 category ID 都是正整数
		for _, catID := range p.Category {
			if catID <= 0 {
				return fmt.Errorf("category ID must be positive (or 0 for root categories), got %d", catID)
			}
		}
	}

	// 验证 includeParents 值（如果提供）
	if p.IncludeParents != nil {
		if *p.IncludeParents != 0 && *p.IncludeParents != 1 {
			return fmt.Errorf("includeParents must be 0 or 1, got %d", *p.IncludeParents)
		}
	}

	return nil
}

// ToQueryParams 将 RequestParams 转换为查询参数字典
func (p *RequestParams) ToQueryParams() map[string]string {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)

	// 将 category 数组转换为逗号分隔的字符串
	categoryStrs := make([]string, len(p.Category))
	for i, catID := range p.Category {
		categoryStrs[i] = strconv.Itoa(catID)
	}
	params["category"] = strings.Join(categoryStrs, ",")

	// 可选参数
	if p.IncludeParents != nil {
		params["parents"] = strconv.Itoa(*p.IncludeParents)
	}

	return params
}

// Fetch 获取数据
func (s *Service) Fetch(ctx context.Context, params RequestParams) ([]byte, error) {
	// 验证参数
	if err := params.Validate(); err != nil {
		if s.logger != nil {
			s.logger.Error("invalid request parameters", zap.Error(err))
		}
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 记录请求日志
	if s.logger != nil {
		logFields := []zap.Field{
			zap.Int("domain", params.Domain),
			zap.Ints("category", params.Category),
		}
		if params.IncludeParents != nil {
			logFields = append(logFields, zap.Int("includeParents", *params.IncludeParents))
		}

		s.logger.Info("fetching category lookup data", logFields...)
	}

	// 构建 API 请求参数
	endpoint := "/category"
	requestParams := params.ToQueryParams()

	// 调用 API 获取原始数据
	rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch category lookup data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("category lookup data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
