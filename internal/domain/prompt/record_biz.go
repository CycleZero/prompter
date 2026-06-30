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

// PersistFromActive 核心方法 — ComfyUI 调用，将活动 Prompt 持久化为 Record
// 流程：
//  1. 幂等检查 — 已存在则直接返回
//  2. 从 Redis 获取活动 Prompt（Redis 不可用时回源 MySQL）
//  3. 校验活动 Prompt 有效性（至少一个 Region 有至少一个 Slice）
//  4. 按 Region.SortOrder 对 Region 排序，Region 内按 Slice.SortOrder 排序
//  5. 遍历每个 Region → 每个 Slice：
//     a) 从 DB 查询 Slice 原文
//     b) 解析最终内容（CustomText 优先，否则用原文）
//     c) 构建 AssemblyItem 用于最终 Prompt 拼接
//     d) 构建 PromptRecordRegionSlice 用于持久化
//  6. 调用 AssemblePrompt 生成 FullContent
//  7. 构建 regionPayload 列表（RecordRegion + RecordRegionSlice × N + RegionSlice 回写）
//  8. 事务写入数据库
//  9. 返回创建的 Record
func (b *RecordBiz) PersistFromActive(uuid string) (*model.PromptRecord, error) {
	// 步骤 1：幂等检查 — 已存在则直接返回
	existing, err := b.recordRepo.GetByExternalID(uuid)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// 步骤 2：获取活动 Prompt
	active, err := b.draftRepo.GetActive()
	if err != nil {
		return nil, err
	}

	// 步骤 3：校验活动 Prompt 有效性
	if active == nil || len(active.Regions) == 0 {
		return nil, ErrNoActivePrompt
	}

	// 步骤 4：按 Region.SortOrder → Slice.SortOrder 排序
	// 组装顺序完全由用户拖拽决定：Region 的 SortOrder 决定 Region 先后，
	// Slice 的 SortOrder 决定 Region 内先后
	regions := make([]ActivePromptRegionDTO, len(active.Regions))
	copy(regions, active.Regions)
	sort.SliceStable(regions, func(i, j int) bool { return regions[i].SortOrder < regions[j].SortOrder })

	// 用于最终 Prompt 拼接的片段列表（全局顺序）
	items := make([]AssemblyItem, 0)
	// 用于持久化的 Region 载荷列表
	payloads := make([]*regionPayload, 0, len(regions))
	globalOrder := 0

	// 步骤 5：遍历每个 Region → 每个 Slice
	for _, region := range regions {
		slices := make([]ActiveSliceDTO, len(region.Slices))
		copy(slices, region.Slices)
		sort.SliceStable(slices, func(i, j int) bool { return slices[i].SortOrder < slices[j].SortOrder })

		payload := &regionPayload{
			Region: model.PromptRecordRegion{
				RegionID:  region.RegionID,
				SortOrder: region.SortOrder,
			},
		}

		for _, dto := range slices {
			// 5a. 从 DB 查询 Slice 原文
			slice, err := b.sliceRepo.GetByID(dto.SliceID)
			if err != nil {
				return nil, err
			}

			// 5b. 解析最终内容（CustomText 优先，否则用原文）
			resolved := slice.Content
			if dto.CustomText != nil {
				resolved = *dto.CustomText
			}

			// 5c. 构建 AssemblyItem 用于最终 Prompt 拼接
			items = append(items, AssemblyItem{
				Content:   resolved,
				SortOrder: globalOrder,
			})

			// 5d. 构建 PromptRecordRegionSlice 用于持久化
			payload.Slices = append(payload.Slices, model.PromptRecordRegionSlice{
				SliceID:    dto.SliceID,
				SortOrder:  dto.SortOrder,
				CustomText: dto.CustomText,
			})
			// 回写 Region-Slice 关联，使 combo tree 能反映使用关系
			payload.RegionSliceEntries = append(payload.RegionSliceEntries, model.PromptRegionSlice{
				RegionID:  region.RegionID,
				SliceID:   dto.SliceID,
				SortOrder: dto.SortOrder,
			})
			globalOrder++
		}

		payloads = append(payloads, payload)
	}

	// 步骤 6：组装完整 Prompt
	fullContent := AssemblePrompt(items)

	// 步骤 7：构建记录
	record := &model.PromptRecord{
		ExternalID:  uuid,
		Title:       active.Title,
		FullContent: fullContent,
	}

	// 步骤 8：事务写入数据库（Record → RecordRegion → RecordRegionSlice 三层结构）
	if err := b.recordRepo.CreateWithRegions(record, payloads); err != nil {
		return nil, err
	}

	// 回写 Region-Slice 关联，使 combo tree 能反映使用关系
	// 从 payloads 收集所有 RegionSliceEntries
	regionSliceEntries := make([]model.PromptRegionSlice, 0, globalOrder)
	for _, p := range payloads {
		regionSliceEntries = append(regionSliceEntries, p.RegionSliceEntries...)
	}
	if err := b.recordRepo.UpsertRegionSlices(regionSliceEntries); err != nil {
		return nil, err
	}

	// 步骤 9：返回创建的 Record
	return record, nil
}

// GetByID 根据 ID 获取记录详情
func (b *RecordBiz) GetByID(id uint) (*model.PromptRecord, error) {
	return b.recordRepo.GetByID(id)
}

// GetRegionsByRecordID 获取某条 Record 的所有 Region（按排序字段升序）
func (b *RecordBiz) GetRegionsByRecordID(recordID uint) ([]*model.PromptRecordRegion, error) {
	return b.recordRepo.GetRegionsByRecordID(recordID)
}

// GetRegionSlices 获取某个 Region 下的所有 Slice（按排序字段升序）
func (b *RecordBiz) GetRegionSlices(recordRegionID uint) ([]*model.PromptRecordRegionSlice, error) {
	return b.recordRepo.GetRegionSlices(recordRegionID)
}

// List 分页获取记录列表
func (b *RecordBiz) List(page, pageSize int) ([]*model.PromptRecord, int64, error) {
	return b.recordRepo.List(page, pageSize)
}

// Delete 删除记录
func (b *RecordBiz) Delete(id uint) error {
	return b.recordRepo.Delete(id)
}
