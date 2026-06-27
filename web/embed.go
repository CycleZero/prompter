package web

import "embed"

// DistFS 嵌入前端构建产物，供 Gin 静态文件服务使用
//
//go:embed dist/*
var DistFS embed.FS
