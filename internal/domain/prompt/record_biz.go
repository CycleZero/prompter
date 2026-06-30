package prompt

import (
	"errors"
	"sort"
	"prompter/model"

	"gorm.io/gorm"
)

var (
	// ErrNoActivePrompt 当前没有可用的活动 Prompt
	ErrNoActivePrompt = errors.New("当前没有可用的活动Prompt")
)

// RecordBiz 业务逻辑层 - 处理 PromptRecord 的核心业务逻辑
type RecordBiz struct {
	recordRepo *RecordRepo
	sliceRepo  *SliceRepo
	draftRepo  *DraftRepo
}

func NewRecordBiz(recordRepo *RecordRepo, sliceRepo *SliceRepo, draftRepo *DraftRepo) *RecordBiz {
	return &RecordBiz{
		recordRepo: recordRepo,
		sliceRepo:  sliceRepo,
		draftRepo:  draftRepo,
	}
}

// PersistFromActive 核心方法 — ComfyUI 调用，将活动 Prompt 持久化为记录
// 流程:
//  1. 检查 recordRepo.GetByExternalID(uuid) — 若存在，直接返回（幂等）
//  2. draftRepo.GetActive() — 获取活动 Prompt
//  3. 若 active 为空或 slices 空 → 返回 ErrNoActivePrompt
//  4. 遍历 active.Slices:
//     - 查 sliceRepo.GetByID(sliceID) 获取原始内容
//     - 解析 Content = customText 或 slice.Content
//     - 构建 AssemblyItem{Content: resolved, SortOrder: sortOrder}
//  5. 调用 AssemblePrompt(items) → fullContent
//  6. 构建 model.PromptRecord{ExternalID: uuid, Title: active.Title, FullContent: fullContent}
//  7. 构建 []model.PromptRecordSlice (每个 active slice 映射)
//  8. 调用 recordRepo.CreateWithSlices(record, recordSlices)
//  9. 返回 record, nil
func (b *RecordBiz) PersistFromActive(uuid string) (*model.PromptRecord, error) {
	// 1. 幂等检查 — 已存在则直接返回
	existing, err := b.recordRepo.GetByExternalID(uuid)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// 2. 获取活动 Prompt
	active, err := b.draftRepo.GetActive()
	if err != nil {
		return nil, err
	}

	// 3. 校验活动 Prompt 有效性
	if active == nil || len(active.Regions) == 0 {
		return nil, ErrNoActivePrompt
	}

	// 4. 按 Region.SortOrder → Slice.SortOrder 遍历构建
	// 组装顺序完全由用户拖拽决定：Region 的 SortOrder 决定 Region 先后，
	// Slice 的 SortOrder 决定 Region 内先后
	regions := make([]ActivePromptRegionDTO, len(active.Regions))
	copy(regions, active.Regions)
	sort.SliceStable(regions, func(i, j int) bool { return regions[i].SortOrder < regions[j].SortOrder })

	items := make([]AssemblyItem, 0)
	recordSlices := make([]model.PromptRecordSlice, 0)
	regionSlices := make([]model.PromptRegionSlice, 0) // 回写 Region-Slice 关联
	globalOrder := 0

	for _, region := range regions {
		slices := make([]ActiveSliceDTO, len(region.Slices))
		copy(slices, region.Slices)
		sort.SliceStable(slices, func(i, j int) bool { return slices[i].SortOrder < slices[j].SortOrder })

		for _, dto := range slices {
			slice, err := b.sliceRepo.GetByID(dto.SliceID)
			if err != nil {
				return nil, err
			}

			resolved := slice.Content
			if dto.CustomText != nil {
				resolved = *dto.CustomText
			}

			items = append(items, AssemblyItem{
				Content:   resolved,
				SortOrder: globalOrder,
			})

			recordSlices = append(recordSlices, model.PromptRecordSlice{
				SliceID:         dto.SliceID,
				RegionID:        region.RegionID,
				SortOrder:       globalOrder,
				RegionSortOrder: region.SortOrder,
				CustomText:      dto.CustomText,
			})
			regionSlices = append(regionSlices, model.PromptRegionSlice{
				RegionID:  region.RegionID,
				SliceID:   dto.SliceID,
				SortOrder: dto.SortOrder,
			})
			globalOrder++
		}
	}

	// 5. 组装完整 Prompt
	fullContent := AssemblePrompt(items)

	// 6. 构建记录
	record := &model.PromptRecord{
		ExternalID:  uuid,
		Title:       active.Title,
		FullContent: fullContent,
	}

	// 7-8. 持久化记录及切片关联
	if err := b.recordRepo.CreateWithSlices(record, recordSlices); err != nil {
		return nil, err
	}
	// 回写 Region-Slice 关联，使 combo tree 能反映使用关系
	if err := b.recordRepo.UpsertRegionSlices(regionSlices); err != nil {
		return nil, err
	}

	// 9. 返回记录
	return record, nil
}

// GetByID 根据 ID 获取记录详情
func (b *RecordBiz) GetByID(id uint) (*model.PromptRecord, error) {
	return b.recordRepo.GetByID(id)
}

// GetSlices 获取记录的所有切片
func (b *RecordBiz) GetSlices(recordID uint) ([]model.PromptRecordSlice, error) {
	return b.recordRepo.GetSlicesByRecordID(recordID)
}

// List 分页获取记录列表
func (b *RecordBiz) List(page, pageSize int) ([]*model.PromptRecord, int64, error) {
	return b.recordRepo.List(page, pageSize)
}

// Delete 删除记录
func (b *RecordBiz) Delete(id uint) error {
	return b.recordRepo.Delete(id)
}
