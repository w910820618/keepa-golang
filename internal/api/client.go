package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"go.uber.org/zap"
)

// KeepaBaseURL Keepa API 基础 URL
const KeepaBaseURL = "https://api.keepa.com/"

// Client Keepa API 客户端
type Client struct {
	baseURL           string
	accessKey         string
	httpClient        *http.Client
	logger            *zap.Logger
	printCurlCommand  bool          // 是否打印 curl 命令
	printResponseBody bool          // 是否打印响应体
	tokenManager      *TokenManager // Token 管理器
}

// Config API 客户端配置
type Config struct {
	BaseURL           string
	AccessKey         string
	Timeout           time.Duration
	Logger            *zap.Logger
	PrintCurlCommand  bool          // 是否打印 curl 命令
	PrintResponseBody bool          // 是否打印响应体
	TokenManager      *TokenManager // Token 管理器（可选）
}

// NewClient 创建新的 API 客户端
func NewClient(cfg Config) *Client {
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Second
	}

	// 如果未提供 BaseURL，使用默认的 Keepa API URL
	baseURL := cfg.BaseURL
	if baseURL == "" {
		baseURL = KeepaBaseURL
	}

	return &Client{
		baseURL:   baseURL,
		accessKey: cfg.AccessKey,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		logger:            cfg.Logger,
		printCurlCommand:  cfg.PrintCurlCommand,
		printResponseBody: cfg.PrintResponseBody,
		tokenManager:      cfg.TokenManager,
	}
}

