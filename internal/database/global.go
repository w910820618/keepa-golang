package database

import (
	"sync"
)

var (
	globalDBs *Databases
	mu        sync.RWMutex
)

// SetGlobal 设置全局数据库管理器
func SetGlobal(dbs *Databases) {
	mu.Lock()
	defer mu.Unlock()
	globalDBs = dbs
}

// GetGlobal 获取全局数据库管理器
func GetGlobal() *Databases {
	mu.RLock()
	defer mu.RUnlock()
	return globalDBs
}
