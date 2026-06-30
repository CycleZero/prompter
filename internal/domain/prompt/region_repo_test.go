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
// RegionRepo 测试
// =============================================================================

// TestRegionRepo_Create 测试创建 PromptRegion，验证 ID 回填及字段完整性
func TestRegionRepo_Create(t *testing.T) {
	t.Run("正常创建并回填ID", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `prompt_regions`").
			WillReturnResult(sqlmock.NewResult(42, 1))
		mock.ExpectCommit()

		repo := &RegionRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		region := &model.PromptRegion{Name: "人物", SortOrder: 1, Description: "角色外观描述"}
		err := repo.Create(region)
		if err != nil {
			t.Fatalf("Create 失败: %v", err)
		}
		if region.ID != 42 {
			t.Errorf("回填 ID = %d，预期 42", region.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRegionRepo_GetByID 测试按 ID 查询类别
func TestRegionRepo_GetByID(t *testing.T) {
	t.Run("记录存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		mock.ExpectQuery("SELECT .* FROM `prompt_regions`").
			WillReturnRows(sqlmock.NewRows(regionColumns()).
				AddRow(1, "人物", 0, "角色外观描述", now, now, nil))

		repo := &RegionRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		got, err := repo.GetByID(1)
		if err != nil {
			t.Fatalf("GetByID 失败: %v", err)
		}
		if got.ID != 1 {
			t.Errorf("ID = %d，预期 1", got.ID)
		}
		if got.Name != "人物" {
			t.Errorf("Name = %q，预期 人物", got.Name)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("记录不存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `prompt_regions`").
			WillReturnRows(sqlmock.NewRows(regionColumns()))

		repo := &RegionRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		_, err := repo.GetByID(999)
		if err == nil {
			t.Fatal("期望返回错误，实际为 nil")
		}
		if err != gorm.ErrRecordNotFound {
			t.Errorf("错误类型 = %v，预期 gorm.ErrRecordNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRegionRepo_List 测试列表查询，验证 sort_order ASC 排序
func TestRegionRepo_List(t *testing.T) {
	cols := regionColumns()

	t.Run("空列表", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `prompt_regions`.*ORDER BY sort_order ASC").
			WillReturnRows(sqlmock.NewRows(cols))

		repo := &RegionRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		regions, err := repo.List()
		if err != nil {
			t.Fatalf("List 失败: %v", err)
		}
		if len(regions) != 0 {
			t.Errorf("列表长度 = %d，预期 0", len(regions))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("单条记录", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		mock.ExpectQuery("SELECT .* FROM `prompt_regions`.*ORDER BY sort_order ASC").
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow(1, "主题", 0, "主题描述", now, now, nil))

		repo := &RegionRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		regions, err := repo.List()
		if err != nil {
			t.Fatalf("List 失败: %v", err)
		}
		if len(regions) != 1 {
			t.Fatalf("列表长度 = %d，预期 1", len(regions))
		}
		if regions[0].Name != "主题" {
			t.Errorf("Name = %q，预期 主题", regions[0].Name)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("多条记录验证排序", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		mock.ExpectQuery("SELECT .* FROM `prompt_regions`.*ORDER BY sort_order ASC").
			WillReturnRows(sqlmock.NewRows(cols).
				AddRow(1, "人物", 0, "", now, now, nil).
				AddRow(2, "风格", 1, "", now, now, nil).
				AddRow(3, "质量", 2, "", now, now, nil))

		repo := &RegionRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		regions, err := repo.List()
		if err != nil {
			t.Fatalf("List 失败: %v", err)
		}
		if len(regions) != 3 {
			t.Fatalf("列表长度 = %d，预期 3", len(regions))
		}
		expected := []struct {
			id   uint
			name string
		}{
			{1, "人物"},
			{2, "风格"},
			{3, "质量"},
		}
		for i, exp := range expected {
			if regions[i].ID != exp.id {
				t.Errorf("regions[%d].ID = %d，预期 %d", i, regions[i].ID, exp.id)
			}
			if regions[i].Name != exp.name {
				t.Errorf("regions[%d].Name = %q，预期 %q", i, regions[i].Name, exp.name)
			}
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRegionRepo_Update 测试更新类别
func TestRegionRepo_Update(t *testing.T) {
	t.Run("更新成功", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `prompt_regions`").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		repo := &RegionRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		region := &model.PromptRegion{
			Name:        "更新后的名称",
			SortOrder:   5,
			Description: "更新后的描述",
		}
		region.ID = 1
		err := repo.Update(region)
		if err != nil {
			t.Fatalf("Update 失败: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRegionRepo_Delete 测试删除类别（软删除）
func TestRegionRepo_Delete(t *testing.T) {
	t.Run("删除存在的记录", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `prompt_regions`.*SET.*deleted_at").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		repo := &RegionRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		err := repo.Delete(1)
		if err != nil {
			t.Fatalf("Delete 失败: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("删除不存在的记录", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `prompt_regions`.*SET.*deleted_at").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		repo := &RegionRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		err := repo.Delete(999)
		if err != nil {
			t.Fatalf("删除不存在记录不应报错，但返回: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}
