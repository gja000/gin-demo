package clock

import (
	"sync"
	"time"

	"gin-demo/config"
)

// Now 返回当前时间，支持系统时间注入（用于演示/测试超时分支）
var (
	mu      sync.RWMutex
	injected time.Time
	hasInj  bool
)

func init() {
	c := config.Get()
	if c.InjectNow != "" {
		if t, err := time.Parse(time.RFC3339, c.InjectNow); err == nil {
			mu.Lock()
			injected = t
			hasInj = true
			mu.Unlock()
		}
	}
}

// Now 取得当前时间
func Now() time.Time {
	mu.RLock()
	defer mu.RUnlock()
	if hasInj {
		return injected
	}
	return time.Now()
}

// SetInjected 注入时间（测试/演示用）
func SetInjected(t time.Time) {
	mu.Lock()
	defer mu.Unlock()
	injected = t
	hasInj = true
}

// ClearInjected 清除时间注入
func ClearInjected() {
	mu.Lock()
	defer mu.Unlock()
	hasInj = false
}
