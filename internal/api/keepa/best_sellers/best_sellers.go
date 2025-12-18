package best_sellers

import (
	"context"
	"fmt"
	"strconv"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Best Sellers 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Best Sellers 服务
func NewService(client *api.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// RequestParams 请求参数
type RequestParams struct {
	// 必需参数
	Domain   int    `json:"domain"`   // Amazon 域名代码 (1=US, 2=UK, 3=DE, 4=FR, 5=JP, 6=CA, 8=IT, 9=ES, 10=IN, 11=MX)
	Category string `json:"category"` // 类别 ID 或产品组名称（如 "Beauty"）

	// 可选参数 - 排名范围（与 month/year 和 sublist 互斥）
	Range *int `json:"range,omitempty"` // 排名范围: 0=当前排名, 30=30天平均, 90=90天平均, 180=180天平均

	// 可选参数 - 历史数据（与 range 和 sublist 互斥）
	Month *int `json:"month,omitempty"` // 月份: 1-12 (必须与 Year 同时使用)
	Year  *int `json:"year,omitempty"`  // 年份: 4位数年份 (必须与 Month 同时使用)

	// 可选参数 - 变体控制
	Variations *int `json:"variations,omitempty"` // 0=每个父项返回一个变体(默认), 1=返回所有变体

	// 可选参数 - 子类别列表（与 range 和 month/year 互斥）
	Sublist *int `json:"sublist,omitempty"` // 0=基于主排名(默认), 1=基于子类别排名
}

// Validate 验证请求参数
func (p *RequestParams) Validate() error {
	// 验证 domain (有效值: 1,2,3,4,5,6,8,9,10,11，不包括7)
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

	// 验证 category 不能为空
	if p.Category == "" {
		return fmt.Errorf("category is required")
	}

	// 验证 range 值
	if p.Range != nil {
		validRanges := map[int]bool{
			0:   true, // 当前排名
			30:  true, // 30天平均
			90:  true, // 90天平均
			180: true, // 180天平均
		}
		if !validRanges[*p.Range] {
			return fmt.Errorf("invalid range: %d (valid values: 0, 30, 90, 180)", *p.Range)
		}
	}

	// 验证 month 值
	if p.Month != nil {
		if *p.Month < 1 || *p.Month > 12 {
			return fmt.Errorf("invalid month: %d (must be between 1 and 12)", *p.Month)
		}
	}

	// 验证 year 值
	if p.Year != nil {
		if *p.Year < 2000 || *p.Year > 9999 {
			return fmt.Errorf("invalid year: %d (must be a 4-digit year)", *p.Year)
		}
	}

	// 验证 month 和 year 必须同时使用或都不使用
	if (p.Month != nil && p.Year == nil) || (p.Month == nil && p.Year != nil) {
		return fmt.Errorf("month and year must be used together")
	}

	// 验证 range 不能与 month/year 同时使用
	if p.Range != nil && (p.Month != nil || p.Year != nil) {
		return fmt.Errorf("range cannot be used together with month/year")
	}

	// 验证 range 不能与 sublist 同时使用
	if p.Range != nil && p.Sublist != nil {
		return fmt.Errorf("range cannot be used together with sublist")
	}

	// 验证 month/year 不能与 sublist 同时使用
	if (p.Month != nil || p.Year != nil) && p.Sublist != nil {
		return fmt.Errorf("month/year cannot be used together with sublist")
	}

	// 验证 variations 值
	if p.Variations != nil {
		if *p.Variations != 0 && *p.Variations != 1 {
			return fmt.Errorf("invalid variations: %d (valid values: 0, 1)", *p.Variations)
		}
	}

	// 验证 sublist 值
	if p.Sublist != nil {
		if *p.Sublist != 0 && *p.Sublist != 1 {
			return fmt.Errorf("invalid sublist: %d (valid values: 0, 1)", *p.Sublist)
		}
	}

	return nil
}

// ToQueryParams 将 RequestParams 转换为查询参数字典
func (p *RequestParams) ToQueryParams() map[string]string {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)
	params["category"] = p.Category

	// 可选参数
	if p.Range != nil {
		params["range"] = strconv.Itoa(*p.Range)
	}

	if p.Month != nil {
		params["month"] = strconv.Itoa(*p.Month)
	}

	if p.Year != nil {
		params["year"] = strconv.Itoa(*p.Year)
	}

	if p.Variations != nil {
		params["variations"] = strconv.Itoa(*p.Variations)
	}

	if p.Sublist != nil {
		params["sublist"] = strconv.Itoa(*p.Sublist)
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
			zap.String("category", params.Category),
		}
		if params.Range != nil {
			logFields = append(logFields, zap.Int("range", *params.Range))
		}
		if params.Month != nil {
			logFields = append(logFields, zap.Int("month", *params.Month))
		}
		if params.Year != nil {
			logFields = append(logFields, zap.Int("year", *params.Year))
		}
		if params.Variations != nil {
			logFields = append(logFields, zap.Int("variations", *params.Variations))
		}
		if params.Sublist != nil {
			logFields = append(logFields, zap.Int("sublist", *params.Sublist))
		}

		s.logger.Info("fetching best sellers data", logFields...)
	}

	// 构建 API 请求参数
	endpoint := "/bestsellers"
	requestParams := params.ToQueryParams()

	// 调用 API 获取原始数据
	rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch best sellers data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("best sellers data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
