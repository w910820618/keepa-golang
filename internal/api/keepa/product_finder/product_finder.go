package product_finder

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"keepa/internal/api"

	"go.uber.org/zap"
)

// Service Product Finder 服务
type Service struct {
	client *api.Client
	logger *zap.Logger
}

// NewService 创建新的 Product Finder 服务
func NewService(client *api.Client, logger *zap.Logger) *Service {
	return &Service{
		client: client,
		logger: logger,
	}
}

// RequestMethod 请求方法类型
type RequestMethod string

const (
	MethodGET  RequestMethod = "GET"
	MethodPOST RequestMethod = "POST"
)

// SortDirection 排序方向
type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

// SortCriteria 排序条件
type SortCriteria struct {
	Field     string        `json:"field"`     // 字段名（不包含 _lte 或 _gte 后缀）
	Direction SortDirection `json:"direction"` // 排序方向: "asc" 或 "desc"
}

// QueryJSON Product Finder 查询 JSON 结构
// 注意：根据文档，查询参数很多，这里先保留框架，后续可以方便地扩充查询参数字段
type QueryJSON struct {
	// 分页参数
	Page    *int `json:"page,omitempty"`    // 页码，从 0 开始，默认 0
	PerPage *int `json:"perPage,omitempty"` // 每页结果数，默认 50，最小 50

	// 排序参数（最多 3 个排序条件）
	Sort []SortCriteria `json:"sort,omitempty"` // 排序条件数组

	// TODO: 后续可以在这里添加查询参数字段
	// 例如：
	// LastPriceChangeLTE *int      `json:"lastPriceChange_lte,omitempty"`
	// LastPriceChangeGTE *int      `json:"lastPriceChange_gte,omitempty"`
	// Brand              []string  `json:"brand,omitempty"`
	// ProductGroup       []string  `json:"productGroup,omitempty"`
	// ... 等等
}

// RequestParams Product Finder 请求参数
type RequestParams struct {
	// 必需参数
	Domain int `json:"domain"` // Amazon 域名代码 (1=com, 2=co.uk, 3=de, 4=fr, 5=co.jp, 6=ca, 8=it, 9=es, 10=in, 11=com.mx, 12=com.br)

	// 查询 JSON（必需，至少要有一个过滤条件）
	Query QueryJSON `json:"query"`

	// 可选参数
	Stats  *int          `json:"stats,omitempty"`  // 是否包含搜索洞察数据，1 表示包含（token cost: 30 + 1 per 1,000,000 products）
	Method RequestMethod `json:"method,omitempty"` // 请求方法，GET 或 POST，默认 POST
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

	// 验证查询 JSON
	if err := p.Query.Validate(); err != nil {
		return fmt.Errorf("invalid query: %w", err)
	}

	// 验证 stats 参数（如果提供）
	if p.Stats != nil && *p.Stats != 0 && *p.Stats != 1 {
		return fmt.Errorf("invalid stats value: %d (must be 0 or 1)", *p.Stats)
	}

	// 验证 method（如果提供）
	if p.Method != "" && p.Method != MethodGET && p.Method != MethodPOST {
		return fmt.Errorf("invalid method: %s (must be GET or POST)", p.Method)
	}

	return nil
}

// Validate 验证查询 JSON
func (q *QueryJSON) Validate() error {
	// 验证 perPage（如果提供）
	if q.PerPage != nil {
		if *q.PerPage < 50 {
			return fmt.Errorf("perPage must be at least 50, got: %d", *q.PerPage)
		}
		// 如果 page 不是 0，则 page * perPage 不能超过 10000
		if q.Page != nil && *q.Page > 0 {
			maxPerPage := 10000 / (*q.Page + 1)
			if *q.PerPage > maxPerPage {
				return fmt.Errorf("when page is %d, perPage cannot exceed %d (page * perPage must not exceed 10000)", *q.Page, maxPerPage)
			}
		}
	}

	// 验证 sort（最多 3 个排序条件）
	if len(q.Sort) > 3 {
		return fmt.Errorf("sort can have at most 3 criteria, got: %d", len(q.Sort))
	}

	// 验证每个排序条件
	for i, sort := range q.Sort {
		if sort.Field == "" {
			return fmt.Errorf("sort[%d].field cannot be empty", i)
		}
		if sort.Direction != SortAsc && sort.Direction != SortDesc {
			return fmt.Errorf("sort[%d].direction must be 'asc' or 'desc', got: %s", i, sort.Direction)
		}
	}

	// TODO: 后续在这里添加查询参数的验证逻辑
	// 例如：验证至少有一个查询条件

	return nil
}

