package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"gin-demo/config"
	"gin-demo/internal/handler"
	"gin-demo/internal/seed"
	"gin-demo/internal/service"
	"gin-demo/pkg/db"

	"github.com/gin-gonic/gin"
)

func main() {
	c := config.Get()

	if err := db.InitMySQL(); err != nil {
		log.Fatalf("MySQL 初始化失败: %v", err)
	}
	if err := db.InitRedis(); err != nil {
		log.Printf("警告: Redis 初始化失败(将仅影响缓存): %v", err)
	}
	if err := seed.Run(); err != nil {
		log.Fatalf("种子数据初始化失败: %v", err)
	}

	// 启动超时扫描（BR-19）
	service.StartTimeoutScanner()

	r := gin.Default()
	handler.SetupRoutes(r)

	// 托管前端构建产物（若存在）
	serveFrontend(r, c.WebDist)

	addr := ":" + c.ServerPort
	log.Printf("服务启动于 %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

func serveFrontend(r *gin.Engine, dist string) {
	if info, err := os.Stat(dist); err != nil || !info.IsDir() {
		log.Printf("前端构建目录 %s 不存在，跳过静态托管", dist)
		return
	}
	r.Static("/assets", filepath.Join(dist, "assets"))
	r.NoRoute(func(c *gin.Context) {
		index := filepath.Join(dist, "index.html")
		if _, err := os.Stat(index); err == nil {
			c.File(index)
		} else {
			c.JSON(http.StatusNotFound, gin.H{"code": "NOT_FOUND", "message": "接口或页面不存在"})
		}
	})
}
