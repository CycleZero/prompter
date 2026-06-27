package prompt

import (
	"sort"
	"strings"
)

// AssemblyItem 表示一个待组装的 Prompt 片段
// Content 应为已解析的最终文本（CustomText 覆盖后或 Slice 原文）
type AssemblyItem struct {
	Content   string
	SortOrder int
}

// AssemblePrompt 将多个片段按 SortOrder 排序后以 ", " 拼接
// 使用稳定排序以保证相同 SortOrder 时保持原始顺序
func AssemblePrompt(items []AssemblyItem) string {
	if len(items) == 0 {
		return ""
	}

	sorted := make([]AssemblyItem, len(items))
	copy(sorted, items)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].SortOrder < sorted[j].SortOrder
	})

	parts := make([]string, len(sorted))
	for i, item := range sorted {
		parts[i] = item.Content
	}
	return strings.Join(parts, ", ")
}
