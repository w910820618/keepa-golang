package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"go.uber.org/zap"
)

// KeepaBaseURL Keepa API 基础 URL
const KeepaBaseURL = "https://api.keepa.com/"

// Client Keepa API 客户端
type Client struct {
	baseURL    string
	accessKey  string
	httpClient *http.Client
	logger     *zap.Logger
}

// Config API 客户端配置
type Config struct {
	BaseURL   string
	AccessKey string
	Timeout   time.Duration
	Logger    *zap.Logger
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
		logger: cfg.Logger,
	}
}

// DoRequest 执行 HTTP 请求
func (c *Client) DoRequest(ctx context.Context, endpoint string, params map[string]string) (*http.Response, error) {
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

	// 4. 创建 HTTP 请求（使用 context 支持超时和取消）
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, finalURL, nil)
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
	if c.logger != nil {
		c.logger.Debug("sending HTTP request",
			zap.String("method", req.Method),
			zap.String("url", finalURL),
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("HTTP request failed",
				zap.String("url", finalURL),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// 6. 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		err := fmt.Errorf("HTTP error: status code %d", resp.StatusCode)
		if c.logger != nil {
			c.logger.Error("HTTP response error",
				zap.Int("status_code", resp.StatusCode),
				zap.String("status", resp.Status),
				zap.String("url", finalURL),
			)
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

	if c.logger != nil {
		c.logger.Debug("response body read successfully",
			zap.String("endpoint", endpoint),
			zap.Int("body_size", len(body)),
		)
	}

	// 3. 返回原始字节数据
	return body, nil
}

// PostRawData 使用 POST 方法发送 JSON 数据并获取原始响应
func (c *Client) PostRawData(ctx context.Context, endpoint string, jsonBody interface{}) ([]byte, error) {
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

	// 5. 创建 HTTP POST 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, finalURL, bytes.NewBuffer(bodyBytes))
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
	if c.logger != nil {
		c.logger.Debug("sending HTTP POST request",
			zap.String("method", req.Method),
			zap.String("url", finalURL),
			zap.Int("body_size", len(bodyBytes)),
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("HTTP request failed",
				zap.String("url", finalURL),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 7. 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("HTTP error: status code %d", resp.StatusCode)
		if c.logger != nil {
			c.logger.Error("HTTP response error",
				zap.Int("status_code", resp.StatusCode),
				zap.String("status", resp.Status),
				zap.String("url", finalURL),
			)
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

	if c.logger != nil {
		c.logger.Debug("POST request succeeded",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", finalURL),
			zap.Int("response_size", len(body)),
		)
	}

	return body, nil
}

// PostRawDataWithParams 使用 POST 方法发送 JSON 数据并获取原始响应，支持额外的查询参数
func (c *Client) PostRawDataWithParams(ctx context.Context, endpoint string, params map[string]string, jsonBody interface{}) ([]byte, error) {
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

	// 5. 创建 HTTP POST 请求
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, finalURL, bytes.NewBuffer(bodyBytes))
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
	if c.logger != nil {
		c.logger.Debug("sending HTTP POST request with params",
			zap.String("method", req.Method),
			zap.String("url", finalURL),
			zap.Int("body_size", len(bodyBytes)),
		)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		if c.logger != nil {
			c.logger.Error("HTTP request failed",
				zap.String("url", finalURL),
				zap.Error(err),
			)
		}
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	// 7. 检查响应状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		err := fmt.Errorf("HTTP error: status code %d", resp.StatusCode)
		if c.logger != nil {
			c.logger.Error("HTTP response error",
				zap.Int("status_code", resp.StatusCode),
				zap.String("status", resp.Status),
				zap.String("url", finalURL),
			)
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

	if c.logger != nil {
		c.logger.Debug("POST request with params succeeded",
			zap.Int("status_code", resp.StatusCode),
			zap.String("url", finalURL),
			zap.Int("response_size", len(body)),
		)
	}

	return body, nil
}
