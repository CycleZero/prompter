package prompt

import (
	"prompter/infra"
	"prompter/model"

	"gorm.io/gorm"
)

// RegionRepo 数据访问层 - 封装对 PromptRegion 表的数据库操作
type RegionRepo struct {
	db   *gorm.DB
	data *infra.Data
}

func NewRegionRepo(data *infra.Data) *RegionRepo {
	if err := data.DB.AutoMigrate(&model.PromptRegion{}); err != nil {
		panic(err)
	}
	return &RegionRepo{db: data.DB, data: data}
}

// Create 创建类别
func (r *RegionRepo) Create(region *model.PromptRegion) error {
	return r.db.Create(region).Error
}

// GetByID 根据 ID 获取类别
func (r *RegionRepo) GetByID(id uint) (*model.PromptRegion, error) {
	var region model.PromptRegion
	err := r.db.First(&region, id).Error
	if err != nil {
		return nil, err
	}
	return &region, nil
}

// List 获取所有类别，按排序字段升序
func (r *RegionRepo) List() ([]*model.PromptRegion, error) {
	var regions []*model.PromptRegion
	err := r.db.Order("sort_order ASC").Find(&regions).Error
	return regions, err
}

// Update 更新类别
func (r *RegionRepo) Update(region *model.PromptRegion) error {
	return r.db.Save(region).Error
}

// Delete 删除类别
func (r *RegionRepo) Delete(id uint) error {
	return r.db.Delete(&model.PromptRegion{}, id).Error
}
