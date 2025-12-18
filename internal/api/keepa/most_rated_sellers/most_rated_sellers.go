package most_rated_sellers

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Most Rated Sellers 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Most Rated Sellers 服务
func NewService(client *api.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// RequestParams 请求参数
type RequestParams struct {
	// 必需参数
	Domain int `json:"domain"` // Amazon 域名代码 (1=com, 2=co.uk, 3=de, 4=fr, 5=co.jp, 6=ca, 8=it, 9=es, 10=in, 11=com.mx)
	// 注意：不支持 Amazon Brazil (domain=7)
}

// Validate 验证请求参数
func (p *RequestParams) Validate() error {
	// 验证 domain (有效值: 1,2,3,4,5,6,8,9,10,11，不包括7/Brazil)
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
		return fmt.Errorf("invalid domain: %d (valid domains: 1,2,3,4,5,6,8,9,10,11; domain 7/Brazil is not supported)", p.Domain)
	}

	return nil
}

// ToQueryParams 将 RequestParams 转换为查询参数字典
func (p *RequestParams) ToQueryParams() map[string]string {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)

	return params
}

// Response API 响应结构
type Response struct {
	SellerIDList []string `json:"sellerIdList"` // 有序的卖家ID列表，从评分最多的卖家开始
}

// Fetch 获取 Most Rated Sellers 数据并返回解析后的响应
// Token Cost: 50
func (s *Service) Fetch(ctx context.Context, params RequestParams) (*Response, error) {
	// 验证参数
	if err := params.Validate(); err != nil {
		if s.logger != nil {
			s.logger.Error("invalid request parameters", zap.Error(err))
		}
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 记录请求日志
	if s.logger != nil {
		s.logger.Info("fetching most rated sellers data",
			zap.Int("domain", params.Domain),
		)
	}

	// 构建 API 请求参数
	endpoint := "/topseller"
	requestParams := params.ToQueryParams()

	// 调用 API 获取原始数据
	rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch most rated sellers data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	// 解析 JSON 响应
	var response Response
	if err := json.Unmarshal(rawData, &response); err != nil {
		if s.logger != nil {
			s.logger.Error("failed to parse most rated sellers response",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("most rated sellers data fetched successfully",
			zap.Int("seller_count", len(response.SellerIDList)),
		)
	}

	return &response, nil
}

// FetchRaw 获取原始数据
// Token Cost: 50
// 注意：列表按评分最多的卖家排序，每天更新，最多包含 100,000 个卖家ID
// 不支持 Amazon Brazil (domain=7)
func (s *Service) FetchRaw(ctx context.Context, params RequestParams) ([]byte, error) {
	// 验证参数
	if err := params.Validate(); err != nil {
		if s.logger != nil {
			s.logger.Error("invalid request parameters", zap.Error(err))
		}
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// 记录请求日志
	if s.logger != nil {
		s.logger.Info("fetching most rated sellers data",
			zap.Int("domain", params.Domain),
		)
	}

	// 构建 API 请求参数
	endpoint := "/topseller"
	requestParams := params.ToQueryParams()

	// 调用 API 获取原始数据
	rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch most rated sellers data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("most rated sellers data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
