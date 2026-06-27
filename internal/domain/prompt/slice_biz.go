package prompt

import "prompter/model"

// SliceBiz 业务逻辑层 - 处理 PromptSlice 的业务规则
type SliceBiz struct {
	sliceRepo *SliceRepo
}

func NewSliceBiz(sliceRepo *SliceRepo) *SliceBiz {
	return &SliceBiz{sliceRepo: sliceRepo}
}

// Create 创建提示词块并关联到指定类别
func (b *SliceBiz) Create(content, translatedContent, originLang, targetLang string, regionIDs []uint) (*model.PromptSlice, error) {
	slice := &model.PromptSlice{
		Content:           content,
		TranslatedContent: translatedContent,
		OriginLanguage:    model.Language(originLang),
		TargetLanguage:    model.Language(targetLang),
	}
	if err := b.sliceRepo.Create(slice, regionIDs); err != nil {
		return nil, err
	}
	return slice, nil
}

// GetByID 根据 ID 获取提示词块
func (b *SliceBiz) GetByID(id uint) (*model.PromptSlice, error) {
	return b.sliceRepo.GetByID(id)
}

// ListByRegion 查询指定类别下的所有提示词块
func (b *SliceBiz) ListByRegion(regionID uint) ([]*model.PromptSlice, error) {
	return b.sliceRepo.ListByRegion(regionID)
}

// ListByType 查询指定语义分类下的所有提示词块
func (b *SliceBiz) ListByType(typeID uint) ([]*model.PromptSlice, error) {
	return b.sliceRepo.ListByType(typeID)
}

// Update 更新提示词块 — 先查再改后保存
func (b *SliceBiz) Update(id uint, content, translatedContent, originLang, targetLang string) (*model.PromptSlice, error) {
	slice, err := b.sliceRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	slice.Content = content
	slice.TranslatedContent = translatedContent
	slice.OriginLanguage = model.Language(originLang)
	slice.TargetLanguage = model.Language(targetLang)
	if err := b.sliceRepo.Update(slice); err != nil {
		return nil, err
	}
	return slice, nil
}

// Delete 删除提示词块
func (b *SliceBiz) Delete(id uint) error {
	return b.sliceRepo.Delete(id)
}
