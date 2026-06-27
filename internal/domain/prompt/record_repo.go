package prompt

import (
	"prompter/infra"
	"prompter/model"

	"gorm.io/gorm"
)

// RecordRepo 数据访问层 - 封装对 PromptRecord 表的数据库操作
type RecordRepo struct {
	db   *gorm.DB
	data *infra.Data
}

func NewRecordRepo(data *infra.Data) *RecordRepo {
	if err := data.DB.AutoMigrate(&model.PromptRecord{}, &model.PromptRecordSlice{}); err != nil {
		panic(err)
	}
	return &RecordRepo{db: data.DB, data: data}
}

// CreateWithSlices 创建记录并批量插入关联切片
func (r *RecordRepo) CreateWithSlices(record *model.PromptRecord, slices []model.PromptRecordSlice) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(record).Error; err != nil {
			return err
		}
		for i := range slices {
			slices[i].RecordID = record.ID
		}
		if len(slices) > 0 {
			return tx.Create(&slices).Error
		}
		return nil
	})
}

// GetByExternalID 通过外部 ID 查找记录
func (r *RecordRepo) GetByExternalID(externalID string) (*model.PromptRecord, error) {
	var record model.PromptRecord
	err := r.db.Where("external_id = ?", externalID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetByID 根据 ID 获取记录
func (r *RecordRepo) GetByID(id uint) (*model.PromptRecord, error) {
	var record model.PromptRecord
	err := r.db.First(&record, id).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// GetSlicesByRecordID 获取记录的所有关联切片，按排序升序
func (r *RecordRepo) GetSlicesByRecordID(recordID uint) ([]model.PromptRecordSlice, error) {
	var slices []model.PromptRecordSlice
	err := r.db.Where("record_id = ?", recordID).Order("sort_order ASC").Find(&slices).Error
	return slices, err
}

// List 分页获取记录列表，按 ID 降序
func (r *RecordRepo) List(page, pageSize int) ([]*model.PromptRecord, int64, error) {
	var records []*model.PromptRecord
	var total int64

	db := r.db.Model(&model.PromptRecord{})
	db.Count(&total)

	offset := (page - 1) * pageSize
	err := db.Offset(offset).Limit(pageSize).Order("id DESC").Find(&records).Error
	return records, total, err
}

// Delete 删除记录
func (r *RecordRepo) Delete(id uint) error {
	return r.db.Delete(&model.PromptRecord{}, id).Error
}
