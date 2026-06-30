package router

import (
	"prompter/internal/domain"
	"prompter/internal/router/middleware"

	"github.com/gin-gonic/gin"
)

// RegisterFunc 路由注册函数类型
type RegisterFunc func(root gin.IRouter, serviceHub *domain.ServiceHub)

// NewRegisterFunc 创建路由注册函数
func NewRegisterFunc() RegisterFunc {
	return RegisterRouter
}

// RegisterRouter 注册所有路由
func RegisterRouter(root gin.IRouter, serviceHub *domain.ServiceHub) {
	if !middleware.IsMiddleWireRegisterFinished {
		panic("中间件注册未完成")
	}

	// 全局中间件
	root.Use(middleware.CORS())
	root.Use(middleware.AddMetaData())
	// 前端静态文件中间件（必须在路由注册前，放行 /api /debug，拦截其余请求）
	root.Use(middleware.FrontendFileHandler())

	// 注册业务路由
	api := root.Group("/api")
	RegisterDemoRouter(api, serviceHub)
	RegisterPromptRouter(api, serviceHub)
}

// RegisterDemoRouter 注册 Demo 模块路由
func RegisterDemoRouter(api gin.IRouter, hub *domain.ServiceHub) {
	demo := api.Group("/demo")
	{
		demo.POST("", hub.DemoService.Create)
		demo.GET("", hub.DemoService.List)
		demo.GET("/:id", hub.DemoService.GetByID)
		demo.PUT("/:id", hub.DemoService.Update)
		demo.DELETE("/:id", hub.DemoService.Delete)
	}
}

// RegisterPromptRouter 注册 Prompt 模块路由
func RegisterPromptRouter(api gin.IRouter, hub *domain.ServiceHub) {
	// Region 类别管理
	regions := api.Group("/regions")
	{
		regions.POST("", hub.PromptService.CreateRegion)
		regions.GET("", hub.PromptService.ListRegions)
		regions.GET("/:id", hub.PromptService.GetRegion)
		regions.PUT("/:id", hub.PromptService.UpdateRegion)
		regions.DELETE("/:id", hub.PromptService.DeleteRegion)
	}

	// Slice 提示词块管理
	slices := api.Group("/slices")
	{
		slices.POST("", hub.PromptService.CreateSlice)
		slices.GET("", hub.PromptService.ListSlices)
		slices.GET("/search", hub.PromptService.SearchSlices)
		slices.GET("/:id", hub.PromptService.GetSlice)
		slices.PUT("/:id", hub.PromptService.UpdateSlice)
		slices.DELETE("/:id", hub.PromptService.DeleteSlice)
	}

	// Active Prompt 活动 Prompt
	api.GET("/active-prompt", hub.PromptService.GetActivePrompt)
	api.PUT("/active-prompt", hub.PromptService.UpdateActivePrompt)

	// Record 已保存记录
	api.POST("/records/:uuid", hub.PromptService.PersistRecord)
	api.GET("/records", hub.PromptService.ListRecords)
	api.GET("/records/:id", hub.PromptService.GetRecord)
	api.DELETE("/records/:id", hub.PromptService.DeleteRecord)

	// Combo Tree
	api.GET("/combo/tree", hub.PromptService.GetComboTree)

	// SliceType 语义分类树
	api.GET("/slice-types", hub.PromptService.GetSliceTypeTree)
}
