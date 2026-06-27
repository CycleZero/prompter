# AGENTS.md — Prompter 项目规范

## 语言规范

所有日志、注释、错误定义必须使用中文。日志调用、注释文本、错误消息字符串均不得使用英文。

### 适用范围
- Go 代码中的注释（行注释 `//` 和块注释 `/* */`）
- 错误变量定义中的错误消息（`errors.New("...")`、`fmt.Errorf("...")`）
- 日志输出内容（`logger.Error("...")`、`logger.Info("...")` 等）
- Swagger/OpenAPI 注解中的描述文本
- API 响应中的 `error`/`message` 字段值

### 不适用范围
- Go 关键字、类型名、变量名、函数名（遵循 Go 语言标识符规范，使用英文）
- GORM 模型字段的 `comment` 标签（已为中文）
- JSON 字段名（使用 snake_case 英文）
- 依赖库方法名、包路径

### 示例

✅ 正确：
```go
// CreateRegion 创建新类别
// @Summary 创建类别
// @Tags prompt
// @Accept json
// @Produce json
// @Param request body CreateRegionRequest true "创建请求"
// @Success 200 {object} RegionResponse
// @Router /api/regions [post]
func (s *PromptService) CreateRegion(c *gin.Context) {
    // 解析请求参数
    var req CreateRegionRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
        return
    }
    // ...
}

// ErrNoActivePrompt 当前没有可用的活动 Prompt
var ErrNoActivePrompt = errors.New("当前没有可用的活动Prompt")
```

❌ 错误：
```go
// CreateRegion creates a new region
// @Summary Create region
// ...
var ErrNoActivePrompt = errors.New("no active prompt available")
```