// buildCurlCommand 构建 curl 命令字符串，用于调试和导入到 Postman
// 生成的命令可以直接复制到终端执行，也可以导入到 Postman 中
func buildCurlCommand(method, url string, headers http.Header, body []byte) string {
	var parts []string
	parts = append(parts, "curl")
	parts = append(parts, "-X", method)

	// 添加 URL（用双引号包裹，更适合 Postman 导入）
	// 转义 URL 中的双引号和反斜杠
	escapedURL := strings.ReplaceAll(url, `\`, `\\`)
	escapedURL = strings.ReplaceAll(escapedURL, `"`, `\"`)
	parts = append(parts, fmt.Sprintf(`"%s"`, escapedURL))

	// 添加请求头
	for key, values := range headers {
		for _, value := range values {
			// 转义双引号和反斜杠，使用双引号包裹值
			escapedValue := strings.ReplaceAll(value, `\`, `\\`)
			escapedValue = strings.ReplaceAll(escapedValue, `"`, `\"`)
			parts = append(parts, "-H", fmt.Sprintf(`"%s: %s"`, key, escapedValue))
		}
	}

	// 对于 POST/PUT 等有 body 的请求，添加 -d 参数
	// 使用单引号包裹 JSON body，这样 JSON 内部的双引号不需要转义
	if len(body) > 0 {
		// 尝试格式化 JSON，如果失败则使用原始字符串
		var prettyJSON bytes.Buffer
		if err := json.Indent(&prettyJSON, body, "", "  "); err == nil && len(prettyJSON.Bytes()) > 0 {
			// JSON 格式化成功，使用格式化后的内容
			bodyStr := prettyJSON.String()
			// 转义单引号（在 shell 中，单引号内的单引号需要用 '\'' 转义）
			bodyStr = strings.ReplaceAll(bodyStr, `'`, `'\''`)
			parts = append(parts, "-d", fmt.Sprintf(`'%s'`, bodyStr))
		} else {
			// JSON 格式化失败，使用原始内容
			bodyStr := string(body)
			// 转义单引号
			bodyStr = strings.ReplaceAll(bodyStr, `'`, `'\''`)
			parts = append(parts, "-d", fmt.Sprintf(`'%s'`, bodyStr))
		}
	}

	return strings.Join(parts, " ")
}

// handle429Error 处理429错误，读取响应体更新token信息并等待恢复
func (c *Client) handle429Error(resp *http.Response, endpoint string, ctx context.Context) error {
	// 读取响应体以获取token信息
	body, readErr := io.ReadAll(resp.Body)
	resp.Body.Close()

	if readErr == nil && c.tokenManager != nil {
		// 尝试从响应中更新token信息
		if tokenInfo, err := ParseTokenInfoFromResponse(body); err == nil {
			c.tokenManager.UpdateFromTokenInfo(tokenInfo)
		} else {
			// 如果解析失败，尝试直接更新
			c.tokenManager.UpdateFromResponse(body)
		}
	}

	// 等待token恢复
	if c.tokenManager != nil {
		if c.logger != nil {
			c.logger.Warn("received 429 error, waiting for token refill",
				zap.String("endpoint", endpoint),
			)
		}

		// 等待恢复（即使原始context被取消，也要等待token恢复）
		if err := c.tokenManager.WaitIfNeeded(ctx); err != nil {
			// 即使等待出错，也继续尝试，因为token可能已经恢复
			if c.logger != nil {
				c.logger.Warn("token wait returned error, but will retry request",
					zap.Error(err),
				)
			}
		}
	}

	return nil
}

// formatResponseBody 格式化响应体用于日志输出
// 如果是 JSON，则格式化；否则返回原始字符串（限制长度）
func formatResponseBody(body []byte, maxLength int) string {
	if maxLength <= 0 {
		maxLength = 10000 // 默认最大长度 10KB
	}

	// 尝试解析为 JSON
	var jsonData interface{}
	if err := json.Unmarshal(body, &jsonData); err == nil {
		// 是有效的 JSON，格式化输出
		prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
		if err == nil {
			bodyStr := string(prettyJSON)
			// 如果超过最大长度，截断并添加提示
			if len(bodyStr) > maxLength {
				return bodyStr[:maxLength] + "\n... (响应体已截断，实际长度: " + fmt.Sprintf("%d", len(body)) + " 字节)"
			}
			return bodyStr
		}
	}

	// 不是 JSON 或格式化失败，返回原始字符串
	bodyStr := string(body)
	if len(bodyStr) > maxLength {
		return bodyStr[:maxLength] + "\n... (响应体已截断，实际长度: " + fmt.Sprintf("%d", len(body)) + " 字节)"
	}
	return bodyStr
}

// DoRequest 执行 HTTP 请求
func (c *Client) DoRequest(ctx context.Context, endpoint string, params map[string]string) (*http.Response, error) {
	// 在发送请求前，检查是否需要等待
	// 注意：即使原始context被取消，我们也要等待token恢复（如果需要）
	if c.tokenManager != nil {
		// 使用独立的context等待，不依赖请求的context
		// 因为即使请求context被取消，我们也需要等待token恢复
		if err := c.tokenManager.WaitIfNeeded(ctx); err != nil {
			// WaitIfNeeded现在会继续等待（即使context被取消），所以这里的错误通常是超时
			// 即使等待出错，也继续尝试请求，因为token可能已经恢复
			if c.logger != nil {
				c.logger.Warn("token wait returned error, but will continue with request",
					zap.Error(err),
				)
			}
		}
	}

	// 检查原始context是否已被取消，如果已取消，使用新的context
	// 这样可以避免HTTP请求立即失败
	requestCtx := ctx
	select {
	case <-ctx.Done():
		// 原始context已被取消，创建新的context用于请求
		if c.logger != nil {
			c.logger.Warn("original context canceled, using new context for HTTP request",
				zap.String("endpoint", endpoint),
			)
		}
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(context.Background(), c.httpClient.Timeout)
		defer cancel()
	default:
		// 原始context仍然有效，继续使用
	}

	if c.logger != nil {
		c.logger.Debug("sending API request",
			zap.String("endpoint", endpoint),
			zap.Any("params", params),
		)
	}

	// 1. 构建完整的 URL
	fullURL, err := url.JoinPath(c.baseURL, endpoint)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to build URL",
				zap.String("base_url", c.baseURL),
				zap.String("endpoint", endpoint),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// 2. 解析 URL 以便添加查询参数
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to parse URL",
				zap.String("url", fullURL),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// 3. 构建查询参数
	query := parsedURL.Query()

	// 添加 access key（必需参数）
	if c.accessKey == "" {
		err := fmt.Errorf("access key is required but not set")
		if c.logger != nil {
			c.logger.Error("access key missing", zap.Error(err))
		}
		return nil, err
	}
	query.Set("key", c.accessKey)

	// 添加其他查询参数
	for key, value := range params {
		query.Set(key, value)
	}

	parsedURL.RawQuery = query.Encode()
	finalURL := parsedURL.String()

	// 4. 创建 HTTP 请求（使用 requestCtx，可能是原始context或新创建的context）
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, finalURL, nil)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to create request",
				zap.String("url", finalURL),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "keepa-client/1.0")
	req.Header.Set("Accept", "application/json")

	// 5. 发送请求
	// 记录完整的 curl 命令（如果配置启用）
	if c.logger != nil && c.printCurlCommand {
		curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, nil)
		c.logger.Info("sending HTTP request to Keepa API",
			zap.String("method", req.Method),
			zap.String("url", finalURL),
			zap.String("curl_command", curlCmd),
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			logFields := []zap.Field{
				zap.String("url", finalURL),
				zap.Error(err),
			}
			if c.printCurlCommand {
				curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, nil)
				logFields = append(logFields, zap.String("curl_command", curlCmd))
			}
			c.logger.Error("HTTP request failed", logFields...)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// 6. 检查响应状态码
	// 对于429错误，需要读取响应体获取token信息，然后等待恢复
	if resp.StatusCode == http.StatusTooManyRequests {
		// 读取响应体以获取token信息
		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if readErr == nil && c.tokenManager != nil {
			// 尝试从响应中更新token信息
			if tokenInfo, err := ParseTokenInfoFromResponse(body); err == nil {
				c.tokenManager.UpdateFromTokenInfo(tokenInfo)
			} else {
				// 如果解析失败，尝试直接更新
				c.tokenManager.UpdateFromResponse(body)
			}
		}

		// 等待token恢复
		if c.tokenManager != nil {
			if c.logger != nil {
				c.logger.Warn("received 429 error, waiting for token refill",
					zap.String("endpoint", endpoint),
				)
			}

			// 等待恢复（即使原始context被取消，也要等待token恢复）
			if err := c.tokenManager.WaitIfNeeded(ctx); err != nil {
				// 即使等待出错，也继续尝试，因为token可能已经恢复
				if c.logger != nil {
					c.logger.Warn("token wait returned error, but will retry request",
						zap.Error(err),
					)
				}
			}

			// 重试请求（使用新的context，因为原始context可能已取消）
			// 创建一个新的context用于重试
			retryCtx, cancel := context.WithTimeout(context.Background(), c.httpClient.Timeout)
			defer cancel()
			return c.DoRequest(retryCtx, endpoint, params)
		}

		err := fmt.Errorf("HTTP error: status code %d (Too Many Requests)", resp.StatusCode)
		if c.logger != nil {
			logFields := []zap.Field{
				zap.Int("status_code", resp.StatusCode),
				zap.String("status", resp.Status),
				zap.String("url", finalURL),
			}
			if c.printCurlCommand {
				curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, nil)
				logFields = append(logFields, zap.String("curl_command", curlCmd))
			}
			c.logger.Error("HTTP response error", logFields...)
		}
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		err := fmt.Errorf("HTTP error: status code %d", resp.StatusCode)
		if c.logger != nil {
			logFields := []zap.Field{
				zap.Int("status_code", resp.StatusCode),
				zap.String("status", resp.Status),
				zap.String("url", finalURL),
			}
			if c.printCurlCommand {
				curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, nil)
				logFields = append(logFields, zap.String("curl_command", curlCmd))
			}
			c.logger.Error("HTTP response error", logFields...)
		}
		return nil, err
	}

	if c.logger != nil {
		c.logger.Debug("HTTP request succeeded",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", finalURL),
		)
	}

	return resp, nil
}

