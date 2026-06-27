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
	if err := data.DB.AutoMigrate(&model.PromptSlice{}, &model.PromptRegionSlice{}); err != nil {
		panic(err)
	}
	return &SliceRepo{db: data.DB, data: data}
}

// Create 创建提示词块并关联到指定类别
func (r *SliceRepo) Create(slice *model.PromptSlice, regionIDs []uint) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(slice).Error; err != nil {
			return err
		}
		for _, regionID := range regionIDs {
			prs := &model.PromptRegionSlice{
				RegionID: regionID,
				SliceID:  slice.ID,
			}
			if err := tx.Create(prs).Error; err != nil {
				return err
			}
		}
		return nil
	})
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

// ListByRegion 查询指定类别下的所有提示词块，按类别内排序
func (r *SliceRepo) ListByRegion(regionID uint) ([]*model.PromptSlice, error) {
	var slices []*model.PromptSlice
	err := r.db.Joins("JOIN prompt_region_slices ON prompt_region_slices.slice_id = prompt_slices.id").
		Where("prompt_region_slices.region_id = ? AND prompt_region_slices.deleted_at IS NULL", regionID).
		Order("prompt_region_slices.sort_order ASC").
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

// GetRegionSlices 获取块关联的所有类别 ID
func (r *SliceRepo) GetRegionSlices(sliceID uint) ([]uint, error) {
	var regionIDs []uint
	err := r.db.Model(&model.PromptRegionSlice{}).
		Where("slice_id = ?", sliceID).
		Pluck("region_id", &regionIDs).Error
	return regionIDs, err
}
