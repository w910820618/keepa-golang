package products

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Products 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Products 服务
func NewService(client *api.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// StatsValue 统计信息值，可以是天数（整数）或日期范围（字符串）
// 日期范围格式：两个时间戳（Unix epoch time in milliseconds）或两个日期字符串（ISO8601格式）
// 例如："2015-10-20,2015-12-24" 或 "1445299200000,1450915200000"
type StatsValue struct {
	Days      *int    // 天数（正整数）
	DateRange *string // 日期范围（格式：两个时间戳或两个日期字符串，用逗号分隔）
}

// RequestParams Product 请求参数
type RequestParams struct {
	// 必需参数
	Domain int `json:"domain"` // Amazon 域名代码 (1=com, 2=co.uk, 3=de, 4=fr, 5=co.jp, 6=ca, 8=it, 9=es, 10=in, 11=com.mx, 12=com.br)

	// 必需参数（二选一，不能同时使用）
	ASINs []string `json:"asins,omitempty"` // ASIN 列表（最多100个，逗号分隔）
	Codes []string `json:"codes,omitempty"` // 产品代码列表（UPC, EAN, ISBN-13，最多100个，逗号分隔）

	// 可选参数
	Stats                *StatsValue `json:"stats,omitempty"`                 // 统计信息：天数（正整数）或日期范围（字符串）
	Update               *int        `json:"update,omitempty"`                // 更新阈值（小时数），-1=不更新，0=总是获取最新，>0=超过此小时数则刷新
	History              *int        `json:"history,omitempty"`               // 是否排除历史数据：0=排除, 1=包含（默认包含）
	Days                 *int        `json:"days,omitempty"`                  // 限制历史数据到最近 X 天（正整数）
	CodeLimit            *int        `json:"code-limit,omitempty"`            // 使用 code 参数时，限制每个代码返回的产品数量（正整数）
	Offers               *int        `json:"offers,omitempty"`                // 获取市场报价数量（20-100之间的正整数）
	OnlyLiveOffers       *int        `json:"only-live-offers,omitempty"`      // 是否只包含实时报价：0=否, 1=是
	Rental               *int        `json:"rental,omitempty"`                // 是否收集租赁价格：0=否, 1=是（需要与 offers 一起使用）
	Videos               *int        `json:"videos,omitempty"`                // 是否包含视频元数据：0=否, 1=是
	Aplus                *int        `json:"aplus,omitempty"`                 // 是否包含 A+ 内容：0=否, 1=是
	Rating               *int        `json:"rating,omitempty"`                // 是否包含评分和评论数历史：0=否, 1=是
	Buybox               *int        `json:"buybox,omitempty"`                // 是否包含 Buy Box 相关数据：0=否, 1=是
	Stock                *int        `json:"stock,omitempty"`                 // 是否包含库存信息：0=否, 1=是（需要与 offers 一起使用）
	HistoricalVariations *int        `json:"historical-variations,omitempty"` // 是否包含历史变体：0=否, 1=是
}

// Validate 验证请求参数
func (p *RequestParams) Validate() error {
	// 验证 domain (有效值: 1,2,3,4,5,6,8,9,10,11,12)
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
		12: true, // com.br
	}
	if !validDomains[p.Domain] {
		return fmt.Errorf("invalid domain: %d (valid domains: 1,2,3,4,5,6,8,9,10,11,12)", p.Domain)
	}

	// 验证 ASINs 和 Codes 不能同时使用
	if len(p.ASINs) > 0 && len(p.Codes) > 0 {
		return fmt.Errorf("cannot use both asins and codes parameters in the same request")
	}

	// 验证至少需要提供 ASINs 或 Codes 之一
	if len(p.ASINs) == 0 && len(p.Codes) == 0 {
		return fmt.Errorf("either asins or codes parameter is required")
	}

	// 验证 ASINs 数量（最多100个）
	if len(p.ASINs) > 100 {
		return fmt.Errorf("too many ASINs: %d (maximum 100 allowed)", len(p.ASINs))
	}

	// 验证每个 ASIN 格式（如果提供）
	for i, asin := range p.ASINs {
		asin = strings.TrimSpace(asin)
		if asin == "" {
			return fmt.Errorf("ASIN at index %d is empty", i)
		}
		if len(asin) != 10 {
			return fmt.Errorf("invalid ASIN at index %d: %s (ASIN must be exactly 10 characters)", i, asin)
		}
		p.ASINs[i] = asin // 标准化 ASIN（去除空格）
	}

	// 验证 Codes 数量（最多100个）
	if len(p.Codes) > 100 {
		return fmt.Errorf("too many codes: %d (maximum 100 allowed)", len(p.Codes))
	}

	// 验证每个 Code 格式（如果提供）
	for i, code := range p.Codes {
		code = strings.TrimSpace(code)
		if code == "" {
			return fmt.Errorf("code at index %d is empty", i)
		}
		p.Codes[i] = code // 标准化 code（去除空格）
	}

	// 验证 stats 值
	if p.Stats != nil {
		if p.Stats.Days != nil && p.Stats.DateRange != nil {
			return fmt.Errorf("stats cannot have both days and date range")
		}
		if p.Stats.Days != nil && *p.Stats.Days <= 0 {
			return fmt.Errorf("stats days must be a positive integer, got: %d", *p.Stats.Days)
		}
		if p.Stats.DateRange != nil {
			parts := strings.Split(*p.Stats.DateRange, ",")
			if len(parts) != 2 {
				return fmt.Errorf("stats date range must have exactly two parts separated by comma, got: %s", *p.Stats.DateRange)
			}
			// 验证日期范围格式（可以是时间戳或日期字符串）
			for i, part := range parts {
				part = strings.TrimSpace(part)
				if part == "" {
					return fmt.Errorf("stats date range part %d cannot be empty", i+1)
				}
			}
		}
	}

	// 验证 update 值（可以是 -1, 0, 或正整数）
	if p.Update != nil && *p.Update < -1 {
		return fmt.Errorf("invalid update value: %d (must be -1, 0, or a positive integer)", *p.Update)
	}

	// 验证 history 值
	if p.History != nil && *p.History != 0 && *p.History != 1 {
		return fmt.Errorf("invalid history value: %d (must be 0 or 1)", *p.History)
	}

	// 验证 days 值（必须是正整数）
	if p.Days != nil && *p.Days <= 0 {
		return fmt.Errorf("invalid days value: %d (must be a positive integer)", *p.Days)
	}

	// 验证 code-limit 值（必须是正整数）
	if p.CodeLimit != nil && *p.CodeLimit <= 0 {
		return fmt.Errorf("invalid code-limit value: %d (must be a positive integer)", *p.CodeLimit)
	}

	// 验证 offers 值（必须是 20-100 之间的正整数）
	if p.Offers != nil {
		if *p.Offers < 20 || *p.Offers > 100 {
			return fmt.Errorf("invalid offers value: %d (must be between 20 and 100)", *p.Offers)
		}
	}

	// 验证 only-live-offers 值
	if p.OnlyLiveOffers != nil && *p.OnlyLiveOffers != 0 && *p.OnlyLiveOffers != 1 {
		return fmt.Errorf("invalid only-live-offers value: %d (must be 0 or 1)", *p.OnlyLiveOffers)
	}

	// 验证 rental 值
	if p.Rental != nil && *p.Rental != 0 && *p.Rental != 1 {
		return fmt.Errorf("invalid rental value: %d (must be 0 or 1)", *p.Rental)
	}

	// 验证 rental 需要与 offers 一起使用
	if p.Rental != nil && *p.Rental == 1 && (p.Offers == nil || *p.Offers == 0) {
		return fmt.Errorf("rental parameter requires offers parameter to be set")
	}

	// 验证 videos 值
	if p.Videos != nil && *p.Videos != 0 && *p.Videos != 1 {
		return fmt.Errorf("invalid videos value: %d (must be 0 or 1)", *p.Videos)
	}

	// 验证 aplus 值
	if p.Aplus != nil && *p.Aplus != 0 && *p.Aplus != 1 {
		return fmt.Errorf("invalid aplus value: %d (must be 0 or 1)", *p.Aplus)
	}

	// 验证 rating 值
	if p.Rating != nil && *p.Rating != 0 && *p.Rating != 1 {
		return fmt.Errorf("invalid rating value: %d (must be 0 or 1)", *p.Rating)
	}

	// 验证 buybox 值
	if p.Buybox != nil && *p.Buybox != 0 && *p.Buybox != 1 {
		return fmt.Errorf("invalid buybox value: %d (must be 0 or 1)", *p.Buybox)
	}

	// 验证 stock 值
	if p.Stock != nil && *p.Stock != 0 && *p.Stock != 1 {
		return fmt.Errorf("invalid stock value: %d (must be 0 or 1)", *p.Stock)
	}

	// 验证 stock 需要与 offers 一起使用
	if p.Stock != nil && *p.Stock == 1 && (p.Offers == nil || *p.Offers == 0) {
		return fmt.Errorf("stock parameter requires offers parameter to be set")
	}

	// 验证 historical-variations 值
	if p.HistoricalVariations != nil && *p.HistoricalVariations != 0 && *p.HistoricalVariations != 1 {
		return fmt.Errorf("invalid historical-variations value: %d (must be 0 or 1)", *p.HistoricalVariations)
	}

	return nil
}

