package product_searches

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Product Searches 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Product Searches 服务
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

// RequestParams Product Search 请求参数
type RequestParams struct {
	// 必需参数
	Domain int    `json:"domain"` // Amazon 域名代码 (1=com, 2=co.uk, 3=de, 4=fr, 5=co.jp, 6=ca, 8=it, 9=es, 10=in, 11=com.mx)
	Term   string `json:"term"`   // 搜索关键词（需要 URL 编码）

	// 可选参数
	AsinsOnly *int        `json:"asins-only,omitempty"` // 如果为 1，只返回 ASINs 列表，不返回完整商品对象
	Page      *int        `json:"page,omitempty"`       // 页码：0-9，每页最多 10 个结果
	Stats     *StatsValue `json:"stats,omitempty"`      // 统计信息：天数（正整数）或日期范围（字符串）
	Update    *int        `json:"update,omitempty"`     // 更新阈值（小时数），如果商品最后更新时间超过此值则强制刷新
	History   *int        `json:"history,omitempty"`    // 是否包含历史数据：0=不包含, 1=包含（默认包含）
	Rating    *int        `json:"rating,omitempty"`     // 是否包含评分数据：0=不包含, 1=包含
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
	if strings.TrimSpace(p.Term) == "" {
		return fmt.Errorf("term is required and cannot be empty")
	}

	// 验证 asins-only 值
	if p.AsinsOnly != nil && *p.AsinsOnly != 0 && *p.AsinsOnly != 1 {
		return fmt.Errorf("invalid asins-only value: %d (must be 0 or 1)", *p.AsinsOnly)
	}

	// 验证 page 值（0-9）
	if p.Page != nil {
		if *p.Page < 0 || *p.Page > 9 {
			return fmt.Errorf("invalid page value: %d (must be between 0 and 9)", *p.Page)
		}
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

	// 验证 update 值（必须是非负整数）
	if p.Update != nil && *p.Update < 0 {
		return fmt.Errorf("invalid update value: %d (must be non-negative)", *p.Update)
	}

	// 验证 history 值
	if p.History != nil && *p.History != 0 && *p.History != 1 {
		return fmt.Errorf("invalid history value: %d (must be 0 or 1)", *p.History)
	}

	// 验证 rating 值
	if p.Rating != nil && *p.Rating != 0 && *p.Rating != 1 {
		return fmt.Errorf("invalid rating value: %d (must be 0 or 1)", *p.Rating)
	}

	return nil
}

// ToQueryParams 将 RequestParams 转换为查询参数字典
func (p *RequestParams) ToQueryParams() map[string]string {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)
	params["type"] = "product"
	params["term"] = p.Term

	// 可选参数
	if p.AsinsOnly != nil && *p.AsinsOnly == 1 {
		params["asins-only"] = "1"
	}

	if p.Page != nil {
		params["page"] = strconv.Itoa(*p.Page)
	}

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

	if p.Rating != nil && *p.Rating == 1 {
		params["rating"] = "1"
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
			zap.String("term", params.Term),
		}
		if params.AsinsOnly != nil {
			logFields = append(logFields, zap.Int("asins-only", *params.AsinsOnly))
		}
		if params.Page != nil {
			logFields = append(logFields, zap.Int("page", *params.Page))
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
		if params.Rating != nil {
			logFields = append(logFields, zap.Int("rating", *params.Rating))
		}
		s.logger.Info("fetching product search data", logFields...)
	}

	// 构建 API 请求参数
	endpoint := "/search"
	requestParams := params.ToQueryParams()

	// 调用 API 获取原始数据
	rawData, err := s.client.GetRawData(ctx, endpoint, requestParams)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to fetch product search data",
				zap.Error(err),
				zap.String("endpoint", endpoint),
			)
		}
		return nil, fmt.Errorf("failed to fetch data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("product search data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
