package database

import (
	"context"
	"database/sql"
	"fmt"

	"go.mongodb.org/mongo-driver/mongo"
)

// GetMySQL 获取 MySQL 数据库连接
// 如果未启用或未初始化，返回错误
func GetMySQL() (*sql.DB, error) {
	dbs := GetGlobal()
	if dbs == nil {
		return nil, fmt.Errorf("database manager not initialized")
	}
	if dbs.MySQL == nil {
		return nil, fmt.Errorf("MySQL is not enabled or not connected")
	}
	return dbs.MySQL, nil
}

// GetPostgreSQL 获取 PostgreSQL 数据库连接
// 如果未启用或未初始化，返回错误
func GetPostgreSQL() (*sql.DB, error) {
	dbs := GetGlobal()
	if dbs == nil {
		return nil, fmt.Errorf("database manager not initialized")
	}
	if dbs.PostgreSQL == nil {
		return nil, fmt.Errorf("PostgreSQL is not enabled or not connected")
	}
	return dbs.PostgreSQL, nil
}

// GetMongoDB 获取 MongoDB 数据库连接
// 如果未启用或未初始化，返回错误
func GetMongoDB() (*mongo.Database, error) {
	dbs := GetGlobal()
	if dbs == nil {
		return nil, fmt.Errorf("database manager not initialized")
	}
	if dbs.MongoDB == nil {
		return nil, fmt.Errorf("MongoDB is not enabled or not connected")
	}
	return dbs.MongoDB, nil
}

// MustGetMySQL 获取 MySQL 数据库连接，如果失败则 panic
// 仅在确定数据库已初始化时使用
func MustGetMySQL() *sql.DB {
	db, err := GetMySQL()
	if err != nil {
		panic(err)
	}
	return db
}

// MustGetPostgreSQL 获取 PostgreSQL 数据库连接，如果失败则 panic
// 仅在确定数据库已初始化时使用
func MustGetPostgreSQL() *sql.DB {
	db, err := GetPostgreSQL()
	if err != nil {
		panic(err)
	}
	return db
}

// MustGetMongoDB 获取 MongoDB 数据库连接，如果失败则 panic
// 仅在确定数据库已初始化时使用
func MustGetMongoDB() *mongo.Database {
	db, err := GetMongoDB()
	if err != nil {
		panic(err)
	}
	return db
}

// PingAll 检查所有已启用数据库的连接状态
func PingAll(ctx context.Context) error {
	dbs := GetGlobal()
	if dbs == nil {
		return fmt.Errorf("database manager not initialized")
	}
	return dbs.Ping(ctx)
}

