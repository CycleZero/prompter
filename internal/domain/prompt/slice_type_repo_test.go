package prompt

import (
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"gorm.io/gorm"
	"prompter/infra"
	"prompter/model"
)

// =============================================================================
// SliceTypeRepo 测试
// =============================================================================

// sliceTypeColumns 返回 slice_types 表的 sqlmock 列定义（与 GORM 扫描顺序一致）
func sliceTypeColumns() []string {
	return []string{
		"id", "created_at", "updated_at", "deleted_at",
		"name", "parent_id", "sort_order",
	}
}

// stCols 为保持与旧代码兼容的别名
func stCols() []string { return sliceTypeColumns() }

// TestSliceTypeRepo_ListAll 测试获取所有分类（平铺），验证按 sort_order 排序
func TestSliceTypeRepo_ListAll(t *testing.T) {
	t.Run("父类型与子类型平铺按sort_order排序", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		parentID1 := uint(1)
		parentID2 := uint(2)
		mock.ExpectQuery("SELECT .* FROM `slice_types`.*ORDER BY sort_order ASC").
			WillReturnRows(sqlmock.NewRows(sliceTypeColumns()).
				AddRow(1, now, now, nil, "人物", nil, 0).
				AddRow(3, now, now, nil, "头发", &parentID1, 1).
				AddRow(4, now, now, nil, "眼睛", &parentID1, 2).
				AddRow(2, now, now, nil, "画面", nil, 3).
				AddRow(5, now, now, nil, "画质", &parentID2, 4))

		repo := &SliceTypeRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		types, err := repo.ListAll()
		if err != nil {
			t.Fatalf("ListAll 失败: %v", err)
		}
		if len(types) != 5 {
			t.Fatalf("类型数量 = %d，预期 5", len(types))
		}
		// 验证按 sort_order ASC
		for i := 1; i < len(types); i++ {
			if types[i].SortOrder < types[i-1].SortOrder {
				t.Errorf("排序错误: types[%d].SortOrder (%d) < types[%d].SortOrder (%d)",
					i, types[i].SortOrder, i-1, types[i-1].SortOrder)
			}
		}
		if types[0].Name != "人物" || types[0].ParentID != nil {
			t.Errorf("types[0] = {Name:%q, ParentID:%v}，预期 {人物, nil}", types[0].Name, types[0].ParentID)
		}
		if types[1].Name != "头发" || *types[1].ParentID != 1 {
			t.Errorf("types[1] = {Name:%q, ParentID:%v}，预期 {头发, 1}", types[1].Name, types[1].ParentID)
		}
		if types[3].Name != "画面" || types[3].ParentID != nil {
			t.Errorf("types[3] = {Name:%q, ParentID:%v}，预期 {画面, nil}", types[3].Name, types[3].ParentID)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("空列表", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `slice_types`.*ORDER BY sort_order ASC").
			WillReturnRows(sqlmock.NewRows(sliceTypeColumns()))

		repo := &SliceTypeRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		types, err := repo.ListAll()
		if err != nil {
			t.Fatalf("ListAll 失败: %v", err)
		}
		if len(types) != 0 {
			t.Errorf("类型数量 = %d，预期 0", len(types))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestSliceTypeRepo_Create 测试创建分类
func TestSliceTypeRepo_Create(t *testing.T) {
	t.Run("创建父类型", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `slice_types`").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectCommit()

		repo := &SliceTypeRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		st := &model.SliceType{Name: "人物", SortOrder: 0}
		err := repo.Create(st)
		if err != nil {
			t.Fatalf("Create 失败: %v", err)
		}
		if st.ID != 1 {
			t.Errorf("回填 ID = %d，预期 1", st.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("创建子类型带ParentID", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `slice_types`").
			WillReturnResult(sqlmock.NewResult(5, 1))
		mock.ExpectCommit()

		parentID := uint(1)
		repo := &SliceTypeRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		st := &model.SliceType{Name: "头发", ParentID: &parentID, SortOrder: 1}
		err := repo.Create(st)
		if err != nil {
			t.Fatalf("Create 失败: %v", err)
		}
		if st.ID != 5 {
			t.Errorf("回填 ID = %d，预期 5", st.ID)
		}
		if st.ParentID == nil || *st.ParentID != 1 {
			t.Errorf("ParentID = %v，预期 1", st.ParentID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestSliceTypeRepo_GetByID 测试按 ID 查询分类
func TestSliceTypeRepo_GetByID(t *testing.T) {
	t.Run("记录存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		parentID := uint(1)
		mock.ExpectQuery("SELECT .* FROM `slice_types`").
			WillReturnRows(sqlmock.NewRows(sliceTypeColumns()).
				AddRow(3, now, now, nil, "头发", &parentID, 1))

		repo := &SliceTypeRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		got, err := repo.GetByID(3)
		if err != nil {
			t.Fatalf("GetByID 失败: %v", err)
		}
		if got.ID != 3 {
			t.Errorf("ID = %d，预期 3", got.ID)
		}
		if got.Name != "头发" {
			t.Errorf("Name = %q，预期 头发", got.Name)
		}
		if got.ParentID == nil || *got.ParentID != 1 {
			t.Errorf("ParentID = %v，预期 1", got.ParentID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("记录不存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `slice_types`").
			WillReturnRows(sqlmock.NewRows(sliceTypeColumns()))

		repo := &SliceTypeRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		_, err := repo.GetByID(999)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("错误类型 = %v，预期 gorm.ErrRecordNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestSliceTypeRepo_Delete 测试删除分类
func TestSliceTypeRepo_Delete(t *testing.T) {
	gormDB, mock, sqlDB := newMockDB(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `slice_types`.*SET.*deleted_at").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := &SliceTypeRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
	err := repo.Delete(1)
	if err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的 mock 期望: %v", err)
	}
}
