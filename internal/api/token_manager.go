package api

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// TokenInfo Keepa API 响应中的 token 信息
type TokenInfo struct {
	RefillRate          int     `json:"refillRate"`          // 每分钟生成的token数量
	RefillIn            int     `json:"refillIn"`            // 距离下次填充的毫秒数
	TokensLeft          int     `json:"tokensLeft"`          // 当前剩余的token数（可能为负）
	TokensConsumed      int     `json:"tokensConsumed"`      // 本次请求消耗的token数
	TokenFlowReduction  float64 `json:"tokenFlowReduction"`  // token填充率减少值
	ProcessingTimeInMs  int     `json:"processingTimeInMs"`  // 服务器处理时间（毫秒）
	Timestamp           int64   `json:"timestamp"`           // 时间戳
	Error               *ErrorInfo `json:"error,omitempty"`   // 错误信息（如果有）
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Details string `json:"details"`
}

// TokenManager Token 管理器
type TokenManager struct {
	mu                  sync.RWMutex
	tokensLeft          int           // 当前剩余的token数
	refillRate          int           // 每分钟生成的token数量
	refillIn            time.Duration // 距离下次填充的时间
	lastUpdate          time.Time     // 最后更新时间
	minTokensThreshold  int           // 最小token阈值
	maxWaitTime         time.Duration // 最大等待时间
	enableRateLimit     bool          // 是否启用速率限制
	logger              *zap.Logger
}

// maxWaitTime 字段需要导出以便在 handle429Error 中使用
func (tm *TokenManager) GetMaxWaitTime() time.Duration {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.maxWaitTime
}

// TokenManagerConfig Token 管理器配置
type TokenManagerConfig struct {
	MinTokensThreshold int           // 最小token阈值
	MaxWaitTime         time.Duration // 最大等待时间
	EnableRateLimit     bool          // 是否启用速率限制
	Logger              *zap.Logger
}

// NewTokenManager 创建新的 Token 管理器
func NewTokenManager(cfg TokenManagerConfig) *TokenManager {
	if cfg.MinTokensThreshold <= 0 {
		cfg.MinTokensThreshold = 5 // 默认值
	}
	if cfg.MaxWaitTime == 0 {
		cfg.MaxWaitTime = 60 * time.Minute // 默认60分钟
	}

	return &TokenManager{
		tokensLeft:         0, // 初始值未知，等待第一次响应
		refillRate:         0,
		refillIn:            0,
		lastUpdate:          time.Now(),
		minTokensThreshold: cfg.MinTokensThreshold,
		maxWaitTime:         cfg.MaxWaitTime,
		enableRateLimit:     cfg.EnableRateLimit,
		logger:              cfg.Logger,
	}
}

// UpdateFromResponse 从 API 响应中更新 token 信息
func (tm *TokenManager) UpdateFromResponse(body []byte) error {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	// 解析 token 信息
	if refillRate, ok := response["refillRate"].(float64); ok {
		tm.refillRate = int(refillRate)
	} else if refillRate, ok := response["refillRate"].(int); ok {
		tm.refillRate = refillRate
	}

	if refillIn, ok := response["refillIn"].(float64); ok {
		tm.refillIn = time.Duration(refillIn) * time.Millisecond
	} else if refillIn, ok := response["refillIn"].(int); ok {
		tm.refillIn = time.Duration(refillIn) * time.Millisecond
	}

	if tokensLeft, ok := response["tokensLeft"].(float64); ok {
		tm.tokensLeft = int(tokensLeft)
	} else if tokensLeft, ok := response["tokensLeft"].(int); ok {
		tm.tokensLeft = tokensLeft
	}

	tm.lastUpdate = time.Now()

	if tm.logger != nil {
		tm.logger.Debug("token info updated",
			zap.Int("tokens_left", tm.tokensLeft),
			zap.Int("refill_rate", tm.refillRate),
			zap.Duration("refill_in", tm.refillIn),
		)
	}

	return nil
}

// UpdateFromTokenInfo 从 TokenInfo 结构更新
func (tm *TokenManager) UpdateFromTokenInfo(info *TokenInfo) {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.tokensLeft = info.TokensLeft
	tm.refillRate = info.RefillRate
	tm.refillIn = time.Duration(info.RefillIn) * time.Millisecond
	tm.lastUpdate = time.Now()

	if tm.logger != nil {
		tm.logger.Debug("token info updated from TokenInfo",
			zap.Int("tokens_left", tm.tokensLeft),
			zap.Int("refill_rate", tm.refillRate),
			zap.Duration("refill_in", tm.refillIn),
		)
	}
}

// GetTokenInfo 获取当前 token 信息
func (tm *TokenManager) GetTokenInfo() (tokensLeft int, refillRate int, refillIn time.Duration) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 计算经过的时间，更新 tokensLeft（如果 refillRate > 0）
	// 注意：这里不做自动更新，因为 API 响应是权威数据源
	// 实际的 token 状态应该通过 API 响应来更新

	return tm.tokensLeft, tm.refillRate, tm.refillIn
}

