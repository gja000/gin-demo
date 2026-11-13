package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config 全局配置（敏感值仅从环境变量/ .env 读取，禁止硬编码明文写入提交）
type Config struct {
	ServerPort    string
	MySQLDSN      string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	JWTSecret     string
	VehicleSecret string // 车端令牌签名密钥，与用户令牌隔离
	// 超时阈值（秒），可配置用于演示缩短
	TimeoutSeconds int
	// 超时扫描周期（秒）
	ScanIntervalSeconds int
	// 系统时间注入：RFC3339，演示/测试用，未设置则使用真实时间
	InjectNow string
	// 前端构建产物目录（由后端静态托管）
	WebDist string
	// 开发环境数据重置开关（仅本地联调使用，生产务必关闭）
	EnableReset bool
}

var C *Config

func loadEnv() {
	_ = godotenv.Load()
}

func Get() *Config {
	if C != nil {
		return C
	}
	loadEnv()
	timeout := 60
	if v, err := strconv.Atoi(os.Getenv("TIMEOUT_SECONDS")); err == nil && v > 0 {
		timeout = v
	}
	scan := 5
	if v, err := strconv.Atoi(os.Getenv("SCAN_INTERVAL_SECONDS")); err == nil && v > 0 {
		scan = v
	}
	redisDB := 0
	if v, err := strconv.Atoi(os.Getenv("REDIS_DB")); err == nil {
		redisDB = v
	}
	C = &Config{
		ServerPort:         getEnv("SERVER_PORT", "8080"),
		MySQLDSN:           getEnv("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/car?parseTime=True&charset=utf8mb4&loc=Local"),
		RedisAddr:          getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:            redisDB,
		JWTSecret:          getEnv("JWT_SECRET", "dev-user-secret-change-me"),
		VehicleSecret:      getEnv("VEHICLE_SECRET", "dev-vehicle-secret-change-me"),
		TimeoutSeconds:     timeout,
		ScanIntervalSeconds: scan,
		InjectNow:          os.Getenv("APP_NOW"),
		WebDist:            getEnv("WEB_DIST", "frontend/dist"),
		EnableReset:        os.Getenv("ENABLE_RESET") == "1",
	}
	return C
}

func getEnv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

// TimeoutDuration 返回超时阈值
func (c *Config) TimeoutDuration() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}
