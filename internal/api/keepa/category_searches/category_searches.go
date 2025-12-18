package category_searches

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Category Searches 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Category Searches 服务
func NewService(client *api.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// RequestParams 请求参数
type RequestParams struct {
	// 必需参数
	Domain int    `json:"domain"` // Amazon 域名代码 (1=com, 2=co.uk, 3=de, 4=fr, 5=co.jp, 6=ca, 8=it, 9=es, 10=in, 11=com.mx)
	Term   string `json:"term"`   // 搜索关键词，需要 URL 编码。多个空格分隔的关键词是可能的，所有提供的关键词必须匹配。关键词的最小长度是 3 个字符
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

	// 验证 term 不能为空
	if p.Term == "" {
		return fmt.Errorf("term is required")
	}

	// 验证 term 中的每个关键词长度至少为 3 个字符
	// 将 term 按空格分割成关键词
	keywords := strings.Fields(strings.TrimSpace(p.Term))
	if len(keywords) == 0 {
		return fmt.Errorf("term cannot be empty or only whitespace")
	}

	// 检查每个关键词的长度
	for i, keyword := range keywords {
		if len(keyword) < 3 {
			return fmt.Errorf("keyword %d in term must be at least 3 characters long, got: %q (length: %d)", i+1, keyword, len(keyword))
		}
	}

	return nil
}

// ToQueryParams 将 RequestParams 转换为查询参数字典
func (p *RequestParams) ToQueryParams() map[string]string {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)
	params["type"] = "category" // 固定值
	params["term"] = p.Term     // term 会在 client 中自动进行 URL 编码

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
		s.logger.Info("fetching category searches data",
			zap.Int("domain", params.Domain),
			zap.String("term", params.Term),
		)
	}

	// 构建 API 请求参数
	endpoint := "/search"
	requestParams := params.ToQueryParams()

	// 调用 API 获取原始数据
	rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch category searches data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("category searches data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
