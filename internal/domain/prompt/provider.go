package prompt

import "github.com/google/wire"

// ProviderSet 依赖注入集合，注册本模块的所有构造函数
var ProviderSet = wire.NewSet(
	NewRegionRepo,
	NewSliceRepo,
	NewRecordRepo,
	NewDraftRepo,
	NewSliceTypeRepo,
	NewRegionBiz,
	NewSliceBiz,
	NewDraftBiz,
	NewRecordBiz,
	NewSliceTypeBiz,
	NewSearchBiz,
	NewPromptService,
)
