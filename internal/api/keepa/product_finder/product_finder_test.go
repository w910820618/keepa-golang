package product_finder

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestService_Fetch(t *testing.T) {
	// TODO: 实现单元测试
	t.Skip("TODO: implement test")

	ctx := context.Background()
	logger := zap.NewNop()

	service := NewService(nil, logger)

	params := RequestParams{
		Domain: 1,
	}

	_, err := service.Fetch(ctx, params)
	if err != nil {
		t.Errorf("Fetch() error = %v", err)
	}
}