// GetRawData 获取原始数据
func (c *Client) GetRawData(ctx context.Context, endpoint string, params map[string]string) ([]byte, error) {
	// 1. 调用 DoRequest 获取响应
	resp, err := c.DoRequest(ctx, endpoint, params)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 2. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to read response body",
				zap.String("endpoint", endpoint),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 3. 更新 token 信息（如果启用了 token 管理器）
	if c.tokenManager != nil {
		if err := c.tokenManager.UpdateFromResponse(body); err != nil {
			// 更新失败不影响主要功能，只记录警告
			if c.logger != nil {
				c.logger.Warn("failed to update token info from response",
					zap.String("endpoint", endpoint),
					zap.Error(err),
				)
			}
		}
	}

	// 4. 记录响应结果（如果配置启用）
	if c.logger != nil && c.printResponseBody {
		formattedBody := formatResponseBody(body, 10000) // 最大 10KB
		c.logger.Info("Keepa API response received",
			zap.String("endpoint", endpoint),
			zap.Int("status_code", resp.StatusCode),
			zap.Int("body_size", len(body)),
			zap.String("response_body", formattedBody),
		)
	}

	// 5. 返回原始字节数据
	return body, nil
}

// PostRawData 使用 POST 方法发送 JSON 数据并获取原始响应
func (c *Client) PostRawData(ctx context.Context, endpoint string, jsonBody interface{}) ([]byte, error) {
	// 在发送请求前，检查是否需要等待token
	if c.tokenManager != nil {
		if err := c.tokenManager.WaitIfNeeded(ctx); err != nil {
			if c.logger != nil {
				c.logger.Warn("token wait returned error, but will continue with request",
					zap.Error(err),
				)
			}
		}
	}

	// 检查原始context是否已被取消，如果已取消，使用新的context
	requestCtx := ctx
	select {
	case <-ctx.Done():
		if c.logger != nil {
			c.logger.Warn("original context canceled, using new context for HTTP request",
				zap.String("endpoint", endpoint),
			)
		}
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(context.Background(), c.httpClient.Timeout)
		defer cancel()
	default:
	}

	if c.logger != nil {
		c.logger.Debug("sending POST API request",
			zap.String("endpoint", endpoint),
			zap.Any("body", jsonBody),
		)
	}

	// 1. 构建完整的 URL
	fullURL, err := url.JoinPath(c.baseURL, endpoint)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to build URL",
				zap.String("base_url", c.baseURL),
				zap.String("endpoint", endpoint),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// 2. 解析 URL 以便添加查询参数
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to parse URL",
				zap.String("url", fullURL),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// 3. 构建查询参数（只添加 access key）
	query := parsedURL.Query()

	// 添加 access key（必需参数）
	if c.accessKey == "" {
		err := fmt.Errorf("access key is required but not set")
		if c.logger != nil {
			c.logger.Error("access key missing", zap.Error(err))
		}
		return nil, err
	}
	query.Set("key", c.accessKey)

	parsedURL.RawQuery = query.Encode()
	finalURL := parsedURL.String()

	// 4. 序列化 JSON body
	var bodyBytes []byte
	if jsonBody != nil {
		bodyBytes, err = json.Marshal(jsonBody)
		if err != nil {
			if c.logger != nil {
				c.logger.Error("failed to marshal JSON body",
					zap.Error(err),
				)
			}
			return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
		}
	}

	// 5. 创建 HTTP POST 请求（使用 requestCtx，可能是原始context或新创建的context）
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, finalURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to create request",
				zap.String("url", finalURL),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "keepa-client/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 6. 发送请求
	// 记录完整的 curl 命令（如果配置启用）
	if c.logger != nil && c.printCurlCommand {
		curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, bodyBytes)
		c.logger.Info("sending HTTP POST request to Keepa API",
			zap.String("method", req.Method),
			zap.String("url", finalURL),
			zap.Int("body_size", len(bodyBytes)),
			zap.String("curl_command", curlCmd),
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			logFields := []zap.Field{
				zap.String("url", finalURL),
				zap.Error(err),
			}
			if c.printCurlCommand {
				curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, bodyBytes)
				logFields = append(logFields, zap.String("curl_command", curlCmd))
			}
			c.logger.Error("HTTP request failed", logFields...)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 7. 检查响应状态码
	// 对于429错误，需要读取响应体获取token信息，然后等待恢复
	if resp.StatusCode == http.StatusTooManyRequests {
		if err := c.handle429Error(resp, endpoint, ctx); err != nil {
			return nil, err
		}
		// 重试请求（使用新的context，因为原始context可能已取消）
		retryCtx, cancel := context.WithTimeout(context.Background(), c.httpClient.Timeout)
		defer cancel()
		return c.PostRawData(retryCtx, endpoint, jsonBody)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("HTTP error: status code %d", resp.StatusCode)
		if c.logger != nil {
			logFields := []zap.Field{
				zap.Int("status_code", resp.StatusCode),
				zap.String("status", resp.Status),
				zap.String("url", finalURL),
			}
			if c.printCurlCommand {
				curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, bodyBytes)
				logFields = append(logFields, zap.String("curl_command", curlCmd))
			}
			c.logger.Error("HTTP response error", logFields...)
		}
		return nil, err
	}

	// 8. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to read response body",
				zap.String("endpoint", endpoint),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 9. 更新 token 信息（如果启用了 token 管理器）
	if c.tokenManager != nil {
		if err := c.tokenManager.UpdateFromResponse(body); err != nil {
			// 更新失败不影响主要功能，只记录警告
			if c.logger != nil {
				c.logger.Warn("failed to update token info from response",
					zap.String("endpoint", endpoint),
					zap.Error(err),
				)
			}
		}
	}

	// 10. 记录响应结果（如果配置启用）
	if c.logger != nil && c.printResponseBody {
		formattedBody := formatResponseBody(body, 10000) // 最大 10KB
		c.logger.Info("Keepa API response received",
			zap.String("endpoint", endpoint),
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", finalURL),
			zap.Int("body_size", len(body)),
			zap.String("response_body", formattedBody),
		)
	}

	return body, nil
}

