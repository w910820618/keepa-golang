package api

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

// Storage MongoDB 存储管理器
type Storage struct {
	db     *mongo.Database
	logger *zap.Logger
}

// NewStorage 创建新的存储管理器
func NewStorage(db *mongo.Database, logger *zap.Logger) *Storage {
	return &Storage{
		db:     db,
		logger: logger,
	}
}

// SaveRawData 保存原始数据到 MongoDB
func (s *Storage) SaveRawData(ctx context.Context, collectionName string, data interface{}) error {
	if s.logger != nil {
		s.logger.Debug("saving raw data to MongoDB",
			zap.String("collection", collectionName),
		)
	}

	collection := s.db.Collection(collectionName)

	// 如果 data 是 []byte，尝试解析为 JSON
	var doc interface{}
	switch v := data.(type) {
	case []byte:
		// 尝试解析 JSON
		var jsonDoc interface{}
		if err := json.Unmarshal(v, &jsonDoc); err != nil {
			// 如果解析失败，将原始字节作为字符串存储
			doc = bson.M{
				"raw_data":   string(v),
				"created_at": time.Now(),
			}
		} else {
			// 解析成功，添加时间戳
			if docMap, ok := jsonDoc.(map[string]interface{}); ok {
				docMap["created_at"] = time.Now()
				doc = docMap
			} else {
				doc = bson.M{
					"data":       jsonDoc,
					"created_at": time.Now(),
				}
			}
		}
	default:
		// 其他类型，直接使用
		if docMap, ok := v.(map[string]interface{}); ok {
			docMap["created_at"] = time.Now()
			doc = docMap
		} else {
			doc = bson.M{
				"data":       v,
				"created_at": time.Now(),
			}
		}
	}

	// 插入数据
	result, err := collection.InsertOne(ctx, doc)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to insert data to MongoDB",
				zap.String("collection", collectionName),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to insert data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("data saved to MongoDB successfully",
			zap.String("collection", collectionName),
			zap.Any("inserted_id", result.InsertedID),
		)
	}

	return nil
}

// SaveRawDataWithFilter 保存原始数据，如果已存在则更新
func (s *Storage) SaveRawDataWithFilter(ctx context.Context, collectionName string, filter bson.M, data interface{}) error {
	if s.logger != nil {
		s.logger.Debug("saving raw data to MongoDB with filter",
			zap.String("collection", collectionName),
			zap.Any("filter", filter),
		)
	}

	collection := s.db.Collection(collectionName)

	// 如果 data 是 []byte，尝试解析为 JSON
	var doc interface{}
	switch v := data.(type) {
	case []byte:
		// 尝试解析 JSON
		var jsonDoc interface{}
		if err := json.Unmarshal(v, &jsonDoc); err != nil {
			// 如果解析失败，将原始字节作为字符串存储
			doc = bson.M{
				"raw_data":   string(v),
				"updated_at": time.Now(),
			}
		} else {
			// 解析成功，添加时间戳
			if docMap, ok := jsonDoc.(map[string]interface{}); ok {
				docMap["updated_at"] = time.Now()
				doc = docMap
			} else {
				doc = bson.M{
					"data":       jsonDoc,
					"updated_at": time.Now(),
				}
			}
		}
	default:
		// 其他类型，直接使用
		if docMap, ok := v.(map[string]interface{}); ok {
			docMap["updated_at"] = time.Now()
			doc = docMap
		} else {
			doc = bson.M{
				"data":       v,
				"updated_at": time.Now(),
			}
		}
	}

	// 使用 upsert 操作
	opts := options.Update().SetUpsert(true)
	update := bson.M{"$set": doc}

	result, err := collection.UpdateOne(ctx, filter, update, opts)
	if err != nil {
		if s.logger != nil {
			s.logger.Error("failed to upsert data to MongoDB",
				zap.String("collection", collectionName),
				zap.Error(err),
			)
		}
		return fmt.Errorf("failed to upsert data: %w", err)
	}

	if s.logger != nil {
		s.logger.Info("data upserted to MongoDB successfully",
			zap.String("collection", collectionName),
			zap.Int64("matched_count", result.MatchedCount),
			zap.Int64("modified_count", result.ModifiedCount),
			zap.Any("upserted_id", result.UpsertedID),
		)
	}

	return nil
}