// ShouldWait 判断是否需要等待
func (tm *TokenManager) ShouldWait() (bool, time.Duration) {
	if !tm.enableRateLimit {
		return false, 0
	}

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	// 如果 tokensLeft 小于阈值，需要等待
	if tm.tokensLeft < tm.minTokensThreshold {
		// 计算需要等待的时间
		waitTime := tm.refillIn
		
		// 如果 refillIn 为 0 或很小，但 tokensLeft 仍然不足，使用估算值
		if waitTime < time.Second && tm.refillRate > 0 {
			// 估算需要等待的时间：需要恢复 (minTokensThreshold - tokensLeft) 个 token
			tokensNeeded := tm.minTokensThreshold - tm.tokensLeft
			if tokensNeeded > 0 {
				// 每分钟生成 refillRate 个 token
				minutesNeeded := float64(tokensNeeded) / float64(tm.refillRate)
				waitTime = time.Duration(minutesNeeded * float64(time.Minute))
			}
		}

		// 限制最大等待时间
		if waitTime > tm.maxWaitTime {
			waitTime = tm.maxWaitTime
		}

		// 至少等待 1 秒
		if waitTime < time.Second {
			waitTime = time.Second
		}

		return true, waitTime
	}

	return false, 0
}

// WaitIfNeeded 如果需要等待，则等待
// 注意：即使原始context被取消，我们仍然需要等待token恢复，因为这是必要的
func (tm *TokenManager) WaitIfNeeded(ctx context.Context) error {
	shouldWait, waitTime := tm.ShouldWait()
	if !shouldWait {
		return nil
	}

	if tm.logger != nil {
		tm.logger.Info("waiting for token refill",
			zap.Int("tokens_left", tm.tokensLeft),
			zap.Int("min_threshold", tm.minTokensThreshold),
			zap.Duration("wait_time", waitTime),
		)
	}

	// 创建一个独立的context用于等待，不依赖于请求的context
	// 因为即使请求超时，我们也需要等待token恢复
	waitCtx, cancel := context.WithTimeout(context.Background(), waitTime)
	defer cancel()

	// 同时监听原始context，如果被取消，记录警告但继续等待
	originalCtxDone := make(chan struct{})
	if ctx != nil {
		go func() {
			<-ctx.Done()
			close(originalCtxDone)
		}()
	}

	// 等待
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	originalCtxCanceled := false
	for {
		select {
		case <-waitCtx.Done():
			// 等待超时，检查token是否已恢复
			shouldWait, _ := tm.ShouldWait()
			if shouldWait {
				// 仍然需要等待，但已经达到最大等待时间
				if tm.logger != nil {
					tm.logger.Warn("token wait timeout reached, but tokens may still be insufficient",
						zap.Int("tokens_left", tm.tokensLeft),
					)
				}
				// 继续尝试，让API调用决定是否成功
				return nil
			}
			// Token已恢复
			if tm.logger != nil {
				tm.logger.Info("token refilled after wait timeout")
			}
			return nil
		case <-originalCtxDone:
			// 原始context被取消，记录警告但继续等待
			if !originalCtxCanceled {
				originalCtxCanceled = true
				if tm.logger != nil {
					tm.logger.Warn("original request context canceled, but continuing to wait for token refill",
						zap.String("reason", ctx.Err().Error()),
					)
				}
			}
		case <-ticker.C:
			// 每秒检查一次是否还需要等待
			shouldWait, remainingWait := tm.ShouldWait()
			if !shouldWait {
				if tm.logger != nil {
					tm.logger.Info("token refilled, resuming requests")
				}
				return nil
			}
			// 如果剩余等待时间很短，可以提前返回
			if remainingWait < 2*time.Second {
				time.Sleep(remainingWait)
				return nil
			}
		}
	}
}

// ParseTokenInfoFromResponse 从响应中解析 TokenInfo
func ParseTokenInfoFromResponse(body []byte) (*TokenInfo, error) {
	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	info := &TokenInfo{}

	// 解析各个字段
	if v, ok := response["refillRate"]; ok {
		switch val := v.(type) {
		case float64:
			info.RefillRate = int(val)
		case int:
			info.RefillRate = val
		}
	}

	if v, ok := response["refillIn"]; ok {
		switch val := v.(type) {
		case float64:
			info.RefillIn = int(val)
		case int:
			info.RefillIn = val
		}
	}

	if v, ok := response["tokensLeft"]; ok {
		switch val := v.(type) {
		case float64:
			info.TokensLeft = int(val)
		case int:
			info.TokensLeft = val
		}
	}

	if v, ok := response["tokensConsumed"]; ok {
		switch val := v.(type) {
		case float64:
			info.TokensConsumed = int(val)
		case int:
			info.TokensConsumed = val
		}
	}

	if v, ok := response["tokenFlowReduction"]; ok {
		switch val := v.(type) {
		case float64:
			info.TokenFlowReduction = val
		}
	}

	if v, ok := response["processingTimeInMs"]; ok {
		switch val := v.(type) {
		case float64:
			info.ProcessingTimeInMs = int(val)
		case int:
			info.ProcessingTimeInMs = val
		}
	}

	if v, ok := response["timestamp"]; ok {
		switch val := v.(type) {
		case float64:
			info.Timestamp = int64(val)
		case int64:
			info.Timestamp = val
		}
	}

	// 解析错误信息（如果有）
	if errorObj, ok := response["error"].(map[string]interface{}); ok {
		info.Error = &ErrorInfo{}
		if v, ok := errorObj["type"].(string); ok {
			info.Error.Type = v
		}
		if v, ok := errorObj["message"].(string); ok {
			info.Error.Message = v
		}
		if v, ok := errorObj["details"].(string); ok {
			info.Error.Details = v
		}
	}

	return info, nil
}

