package prompt

import (
	"fmt"
	"sort"
	"strconv"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
	"prompter/infra"
	"prompter/model"
)

// SearchBiz 搜索业务逻辑层 — 封装 Bleve 全文搜索
type SearchBiz struct {
	searchIndex *infra.SearchIndex
	sliceRepo   *SliceRepo
}

// NewSearchBiz 创建搜索业务实例
func NewSearchBiz(searchIndex *infra.SearchIndex, sliceRepo *SliceRepo) *SearchBiz {
	return &SearchBiz{searchIndex: searchIndex, sliceRepo: sliceRepo}
}

// Search 执行全文搜索
//   - q: 搜索关键词
//   - typeID: 可选，按分类过滤（nil 表示不限分类）
//   - page, pageSize: 分页参数
func (b *SearchBiz) Search(q string, typeID *uint, page, pageSize int) (*SearchSlicesResponse, error) {
	// 构建 Bleve 查询
	var searchQuery query.Query

	if typeID != nil {
		// 同时匹配关键词和分类
		mustQuery := bleve.NewBooleanQuery()
		mustQuery.AddMust(bleve.NewMatchQuery(q))
		typeQuery := bleve.NewTermQuery(strconv.FormatUint(uint64(*typeID), 10))
		typeQuery.SetField("type_id")
		mustQuery.AddMust(typeQuery)
		searchQuery = mustQuery
	} else {
		searchQuery = bleve.NewMatchQuery(q)
	}

	searchReq := bleve.NewSearchRequest(searchQuery)
	searchReq.Size = pageSize
	searchReq.From = (page - 1) * pageSize
	searchReq.Fields = []string{"*"}
	searchReq.SortBy([]string{"-_score"})

	result, err := b.searchIndex.Search(searchReq)
	if err != nil {
		return nil, fmt.Errorf("搜索失败: %w", err)
	}

	// 提取命中的 Slice ID 列表
	sliceIDs := make([]uint, 0, len(result.Hits))
	scoreMap := make(map[uint]float64)
	for _, hit := range result.Hits {
		id, err := strconv.ParseUint(hit.ID, 10, 64)
		if err != nil {
			continue
		}
		sliceIDs = append(sliceIDs, uint(id))
		scoreMap[uint(id)] = hit.Score
	}

	// 回源 DB 补全字段
	slices, err := b.sliceRepo.GetByIDs(sliceIDs)
	if err != nil {
		return nil, fmt.Errorf("查询 Slice 详情失败: %w", err)
	}

	// 按 score 降序排列结果
	sort.SliceStable(slices, func(i, j int) bool {
		return scoreMap[slices[i].ID] > scoreMap[slices[j].ID]
	})

	// 构建响应
	list := make([]*SearchSliceResponse, 0, len(slices))
	for _, sl := range slices {
		list = append(list, &SearchSliceResponse{
			ID:                sl.ID,
			Content:           sl.Content,
			TranslatedContent: sl.TranslatedContent,
			Score:             scoreMap[sl.ID],
		})
	}

	return &SearchSlicesResponse{
		List:  list,
		Total: int(result.Total),
		Page:  page,
	}, nil
}

// IndexSlice 索引单条 Slice（创建/更新时调用，异步执行）
func (b *SearchBiz) IndexSlice(sl *model.PromptSlice) {
	_ = b.searchIndex.Index(sl.ID, sl) // 忽略索引错误
}

// DeleteSlice 从索引中删除 Slice（异步执行）
func (b *SearchBiz) DeleteSlice(sliceID uint) {
	_ = b.searchIndex.Delete(sliceID) // 忽略索引错误
}

// RebuildIndex 全量重建搜索索引（启动时调用）
func (b *SearchBiz) RebuildIndex() error {
	slices, err := b.sliceRepo.ListAll()
	if err != nil {
		return fmt.Errorf("查询所有 Slice 失败: %w", err)
	}
	return b.searchIndex.RebuildAll(slices)
}
