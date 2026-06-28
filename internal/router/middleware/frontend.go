package middleware

import (
	"io"
	"io/fs"
	"net/http"
	"strings"

	"prompter/web"

	"github.com/gin-gonic/gin"
)

// FrontendFileHandler 前端静态文件处理中间件
// 拦截所有非 API 请求，从嵌入的 static FS 中查找对应文件：
// - /api/* → 放行给后端路由
// - / 或文件不存在 → 返回 index.html（SPA 回退）
// - 其他 → 直接返回静态文件
func FrontendFileHandler() gin.HandlerFunc {
	// 读取 index.html 到内存
	file, err := web.DistFS.Open("dist/index.html")
	if err != nil {
		return func(c *gin.Context) { c.Next() }
	}
	indexHTML, _ := io.ReadAll(file)
	file.Close()

	// 获取子文件系统用于 static file server
	subFS, err := fs.Sub(web.DistFS, "dist")
	if err != nil {
		return func(c *gin.Context) { c.Next() }
	}
	fileServer := http.FileServer(http.FS(subFS))

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		// 后端 API 路由 → 放行
		if strings.HasPrefix(path, "/api") ||
			strings.HasPrefix(path, "/debug") {
			c.Next()
			return
		}

		// SPA 回退：首页或文件不存在时返回 index.html
		if path == "/" || path == "/index.html" || !fileExists(subFS, path) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.Header("Cache-Control", "public, no-cache")
			c.String(http.StatusOK, string(indexHTML))
			c.Abort()
			return
		}

		// 静态资源
		fileServer.ServeHTTP(c.Writer, c.Request)
		c.Abort()
	}
}

// fileExists 检查文件是否存在于给定的文件系统中
func fileExists(fsys fs.FS, path string) bool {
	path = strings.TrimPrefix(path, "/")
	f, err := fsys.Open(path)
	if err != nil {
		return false
	}
	f.Close()
	return true
}
