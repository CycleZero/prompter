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
// SliceRepo 测试
// =============================================================================

// sliceColumns 返回 PromptSlice 表的 sqlmock 列定义（与 GORM 扫描顺序一致）
func sliceColumns() []string {
	return []string{
		"id", "created_at", "updated_at", "deleted_at",
		"type_id", "content", "translated_content",
		"origin_language", "target_language",
	}
}

// uintPtr 返回 uint 的指针，用于构造 TypeID 等可选字段
func uintPtr(v uint) *uint {
	return &v
}

// TestSliceRepo_Create 测试创建 Slice（不携带 regionIDs）
func TestSliceRepo_Create(t *testing.T) {
	t.Run("正常创建不带RegionID", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `prompt_slices`").
			WillReturnResult(sqlmock.NewResult(10, 1))
		mock.ExpectCommit()

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		slice := &model.PromptSlice{
			Content:        "beautiful sunset",
			OriginLanguage: model.English,
			TargetLanguage: model.Chinese,
		}
		err := repo.Create(slice, nil)
		if err != nil {
			t.Fatalf("Create 失败: %v", err)
		}
		if slice.ID != 10 {
			t.Errorf("回填 ID = %d，预期 10", slice.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("创建带TypeID的Slice", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `prompt_slices`").
			WillReturnResult(sqlmock.NewResult(20, 1))
		mock.ExpectCommit()

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		typeID := uint(5)
		slice := &model.PromptSlice{
			TypeID:         &typeID,
			Content:        "masterpiece",
			OriginLanguage: model.English,
			TargetLanguage: model.Chinese,
		}
		err := repo.Create(slice, []uint{1, 2}) // regionIDs 被忽略
		if err != nil {
			t.Fatalf("Create 失败: %v", err)
		}
		if slice.ID != 20 {
			t.Errorf("回填 ID = %d，预期 20", slice.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestSliceRepo_GetByID 测试按 ID 查询提示词块
func TestSliceRepo_GetByID(t *testing.T) {
	t.Run("记录存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		mock.ExpectQuery("SELECT .* FROM `prompt_slices`").
			WillReturnRows(sqlmock.NewRows(sliceColumns()).
				AddRow(1, now, now, nil, nil, "hello", "你好", "english", "chinese"))

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		got, err := repo.GetByID(1)
		if err != nil {
			t.Fatalf("GetByID 失败: %v", err)
		}
		if got.ID != 1 {
			t.Errorf("ID = %d，预期 1", got.ID)
		}
		if got.Content != "hello" {
			t.Errorf("Content = %q，预期 hello", got.Content)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("记录不存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `prompt_slices`").
			WillReturnRows(sqlmock.NewRows(sliceColumns()))

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		_, err := repo.GetByID(999)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("错误类型 = %v，预期 gorm.ErrRecordNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestSliceRepo_ListByRegion 测试通过 Record 层级查询 Region 下的 Slice
func TestSliceRepo_ListByRegion(t *testing.T) {
	t.Run("空结果", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT DISTINCT.*FROM `prompt_slices`.*JOIN.*prompt_record_region_slices.*JOIN.*prompt_record_regions").
			WillReturnRows(sqlmock.NewRows(sliceColumns()))

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		slices, err := repo.ListByRegion(1)
		if err != nil {
			t.Fatalf("ListByRegion 失败: %v", err)
		}
		if len(slices) != 0 {
			t.Errorf("列表长度 = %d，预期 0", len(slices))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("有结果", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		tid := uint(3)
		mock.ExpectQuery("SELECT DISTINCT.*FROM `prompt_slices`.*JOIN.*prompt_record_region_slices.*JOIN.*prompt_record_regions").
			WithArgs(uint(1)). // regionID = 1
			WillReturnRows(sqlmock.NewRows(sliceColumns()).
				AddRow(10, now, now, nil, &tid, "slice-a", "块A", "english", "chinese").
				AddRow(11, now, now, nil, &tid, "slice-b", "块B", "english", "chinese"))

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		slices, err := repo.ListByRegion(1)
		if err != nil {
			t.Fatalf("ListByRegion 失败: %v", err)
		}
		if len(slices) != 2 {
			t.Fatalf("列表长度 = %d，预期 2", len(slices))
		}
		if slices[0].Content != "slice-a" {
			t.Errorf("slices[0].Content = %q，预期 slice-a", slices[0].Content)
		}
		if slices[1].Content != "slice-b" {
			t.Errorf("slices[1].Content = %q，预期 slice-b", slices[1].Content)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestSliceRepo_ListByType 测试按语义分类查询切片
func TestSliceRepo_ListByType(t *testing.T) {
	t.Run("按type_id查询无结果", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `prompt_slices`.*type_id.*ORDER BY id ASC").
			WillReturnRows(sqlmock.NewRows(sliceColumns()))

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		slices, err := repo.ListByType(99)
		if err != nil {
			t.Fatalf("ListByType 失败: %v", err)
		}
		if len(slices) != 0 {
			t.Errorf("列表长度 = %d，预期 0", len(slices))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("按type_id查询有结果", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		tid := uint(5)
		mock.ExpectQuery("SELECT .* FROM `prompt_slices`.*type_id.*ORDER BY id ASC").
			WillReturnRows(sqlmock.NewRows(sliceColumns()).
				AddRow(1, now, now, nil, &tid, "s1", "一", "english", "chinese").
				AddRow(3, now, now, nil, &tid, "s2", "二", "english", "chinese").
				AddRow(7, now, now, nil, &tid, "s3", "三", "english", "chinese"))

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		slices, err := repo.ListByType(5)
		if err != nil {
			t.Fatalf("ListByType 失败: %v", err)
		}
		if len(slices) != 3 {
			t.Fatalf("列表长度 = %d，预期 3", len(slices))
		}
		if slices[0].ID != 1 || slices[2].ID != 7 {
			t.Errorf("排序错误: IDs = [%d, %d, %d]，预期 [1, 3, 7]", slices[0].ID, slices[1].ID, slices[2].ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestSliceRepo_Update 测试更新提示词块
func TestSliceRepo_Update(t *testing.T) {
	gormDB, mock, sqlDB := newMockDB(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_slices`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
	slice := &model.PromptSlice{
		Content:        "updated content",
		OriginLanguage: model.English,
	}
	slice.ID = 1
	err := repo.Update(slice)
	if err != nil {
		t.Fatalf("Update 失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的 mock 期望: %v", err)
	}
}

// TestSliceRepo_Delete 测试删除提示词块
func TestSliceRepo_Delete(t *testing.T) {
	t.Run("删除存在记录", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `prompt_slices`.*SET.*deleted_at").
			WillReturnResult(sqlmock.NewResult(0, 1))
		mock.ExpectCommit()

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		err := repo.Delete(1)
		if err != nil {
			t.Fatalf("Delete 失败: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("删除不存在记录不报错", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectBegin()
		mock.ExpectExec("UPDATE `prompt_slices`.*SET.*deleted_at").
			WillReturnResult(sqlmock.NewResult(0, 0))
		mock.ExpectCommit()

		repo := &SliceRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		err := repo.Delete(999)
		if err != nil {
			t.Fatalf("删除不存在记录不应报错: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}
