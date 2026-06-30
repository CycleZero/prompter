package infra

import (
	"github.com/google/wire"
	"github.com/spf13/viper"
)

var ProviderSet = wire.NewSet(
	NewData,
	NewRedisClient,
	NewCustomRedisClient,
	GetDB,
	NewSearchIndex,
	GetSearchIndexPath,
)

// GetSearchIndexPath 从配置读取搜索索引路径（默认 data/search.bleve）
func GetSearchIndexPath(vc *viper.Viper) string {
	if p := vc.GetString("app.search_index_path"); p != "" {
		return p
	}
	return "data/search.bleve"
}