// PostRawDataWithParams 使用 POST 方法发送 JSON 数据并获取原始响应，支持额外的查询参数
func (c *Client) PostRawDataWithParams(ctx context.Context, endpoint string, params map[string]string, jsonBody interface{}) ([]byte, error) {
	// 在发送请求前，检查是否需要等待token
	if c.tokenManager != nil {
		if err := c.tokenManager.WaitIfNeeded(ctx); err != nil {
			if c.logger != nil {
				c.logger.Warn("token wait returned error, but will continue with request",
					zap.Error(err),
				)
			}
		}
	}

	// 检查原始context是否已被取消，如果已取消，使用新的context
	requestCtx := ctx
	select {
	case <-ctx.Done():
		if c.logger != nil {
			c.logger.Warn("original context canceled, using new context for HTTP request",
				zap.String("endpoint", endpoint),
			)
		}
		var cancel context.CancelFunc
		requestCtx, cancel = context.WithTimeout(context.Background(), c.httpClient.Timeout)
		defer cancel()
	default:
	}

	if c.logger != nil {
		c.logger.Debug("sending POST API request with params",
			zap.String("endpoint", endpoint),
			zap.Any("params", params),
			zap.Any("body", jsonBody),
		)
	}

	// 1. 构建完整的 URL
	fullURL, err := url.JoinPath(c.baseURL, endpoint)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to build URL",
				zap.String("base_url", c.baseURL),
				zap.String("endpoint", endpoint),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to build URL: %w", err)
	}

	// 2. 解析 URL 以便添加查询参数
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to parse URL",
				zap.String("url", fullURL),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	// 3. 构建查询参数
	query := parsedURL.Query()

	// 添加 access key（必需参数）
	if c.accessKey == "" {
		err := fmt.Errorf("access key is required but not set")
		if c.logger != nil {
			c.logger.Error("access key missing", zap.Error(err))
		}
		return nil, err
	}
	query.Set("key", c.accessKey)

	// 添加其他查询参数
	for key, value := range params {
		query.Set(key, value)
	}

	parsedURL.RawQuery = query.Encode()
	finalURL := parsedURL.String()

	// 4. 序列化 JSON body
	var bodyBytes []byte
	if jsonBody != nil {
		bodyBytes, err = json.Marshal(jsonBody)
		if err != nil {
			if c.logger != nil {
				c.logger.Error("failed to marshal JSON body",
					zap.Error(err),
				)
			}
			return nil, fmt.Errorf("failed to marshal JSON body: %w", err)
		}
	}

	// 5. 创建 HTTP POST 请求（使用 requestCtx，可能是原始context或新创建的context）
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, finalURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to create request",
				zap.String("url", finalURL),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("User-Agent", "keepa-client/1.0")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	// 6. 发送请求
	// 记录完整的 curl 命令（如果配置启用）
	if c.logger != nil && c.printCurlCommand {
		curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, bodyBytes)
		c.logger.Info("sending HTTP POST request to Keepa API",
			zap.String("method", req.Method),
			zap.String("url", finalURL),
			zap.Int("body_size", len(bodyBytes)),
			zap.String("curl_command", curlCmd),
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			logFields := []zap.Field{
				zap.String("url", finalURL),
				zap.Error(err),
			}
			if c.printCurlCommand {
				curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, bodyBytes)
				logFields = append(logFields, zap.String("curl_command", curlCmd))
			}
			c.logger.Error("HTTP request failed", logFields...)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 7. 检查响应状态码
	// 对于429错误，需要读取响应体获取token信息，然后等待恢复
	if resp.StatusCode == http.StatusTooManyRequests {
		if err := c.handle429Error(resp, endpoint, ctx); err != nil {
			return nil, err
		}
		// 重试请求（使用新的context，因为原始context可能已取消）
		retryCtx, cancel := context.WithTimeout(context.Background(), c.httpClient.Timeout)
		defer cancel()
		return c.PostRawDataWithParams(retryCtx, endpoint, params, jsonBody)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("HTTP error: status code %d", resp.StatusCode)
		if c.logger != nil {
			logFields := []zap.Field{
				zap.Int("status_code", resp.StatusCode),
				zap.String("status", resp.Status),
				zap.String("url", finalURL),
			}
			if c.printCurlCommand {
				curlCmd := buildCurlCommand(req.Method, finalURL, req.Header, bodyBytes)
				logFields = append(logFields, zap.String("curl_command", curlCmd))
			}
			c.logger.Error("HTTP response error", logFields...)
		}
		return nil, err
	}

	// 8. 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("failed to read response body",
				zap.String("endpoint", endpoint),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// 9. 更新 token 信息（如果启用了 token 管理器）
	if c.tokenManager != nil {
		if err := c.tokenManager.UpdateFromResponse(body); err != nil {
			// 更新失败不影响主要功能，只记录警告
			if c.logger != nil {
				c.logger.Warn("failed to update token info from response",
					zap.String("endpoint", endpoint),
					zap.Error(err),
				)
			}
		}
	}

	// 10. 记录响应结果（如果配置启用）
	if c.logger != nil && c.printResponseBody {
		formattedBody := formatResponseBody(body, 10000) // 最大 10KB
		c.logger.Info("Keepa API response received",
			zap.String("endpoint", endpoint),
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", finalURL),
			zap.Int("body_size", len(body)),
			zap.String("response_body", formattedBody),
		)
	}

	return body, nil
}
