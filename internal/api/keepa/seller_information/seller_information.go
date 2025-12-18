package seller_information

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Seller Information 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Seller Information 服务
func NewService(client *api.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// RequestParams Seller Information 请求参数
type RequestParams struct {
	// 必需参数
	Domain    int      `json:"domain"`     // Amazon 域名代码 (1=com, 2=co.uk, 3=de, 4=fr, 5=co.jp, 6=ca, 8=it, 9=es, 10=in, 11=com.mx)
	SellerIDs []string `json:"seller_ids"` // 卖家 ID 列表（最多100个，逗号分隔）

	// 可选参数
	Storefront *int `json:"storefront,omitempty"` // 是否包含店铺信息：0=否, 1=是（额外9个token）
	Update     *int `json:"update,omitempty"`     // 更新阈值（小时数），如果上次收集超过此值则强制从Amazon收集新数据（50个token，必须与storefront一起使用）
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

	// 验证 SellerIDs 不能为空
	if len(p.SellerIDs) == 0 {
		return fmt.Errorf("seller_ids is required and cannot be empty")
	}

	// 验证 SellerIDs 数量（最多100个）
	if len(p.SellerIDs) > 100 {
		return fmt.Errorf("too many seller IDs: %d (maximum 100 allowed)", len(p.SellerIDs))
	}

	// 验证每个 SellerID 格式（去除空格）
	for i, sellerID := range p.SellerIDs {
		sellerID = strings.TrimSpace(sellerID)
		if sellerID == "" {
			return fmt.Errorf("seller ID at index %d is empty", i)
		}
		p.SellerIDs[i] = sellerID // 标准化 SellerID（去除空格）
	}

	// 验证 storefront 值（0或1）
	if p.Storefront != nil && *p.Storefront != 0 && *p.Storefront != 1 {
		return fmt.Errorf("invalid storefront value: %d (must be 0 or 1)", *p.Storefront)
	}

	// 验证 update 值（必须是正整数）
	if p.Update != nil && *p.Update < 0 {
		return fmt.Errorf("invalid update value: %d (must be a non-negative integer)", *p.Update)
	}

	// 验证：如果使用 storefront，不能使用批量请求（多个seller ID）
	if p.Storefront != nil && *p.Storefront == 1 && len(p.SellerIDs) > 1 {
		return fmt.Errorf("storefront parameter cannot be used with batch requests (multiple seller IDs)")
	}

	// 验证：如果使用 update，必须同时使用 storefront
	if p.Update != nil && (p.Storefront == nil || *p.Storefront != 1) {
		return fmt.Errorf("update parameter requires storefront parameter to be set to 1")
	}

	return nil
}

// ToQueryParams 将 RequestParams 转换为查询参数字典
func (p *RequestParams) ToQueryParams() map[string]string {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)
	params["seller"] = strings.Join(p.SellerIDs, ",")

	// 可选参数
	if p.Storefront != nil && *p.Storefront == 1 {
		params["storefront"] = "1"
	}

	if p.Update != nil {
		params["update"] = strconv.Itoa(*p.Update)
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
			zap.Int("seller_count", len(params.SellerIDs)),
			zap.Strings("seller_ids", params.SellerIDs),
		}
		if params.Storefront != nil {
			logFields = append(logFields, zap.Int("storefront", *params.Storefront))
		}
		if params.Update != nil {
			logFields = append(logFields, zap.Int("update", *params.Update))
		}
		s.logger.Info("fetching seller information data", logFields...)
	}

	// 构建 API 请求参数
	endpoint := "/seller"
	requestParams := params.ToQueryParams()

	// 调用 API 获取原始数据
	rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch seller information data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("seller information data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
