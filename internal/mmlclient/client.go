package mmlclient

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const DefaultServerURL = "http://localhost:8080"

// Client HTTP 客户端，用于与 Keepa Server 通信
type Client struct {
	serverURL  string
	httpClient *http.Client
}

// NewClient 创建新的 HTTP 客户端
func NewClient(serverURL string) *Client {
	if serverURL == "" {
		serverURL = DefaultServerURL
	}
	return &Client{
		serverURL: serverURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// PostJSON 发送 POST 请求并返回格式化的 JSON 响应
func (c *Client) PostJSON(endpoint string, body interface{}) error {
	// 序列化请求体
	var jsonData []byte
	var err error
	if body != nil {
		jsonData, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	// 构建完整 URL
	url := fmt.Sprintf("%s%s", c.serverURL, endpoint)

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP error: status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	// 格式化输出 JSON
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, bodyBytes, "", "  "); err != nil {
		// 如果格式化失败，直接输出原始内容
		fmt.Println(string(bodyBytes))
		return nil
	}

	fmt.Println(prettyJSON.String())
	return nil
}

// GetServerURL 返回服务器 URL
func (c *Client) GetServerURL() string {
	return c.serverURL
}

// PostJSONAndUnmarshal 发送 POST 请求并解析响应为指定类型
func (c *Client) PostJSONAndUnmarshal(endpoint string, body interface{}, result interface{}) error {
	// 序列化请求体
	var jsonData []byte
	var err error
	if body != nil {
		jsonData, err = json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal request: %w", err)
		}
	}

	// 构建完整 URL
	url := fmt.Sprintf("%s%s", c.serverURL, endpoint)

	// 创建请求
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// 检查 HTTP 状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP error: status code %d, response: %s", resp.StatusCode, string(bodyBytes))
	}

	// 解析响应
	if result != nil {
		if err := json.Unmarshal(bodyBytes, result); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w", err)
		}
	}

	return nil
}
