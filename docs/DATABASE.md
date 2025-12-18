# 数据库使用指南

本项目支持同时连接 MySQL、PostgreSQL 和 MongoDB 三种数据库。所有数据库连接都是可选的，您可以根据需要启用相应的数据库。

## 配置数据库

在 `configs/config.yaml` 中配置数据库连接信息：

```yaml
database:
  mysql:
    enabled: true   # 启用 MySQL
    host: localhost
    port: 3306
    username: root
    password: your_password
    database: keepa
    charset: utf8mb4
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m
    conn_max_idle_time: 10m

  postgresql:
    enabled: true   # 启用 PostgreSQL
    host: localhost
    port: 5432
    username: postgres
    password: your_password
    database: keepa
    sslmode: disable
    max_open_conns: 25
    max_idle_conns: 5
    conn_max_lifetime: 5m
    conn_max_idle_time: 10m

  mongodb:
    enabled: true   # 启用 MongoDB
    uri: mongodb://localhost:27017
    database: keepa
    auth_source: admin
    username: ""    # 可选，如果 URI 中已包含则留空
    password: ""    # 可选，如果 URI 中已包含则留空
    replica_set: ""
    max_pool_size: 100
    min_pool_size: 10
    max_idle_time: 30m
```

### 环境变量配置

也可以通过环境变量配置，变量名格式为 `KEEPA_DATABASE_<TYPE>_<FIELD>`：

```bash
export KEEPA_DATABASE_MYSQL_ENABLED=true
export KEEPA_DATABASE_MYSQL_HOST=localhost
export KEEPA_DATABASE_MYSQL_USERNAME=root
export KEEPA_DATABASE_MYSQL_PASSWORD=secret
```

## 在任务中使用数据库

### 方法 1: 使用全局数据库管理器（推荐）

项目提供了全局数据库管理器，您可以在任何任务中方便地获取数据库连接：

```go
package tasks

import (
    "context"
    "database/sql"
    "fmt"
    
    "keepa/internal/database"
    "keepa/internal/task"
)

type MyTask struct {
    name    string
    enabled bool
}

func (t *MyTask) Run(ctx context.Context) error {
    // 获取 MySQL 连接
    mysqlDB, err := database.GetMySQL()
    if err != nil {
        return fmt.Errorf("MySQL not available: %w", err)
    }
    
    var count int
    err = mysqlDB.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
    if err != nil {
        return fmt.Errorf("query failed: %w", err)
    }
    
    fmt.Printf("Total users: %d\n", count)
    
    // 获取 PostgreSQL 连接
    pgDB, err := database.GetPostgreSQL()
    if err == nil {
        // 使用 PostgreSQL...
    }
    
    // 获取 MongoDB 连接
    mongoDB, err := database.GetMongoDB()
    if err == nil {
        // 使用 MongoDB...
        collection := mongoDB.Collection("users")
        // ...
    }
    
    return nil
}
```

### 方法 2: 使用 Must 函数（仅在确定数据库已启用时使用）

如果确定数据库已启用，可以使用 `Must` 函数，它会在数据库未启用时 panic：

```go
func (t *MyTask) Run(ctx context.Context) error {
    // 直接获取连接，如果未启用会 panic
    mysqlDB := database.MustGetMySQL()
    pgDB := database.MustGetPostgreSQL()
    mongoDB := database.MustGetMongoDB()
    
    // 使用数据库...
    return nil
}
```

## 数据库操作示例

### MySQL 操作示例

```go
func (t *MyTask) Run(ctx context.Context) error {
    db, err := database.GetMySQL()
    if err != nil {
        return err
    }
    
    // 查询单行
    var id int
    var name string
    err = db.QueryRowContext(ctx, 
        "SELECT id, name FROM users WHERE id = ?", 
        1,
    ).Scan(&id, &name)
    if err != nil {
        return err
    }
    
    // 查询多行
    rows, err := db.QueryContext(ctx, "SELECT id, name FROM users LIMIT 10")
    if err != nil {
        return err
    }
    defer rows.Close()
    
    for rows.Next() {
        var id int
        var name string
        if err := rows.Scan(&id, &name); err != nil {
            return err
        }
        fmt.Printf("ID: %d, Name: %s\n", id, name)
    }
    
    // 执行更新
    result, err := db.ExecContext(ctx,
        "UPDATE users SET name = ? WHERE id = ?",
        "New Name", 1,
    )
    if err != nil {
        return err
    }
    
    affected, _ := result.RowsAffected()
    fmt.Printf("Updated %d rows\n", affected)
    
    return nil
}
```

