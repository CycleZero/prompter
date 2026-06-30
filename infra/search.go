package infra

import (
	"fmt"
	"strconv"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/analysis/lang/cjk"
	"prompter/model"
)

// SearchIndex 全文搜索引擎封装，基于 Bleve
// 索引存储在磁盘目录中，启动时自动创建或加载
type SearchIndex struct {
	index bleve.Index
	path  string
}

// NewSearchIndex 创建或打开 Bleve 索引
//   - 若 data/search.bleve 目录不存在 → 创建新索引（CJK 分析器）
//   - 若目录已存在 → 直接打开已有索引
func NewSearchIndex(path string) (*SearchIndex, error) {
	idx, err := bleve.Open(path)
	if err == bleve.ErrorIndexPathDoesNotExist {
		// 新建索引：CJK 分析器支持中英文分词
		idxMapping := bleve.NewIndexMapping()
		docMapping := bleve.NewDocumentMapping()

		cjkField := bleve.NewTextFieldMapping()
		cjkField.Analyzer = cjk.AnalyzerName // CJKBigram："女孩"→"女""孩"

		docMapping.AddFieldMappingsAt("content", cjkField)
		docMapping.AddFieldMappingsAt("translated_content", cjkField)

		typeField := bleve.NewNumericFieldMapping()
		docMapping.AddFieldMappingsAt("type_id", typeField)

		idxMapping.AddDocumentMapping("slice", docMapping)
		idx, err = bleve.New(path, idxMapping)
		if err != nil {
			return nil, fmt.Errorf("创建 Bleve 索引失败: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("打开 Bleve 索引失败: %w", err)
	}
	return &SearchIndex{index: idx, path: path}, nil
}

// Index 索引单条 Slice（增量写入）
func (s *SearchIndex) Index(sliceID uint, sl *model.PromptSlice) error {
	doc := map[string]interface{}{
		"content":            sl.Content,
		"translated_content": sl.TranslatedContent,
		"type_id":            sl.TypeID,
	}
	return s.index.Index(strconv.FormatUint(uint64(sliceID), 10), doc)
}

// Delete 从索引中删除单条 Slice
func (s *SearchIndex) Delete(sliceID uint) error {
	return s.index.Delete(strconv.FormatUint(uint64(sliceID), 10))
}

// Search 执行全文搜索，返回 Bleve 原始搜索结果
func (s *SearchIndex) Search(req *bleve.SearchRequest) (*bleve.SearchResult, error) {
	return s.index.Search(req)
}

// RebuildAll 全量重建索引（启动时调用）
func (s *SearchIndex) RebuildAll(slices []*model.PromptSlice) error {
	batch := s.index.NewBatch()
	for _, sl := range slices {
		doc := map[string]interface{}{
			"content":            sl.Content,
			"translated_content": sl.TranslatedContent,
			"type_id":            sl.TypeID,
		}
		id := strconv.FormatUint(uint64(sl.ID), 10)
		if err := batch.Index(id, doc); err != nil {
			return fmt.Errorf("索引 Slice %d 失败: %w", sl.ID, err)
		}
	}
	return s.index.Batch(batch)
}

// Close 关闭索引，确保数据写入磁盘
func (s *SearchIndex) Close() error {
	return s.index.Close()
}
