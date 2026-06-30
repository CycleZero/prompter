package prompt

import (
	"prompter/infra"
	"prompter/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RecordRepo 数据访问层 - 封装对 PromptRecord 表的数据库操作
type RecordRepo struct {
	db   *gorm.DB
	data *infra.Data
}

func NewRecordRepo(data *infra.Data) *RecordRepo {
	if err := data.DB.AutoMigrate(
		&model.PromptRecord{},
		&model.PromptRecordRegion{},
		&model.PromptRecordRegionSlice{},
	); err != nil {
		panic(err)
	}
	return &RecordRepo{db: data.DB, data: data}
}

// regionPayload 持久化时传递的中间结构（不导出，仅供当前包使用）
// 封装了每个 Region 下需持久化的 RecordRegion、RecordRegionSlice 列表
// 以及用于后续回写 PromptRegionSlice 关联的条目
type regionPayload struct {
	Region             model.PromptRecordRegion
	Slices             []model.PromptRecordRegionSlice
	RegionSliceEntries []model.PromptRegionSlice
}

// CreateWithRegions 在事务中创建 Record → RecordRegion → RecordRegionSlice 三层结构
//  1. 创建 PromptRecord 主记录
//  2. 遍历每个载荷：创建 PromptRecordRegion（回填 RecordID）
//  3. 对每个 Region 下的 Slice 列表：创建 PromptRecordRegionSlice（回填 RecordRegionID）
//
// 返回 error 表示整个事务失败并回滚
func (r *RecordRepo) CreateWithRegions(record *model.PromptRecord, payloads []*regionPayload) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 步骤 1：创建 Record
		if err := tx.Create(record).Error; err != nil {
			return err
		}

		// 步骤 2-3：遍历每个 Region 载荷，逐层创建子实体
		for _, p := range payloads {
			// 绑定 Record 关联并创建 RecordRegion
			p.Region.RecordID = record.ID
			if err := tx.Create(&p.Region).Error; err != nil {
				return err
			}

			// 绑定 RecordRegion 关联并批量创建 RecordRegionSlice
			for i := range p.Slices {
				p.Slices[i].RecordRegionID = p.Region.ID
			}
			if len(p.Slices) > 0 {
				if err := tx.Create(&p.Slices).Error; err != nil {
					return err
				}
			}
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

// GetRegionsByRecordID 获取某条 Record 的所有 Region（按 SortOrder 排序）
func (r *RecordRepo) GetRegionsByRecordID(recordID uint) ([]*model.PromptRecordRegion, error) {
	var regions []*model.PromptRecordRegion
	err := r.db.Where("record_id = ?", recordID).Order("sort_order ASC").Find(&regions).Error
	return regions, err
}

// GetRegionSlices 获取某个 RecordRegion 下的所有 Slice（按 SortOrder 排序）
func (r *RecordRepo) GetRegionSlices(recordRegionID uint) ([]*model.PromptRecordRegionSlice, error) {
	var slices []*model.PromptRecordRegionSlice
	err := r.db.Where("record_region_id = ?", recordRegionID).Order("sort_order ASC").Find(&slices).Error
	return slices, err
}

// GetAllRegionSlices 获取某条 Record 的所有 RecordRegionSlice（扁平列表，按 SortOrder 排序）
// 用于需要遍历所有 Slice 但不关心 Region 分组的场景
func (r *RecordRepo) GetAllRegionSlices(recordID uint) ([]*model.PromptRecordRegionSlice, error) {
	var slices []*model.PromptRecordRegionSlice
	err := r.db.
		Joins("JOIN prompt_record_regions ON prompt_record_regions.id = prompt_record_region_slices.record_region_id").
		Where("prompt_record_regions.record_id = ?", recordID).
		Order("prompt_record_region_slices.sort_order ASC").
		Find(&slices).Error
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

// UpsertRegionSlices 批量插入 Region-Slice 关联（已存在则忽略），
// 在持久化记录时调用，使得 combo tree 能反映实际使用关系
func (r *RecordRepo) UpsertRegionSlices(entries []model.PromptRegionSlice) error {
	if len(entries) == 0 {
		return nil
	}
	return r.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&entries).Error
}