### PostgreSQL 操作示例

```go
func (t *MyTask) Run(ctx context.Context) error {
    db, err := database.GetPostgreSQL()
    if err != nil {
        return err
    }
    
    // PostgreSQL 使用 $1, $2 作为占位符
    var id int
    var name string
    err = db.QueryRowContext(ctx,
        "SELECT id, name FROM users WHERE id = $1",
        1,
    ).Scan(&id, &name)
    if err != nil {
        return err
    }
    
    // 事务示例
    tx, err := db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    _, err = tx.ExecContext(ctx, "INSERT INTO users (name) VALUES ($1)", "New User")
    if err != nil {
        return err
    }
    
    return tx.Commit()
}
```

### MongoDB 操作示例

```go
import (
    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo/options"
)

func (t *MyTask) Run(ctx context.Context) error {
    db, err := database.GetMongoDB()
    if err != nil {
        return err
    }
    
    collection := db.Collection("users")
    
    // 插入文档
    doc := bson.M{
        "name": "John Doe",
        "email": "john@example.com",
        "age": 30,
    }
    _, err = collection.InsertOne(ctx, doc)
    if err != nil {
        return err
    }
    
    // 查询文档
    var result bson.M
    err = collection.FindOne(ctx, bson.M{"name": "John Doe"}).Decode(&result)
    if err != nil {
        return err
    }
    
    // 更新文档
    filter := bson.M{"name": "John Doe"}
    update := bson.M{"$set": bson.M{"age": 31}}
    _, err = collection.UpdateOne(ctx, filter, update)
    if err != nil {
        return err
    }
    
    // 查询多个文档
    cursor, err := collection.Find(ctx, bson.M{"age": bson.M{"$gte": 18}})
    if err != nil {
        return err
    }
    defer cursor.Close(ctx)
    
    for cursor.Next(ctx) {
        var user bson.M
        if err := cursor.Decode(&user); err != nil {
            return err
        }
        fmt.Printf("User: %v\n", user)
    }
    
    return nil
}
```

## 连接池配置

### MySQL/PostgreSQL 连接池

- `max_open_conns`: 最大打开连接数
- `max_idle_conns`: 最大空闲连接数
- `conn_max_lifetime`: 连接最大存活时间（超过此时间后连接会被关闭）
- `conn_max_idle_time`: 连接最大空闲时间（超过此时间后空闲连接会被关闭）

### MongoDB 连接池

- `max_pool_size`: 最大连接池大小
- `min_pool_size`: 最小连接池大小
- `max_idle_time`: 连接最大空闲时间

## 健康检查

可以使用 `Ping` 方法检查所有已启用数据库的连接状态：

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

if err := database.PingAll(ctx); err != nil {
    log.Printf("Database health check failed: %v", err)
}
```

## 注意事项

1. **连接管理**: 数据库连接在程序启动时建立，退出时自动关闭。无需手动管理连接的生命周期。

2. **上下文使用**: 所有数据库操作都应该使用 `context.Context`，这样可以：
   - 支持超时控制
   - 支持取消操作
   - 在任务执行时正确处理上下文

3. **错误处理**: 始终检查数据库操作返回的错误，并进行适当的处理。

4. **资源清理**: 
   - SQL 查询结果（`*sql.Rows`）使用后应该调用 `Close()`
   - MongoDB 游标（`*mongo.Cursor`）使用后应该调用 `Close(ctx)`

5. **事务处理**: 对于需要原子性的操作，使用数据库事务。确保在发生错误时回滚事务。

6. **连接池**: 合理配置连接池参数，避免连接过多或过少。

## 完整示例

参考 `internal/tasks/db_example.go` 查看完整的数据库使用示例。