// ToQueryJSONMap 将 QueryJSON 转换为 map[string]interface{} 用于 JSON 序列化
func (q *QueryJSON) ToQueryJSONMap() map[string]interface{} {
	result := make(map[string]interface{})

	// 添加分页参数
	if q.Page != nil {
		result["page"] = *q.Page
	}
	if q.PerPage != nil {
		result["perPage"] = *q.PerPage
	}

	// 添加排序参数
	if len(q.Sort) > 0 {
		sortArray := make([][]interface{}, len(q.Sort))
		for i, sort := range q.Sort {
			sortArray[i] = []interface{}{sort.Field, string(sort.Direction)}
		}
		result["sort"] = sortArray
	}

	// TODO: 后续在这里添加查询参数的转换逻辑
	// 例如：
	// if q.LastPriceChangeLTE != nil {
	//     result["lastPriceChange_lte"] = *q.LastPriceChangeLTE
	// }

	return result
}

// ToQueryParams 将 RequestParams 转换为查询参数字典（用于 GET 请求）
func (p *RequestParams) ToQueryParams() (map[string]string, error) {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)

	// 将 queryJSON 序列化为 JSON 字符串并进行 URL 编码
	queryJSONMap := p.Query.ToQueryJSONMap()
	queryJSONBytes, err := json.Marshal(queryJSONMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal query JSON: %w", err)
	}
	params["selection"] = string(queryJSONBytes)

	// 可选参数 stats
	if p.Stats != nil && *p.Stats == 1 {
		params["stats"] = "1"
	}

	return params, nil
}

// ToPOSTQueryParams 将 RequestParams 转换为 POST 请求的查询参数字典（只包含 URL 中的参数）
func (p *RequestParams) ToPOSTQueryParams() map[string]string {
	params := make(map[string]string)

	// 必需参数
	params["domain"] = strconv.Itoa(p.Domain)

	// 可选参数 stats
	if p.Stats != nil && *p.Stats == 1 {
		params["stats"] = "1"
	}

	return params
}

// ToPOSTBody 将 RequestParams 转换为 POST 请求的 JSON body
func (p *RequestParams) ToPOSTBody() map[string]interface{} {
	return p.Query.ToQueryJSONMap()
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

	// 确定请求方法（默认为 POST）
	method := params.Method
	if method == "" {
		method = MethodPOST
	}

	// 记录请求日志
	if s.logger != nil {
		logFields := []zap.Field{
			zap.Int("domain", params.Domain),
			zap.String("method", string(method)),
		}
		if params.Stats != nil {
			logFields = append(logFields, zap.Int("stats", *params.Stats))
		}
		s.logger.Info("fetching product finder data", logFields...)
	}

	// 构建 API 请求
	endpoint := "/query"
	var rawData []byte
	var err error

	if method == MethodGET {
		// GET 请求：/query?key=<yourAccessKey>&domain=<domainId>&selection=<queryJSON>[&stats=1]
		// selection 参数是 queryJSON 的 JSON 字符串，client.GetRawData 会自动进行 URL 编码
		requestParams, err := params.ToQueryParams()
		if err != nil {
			if s.logger != nil {
				s.logger.Error("failed to build query params", zap.Error(err))
			}
			return nil, fmt.Errorf("failed to build query params: %w", err)
		}

		rawData, err = s.client.GetRawData(ctx, endpoint, requestParams)
		if err != nil {
			if s.logger != nil {
				s.logger.Error("failed to fetch product finder data (GET)",
					zap.Error(err),
					zap.String("endpoint", endpoint),
				)
			}
			return nil, fmt.Errorf("failed to fetch data: %w", err)
		}
	} else {
		// POST 请求：/query?domain=<domainId>&key=<yourAccessKey>[&stats=1]
		// body 是 queryJSON
		requestParams := params.ToPOSTQueryParams()
		requestBody := params.ToPOSTBody()

		rawData, err = s.client.PostRawDataWithParams(ctx, endpoint, requestParams, requestBody)
		if err != nil {
			if s.logger != nil {
				s.logger.Error("failed to fetch product finder data (POST)",
					zap.Error(err),
					zap.String("endpoint", endpoint),
				)
			}
			return nil, fmt.Errorf("failed to fetch data: %w", err)
		}
	}

	if s.logger != nil {
		s.logger.Info("product finder data fetched successfully",
			zap.Int("data_size", len(rawData)),
		)
	}

	return rawData, nil
}
