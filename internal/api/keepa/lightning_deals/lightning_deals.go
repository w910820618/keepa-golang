package lightning_deals

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Lightning Deals 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Lightning Deals 服务
func NewService(client *api.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// LightningDealState Lightning Deal 状态
type LightningDealState string

const (
	StateAvailable    LightningDealState = "AVAILABLE"
	StateWaitlist     LightningDealState = "WAITLIST"
	StateSoldout      LightningDealState = "SOLDOUT"
	StateWaitlistFull LightningDealState = "WAITLISTFULL"
	StateExpired      LightningDealState = "EXPIRED"
	StateSuppressed   LightningDealState = "SUPPRESSED"
)

// RequestParams 请求参数
type RequestParams struct {
	// 必需参数
	Domain int `json:"domain"` // Amazon 域名代码 (1=com, 2=co.uk, 3=de, 4=fr, 5=co.jp, 6=ca, 8=it, 9=es, 10=in, 11=com.mx)

	// 可选参数
	ASIN  string             `json:"asin,omitempty"`  // ASIN，如果指定则只返回该 ASIN 的 deal (token cost: 1)，如果不指定则返回全部列表 (token cost: 500)
	State LightningDealState `json:"state,omitempty"` // 限制返回的 lightning deals 状态
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

	// 验证 state 如果是非空值，必须是有效值
	if p.State != "" {
		validStates := map[LightningDealState]bool{
			StateAvailable:    true,
			StateWaitlist:     true,
			StateSoldout:      true,
			StateWaitlistFull: true,
			StateExpired:      true,
			StateSuppressed:   true,
		}
		if !validStates[p.State] {
			return fmt.Errorf("invalid state: %s (valid states: AVAILABLE, WAITLIST, SOLDOUT, WAITLISTFULL, EXPIRED, SUPPRESSED)", p.State)
		}
	}

	// 验证 ASIN 格式（如果提供）
	if p.ASIN != "" {
		asin := strings.TrimSpace(p.ASIN)
		if len(asin) != 10 {
			return fmt.Errorf("invalid ASIN: %s (ASIN must be exactly 10 characters)", asin)
		}
		p.ASIN = asin // 标准化 ASIN（去除空格）
	}

	return nil
}

// ToQueryParams 将 RequestParams 转换为查询参数字典
func (p *RequestParams) ToQueryParams() map[string]string {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)

	// 可选参数
	if p.ASIN != "" {
		params["asin"] = p.ASIN
	}

	if p.State != "" {
		params["state"] = string(p.State)
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
		}
		if params.ASIN != "" {
			logFields = append(logFields, zap.String("asin", params.ASIN))
		}
		if params.State != "" {
			logFields = append(logFields, zap.String("state", string(params.State)))
		}
		s.logger.Info("fetching lightning deals data", logFields...)
	}

	// 构建 API 请求参数
	endpoint := "/lightningdeal"
	requestParams := params.ToQueryParams()

	// 调用 API 获取原始数据
	rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch lightning deals data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("lightning deals data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
