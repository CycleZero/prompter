package prompt

import (
	"prompter/infra"
	"prompter/model"

	"gorm.io/gorm"
)

// SliceTypeRepo 数据访问层 - 封装对 SliceType 表的数据库操作
type SliceTypeRepo struct {
	db   *gorm.DB
	data *infra.Data
}

// NewSliceTypeRepo 创建 SliceTypeRepo 并自动迁移
func NewSliceTypeRepo(data *infra.Data) *SliceTypeRepo {
	if err := data.DB.AutoMigrate(&model.SliceType{}); err != nil {
		panic(err)
	}
	return &SliceTypeRepo{db: data.DB, data: data}
}

// ListAll 获取所有分类（平铺），按排序字段升序
func (r *SliceTypeRepo) ListAll() ([]*model.SliceType, error) {
	var types []*model.SliceType
	err := r.db.Order("sort_order ASC").Find(&types).Error
	return types, err
}

// Create 创建分类
func (r *SliceTypeRepo) Create(st *model.SliceType) error {
	return r.db.Create(st).Error
}

// GetByID 根据 ID 获取分类
func (r *SliceTypeRepo) GetByID(id uint) (*model.SliceType, error) {
	var st model.SliceType
	err := r.db.First(&st, id).Error
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// Delete 删除分类
func (r *SliceTypeRepo) Delete(id uint) error {
	return r.db.Delete(&model.SliceType{}, id).Error
}
