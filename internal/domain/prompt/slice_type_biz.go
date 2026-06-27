package prompt

import "prompter/model"

// SliceTypeBiz 业务逻辑层 - 处理 SliceType 的业务规则
type SliceTypeBiz struct {
	sliceTypeRepo *SliceTypeRepo
}

// NewSliceTypeBiz 创建 SliceTypeBiz
func NewSliceTypeBiz(sliceTypeRepo *SliceTypeRepo) *SliceTypeBiz {
	return &SliceTypeBiz{sliceTypeRepo: sliceTypeRepo}
}

// GetTree 获取分类树（二级树：ParentID=nil 为根，Children 为二级）
func (b *SliceTypeBiz) GetTree() ([]*SliceTypeResponse, error) {
	all, err := b.sliceTypeRepo.ListAll()
	if err != nil {
		return nil, err
	}

	// 构建 id → response 映射
	m := make(map[uint]*SliceTypeResponse)
	for _, t := range all {
		m[t.ID] = &SliceTypeResponse{
			ID:        t.ID,
			Name:      t.Name,
			ParentID:  t.ParentID,
			SortOrder: t.SortOrder,
			Children:  make([]*SliceTypeResponse, 0),
		}
	}

	// 构建树形结构
	roots := make([]*SliceTypeResponse, 0)
	for _, r := range m {
		if r.ParentID == nil {
			roots = append(roots, r)
		} else if parent, ok := m[*r.ParentID]; ok {
			parent.Children = append(parent.Children, r)
		} else {
			// 孤儿节点当作根节点处理
			roots = append(roots, r)
		}
	}
	return roots, nil
}

// Create 创建分类
func (b *SliceTypeBiz) Create(name string, parentID *uint, sortOrder int) (*model.SliceType, error) {
	st := &model.SliceType{
		Name:      name,
		ParentID:  parentID,
		SortOrder: sortOrder,
	}
	if err := b.sliceTypeRepo.Create(st); err != nil {
		return nil, err
	}
	return st, nil
}

// GetByID 根据 ID 获取分类
func (b *SliceTypeBiz) GetByID(id uint) (*model.SliceType, error) {
	return b.sliceTypeRepo.GetByID(id)
}

// Delete 删除分类
func (b *SliceTypeBiz) Delete(id uint) error {
	return b.sliceTypeRepo.Delete(id)
}
