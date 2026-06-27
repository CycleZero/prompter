package prompt

import "prompter/model"

// RegionBiz 业务逻辑层 - 处理 PromptRegion 的业务规则
type RegionBiz struct {
	regionRepo *RegionRepo
}

func NewRegionBiz(regionRepo *RegionRepo) *RegionBiz {
	return &RegionBiz{regionRepo: regionRepo}
}

// Create 创建新类别
func (b *RegionBiz) Create(name, description string, sortOrder int) (*model.PromptRegion, error) {
	region := &model.PromptRegion{
		Name:        name,
		Description: description,
		SortOrder:   sortOrder,
	}
	if err := b.regionRepo.Create(region); err != nil {
		return nil, err
	}
	return region, nil
}

// GetByID 根据 ID 获取类别
func (b *RegionBiz) GetByID(id uint) (*model.PromptRegion, error) {
	return b.regionRepo.GetByID(id)
}

// List 获取所有类别列表
func (b *RegionBiz) List() ([]*model.PromptRegion, error) {
	return b.regionRepo.List()
}

// Update 更新类别 — 先查再改后保存
func (b *RegionBiz) Update(id uint, name, description string, sortOrder int) (*model.PromptRegion, error) {
	region, err := b.regionRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	region.Name = name
	region.Description = description
	region.SortOrder = sortOrder
	if err := b.regionRepo.Update(region); err != nil {
		return nil, err
	}
	return region, nil
}

// Delete 删除类别
func (b *RegionBiz) Delete(id uint) error {
	return b.regionRepo.Delete(id)
}
