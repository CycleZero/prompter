package prompt

import (
	"prompter/infra"
	"prompter/model"

	"gorm.io/gorm"
)

// SliceRepo 数据访问层 - 封装对 PromptSlice 表的数据库操作
type SliceRepo struct {
	db   *gorm.DB
	data *infra.Data
}

func NewSliceRepo(data *infra.Data) *SliceRepo {
	if err := data.DB.AutoMigrate(&model.PromptSlice{}); err != nil {
		panic(err)
	}
	return &SliceRepo{db: data.DB, data: data}
}

// Create 创建提示词块（regionIDs 参数保留以兼容调用方签名，但不再持久化关联）
func (r *SliceRepo) Create(slice *model.PromptSlice, regionIDs []uint) error {
	return r.db.Create(slice).Error
}

// GetByID 根据 ID 获取提示词块
func (r *SliceRepo) GetByID(id uint) (*model.PromptSlice, error) {
	var slice model.PromptSlice
	err := r.db.First(&slice, id).Error
	if err != nil {
		return nil, err
	}
	return &slice, nil
}

// ListByRegion 查询指定类别下历史上被使用过的 Slice（去重），通过 Record 层级关联
func (r *SliceRepo) ListByRegion(regionID uint) ([]*model.PromptSlice, error) {
	var slices []*model.PromptSlice
	err := r.db.
		Distinct("prompt_slices.*").
		Joins("JOIN prompt_record_region_slices ON prompt_record_region_slices.slice_id = prompt_slices.id").
		Joins("JOIN prompt_record_regions ON prompt_record_regions.id = prompt_record_region_slices.record_region_id").
		Where("prompt_record_regions.region_id = ?", regionID).
		Find(&slices).Error
	return slices, err
}

// Update 更新提示词块
func (r *SliceRepo) Update(slice *model.PromptSlice) error {
	return r.db.Save(slice).Error
}

// Delete 删除提示词块
func (r *SliceRepo) Delete(id uint) error {
	return r.db.Delete(&model.PromptSlice{}, id).Error
}

// ListByType 按语义分类查询所有切片
func (r *SliceRepo) ListByType(typeID uint) ([]*model.PromptSlice, error) {
	var slices []*model.PromptSlice
	err := r.db.Where("type_id = ?", typeID).Order("id ASC").Find(&slices).Error
	return slices, err
}

// GetByIDs 批量按 ID 列表查询 Slice（搜索结果回源 DB 补全字段）
func (r *SliceRepo) GetByIDs(ids []uint) ([]*model.PromptSlice, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var slices []*model.PromptSlice
	err := r.db.Where("id IN ?", ids).Find(&slices).Error
	return slices, err
}

// ListAll 获取所有 Slice（用于搜索索引全量重建）
func (r *SliceRepo) ListAll() ([]*model.PromptSlice, error) {
	var slices []*model.PromptSlice
	err := r.db.Find(&slices).Error
	return slices, err
}