// ToQueryParams 将 RequestParams 转换为查询参数字典
func (p *RequestParams) ToQueryParams() map[string]string {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)

	// ASINs 或 Codes（二选一）
	if len(p.ASINs) > 0 {
		params["asin"] = strings.Join(p.ASINs, ",")
	} else if len(p.Codes) > 0 {
		params["code"] = strings.Join(p.Codes, ",")
	}

	// 可选参数
	// stats 参数处理
	if p.Stats != nil {
		if p.Stats.Days != nil {
			params["stats"] = strconv.Itoa(*p.Stats.Days)
		} else if p.Stats.DateRange != nil {
			params["stats"] = *p.Stats.DateRange
		}
	}

	if p.Update != nil {
		params["update"] = strconv.Itoa(*p.Update)
	}

	if p.History != nil && *p.History == 0 {
		params["history"] = "0"
	}

	if p.Days != nil {
		params["days"] = strconv.Itoa(*p.Days)
	}

	if p.CodeLimit != nil {
		params["code-limit"] = strconv.Itoa(*p.CodeLimit)
	}

	if p.Offers != nil {
		params["offers"] = strconv.Itoa(*p.Offers)
	}

	if p.OnlyLiveOffers != nil && *p.OnlyLiveOffers == 1 {
		params["only-live-offers"] = "1"
	}

	if p.Rental != nil && *p.Rental == 1 {
		params["rental"] = "1"
	}

	if p.Videos != nil && *p.Videos == 1 {
		params["videos"] = "1"
	}

	if p.Aplus != nil && *p.Aplus == 1 {
		params["aplus"] = "1"
	}

	if p.Rating != nil && *p.Rating == 1 {
		params["rating"] = "1"
	}

	if p.Buybox != nil && *p.Buybox == 1 {
		params["buybox"] = "1"
	}

	if p.Stock != nil && *p.Stock == 1 {
		params["stock"] = "1"
	}

	if p.HistoricalVariations != nil && *p.HistoricalVariations == 1 {
		params["historical-variations"] = "1"
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
		if len(params.ASINs) > 0 {
			logFields = append(logFields, zap.Int("asin_count", len(params.ASINs)))
			logFields = append(logFields, zap.Strings("asins", params.ASINs))
		}
		if len(params.Codes) > 0 {
			logFields = append(logFields, zap.Int("code_count", len(params.Codes)))
			logFields = append(logFields, zap.Strings("codes", params.Codes))
		}
		if params.Stats != nil {
			if params.Stats.Days != nil {
				logFields = append(logFields, zap.Int("stats-days", *params.Stats.Days))
			} else if params.Stats.DateRange != nil {
				logFields = append(logFields, zap.String("stats-range", *params.Stats.DateRange))
			}
		}
		if params.Update != nil {
			logFields = append(logFields, zap.Int("update", *params.Update))
		}
		if params.History != nil {
			logFields = append(logFields, zap.Int("history", *params.History))
		}
		if params.Days != nil {
			logFields = append(logFields, zap.Int("days", *params.Days))
		}
		if params.CodeLimit != nil {
			logFields = append(logFields, zap.Int("code-limit", *params.CodeLimit))
		}
		if params.Offers != nil {
			logFields = append(logFields, zap.Int("offers", *params.Offers))
		}
		if params.OnlyLiveOffers != nil {
			logFields = append(logFields, zap.Int("only-live-offers", *params.OnlyLiveOffers))
		}
		if params.Rental != nil {
			logFields = append(logFields, zap.Int("rental", *params.Rental))
		}
		if params.Videos != nil {
			logFields = append(logFields, zap.Int("videos", *params.Videos))
		}
		if params.Aplus != nil {
			logFields = append(logFields, zap.Int("aplus", *params.Aplus))
		}
		if params.Rating != nil {
			logFields = append(logFields, zap.Int("rating", *params.Rating))
		}
		if params.Buybox != nil {
			logFields = append(logFields, zap.Int("buybox", *params.Buybox))
		}
		if params.Stock != nil {
			logFields = append(logFields, zap.Int("stock", *params.Stock))
		}
		if params.HistoricalVariations != nil {
			logFields = append(logFields, zap.Int("historical-variations", *params.HistoricalVariations))
		}
		s.logger.Info("fetching products data", logFields...)
	}

	// 构建 API 请求参数
	endpoint := "/product"
	requestParams := params.ToQueryParams()

	// 调用 API 获取原始数据
	rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch products data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("products data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
