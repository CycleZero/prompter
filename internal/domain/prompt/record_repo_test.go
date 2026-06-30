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
// RecordRepo 测试
// =============================================================================

// recordColumns 返回 PromptRecord 表的 sqlmock 列定义（与 GORM / recordRow 扫描顺序一致）
func recordColumns() []string {
	return []string{
		"id", "external_id", "title", "full_content",
		"created_at", "updated_at", "deleted_at",
	}
}

// recRegionCols 返回 prompt_record_regions 表的所有列名
func recRegionCols() []string {
	return []string{
		"id", "created_at", "updated_at", "deleted_at",
		"record_id", "region_id", "sort_order",
	}
}

// recRegionSliceCols 返回 prompt_record_region_slices 表的所有列名
func recRegionSliceCols() []string {
	return []string{
		"id", "created_at", "updated_at", "deleted_at",
		"record_region_id", "slice_id", "sort_order",
		"content", "translated_content", "custom_text",
	}
}

// TestRecordRepo_CreateWithRegions 测试事务中创建三层结构 Record→Region→Slice，验证 ID 回填
func TestRecordRepo_CreateWithRegions(t *testing.T) {
	t.Run("单Region无Slice", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		// 显式事务，需要 BEGIN/COMMIT
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `prompt_records`").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec("INSERT INTO `prompt_record_regions`").
			WillReturnResult(sqlmock.NewResult(10, 1))
		mock.ExpectCommit()

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		record := &model.PromptRecord{
			ExternalID:  "ext-001",
			Title:       "测试记录",
			FullContent: "full content text",
		}
		payloads := []*regionPayload{
			{
				Region: model.PromptRecordRegion{RegionID: 5, SortOrder: 0},
				Slices: nil,
			},
		}

		err := repo.CreateWithRegions(record, payloads)
		if err != nil {
			t.Fatalf("CreateWithRegions 失败: %v", err)
		}

		if record.ID != 1 {
			t.Errorf("Record.ID = %d，预期 1", record.ID)
		}
		if payloads[0].Region.ID != 10 {
			t.Errorf("Region.ID = %d，预期 10", payloads[0].Region.ID)
		}
		if payloads[0].Region.RecordID != 1 {
			t.Errorf("Region.RecordID = %d，预期 1", payloads[0].Region.RecordID)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("多Region带Slice验证ID回填", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		// 事务：Record → Region1 → Slices1 → Region2 → Slices2
		mock.ExpectBegin()
		mock.ExpectExec("INSERT INTO `prompt_records`").
			WillReturnResult(sqlmock.NewResult(1, 1))
		// Region1
		mock.ExpectExec("INSERT INTO `prompt_record_regions`").
			WillReturnResult(sqlmock.NewResult(100, 1))
		// Region1 的 2 个 Slices（批量插入，LastInsertId 为第一个）
		mock.ExpectExec("INSERT INTO `prompt_record_region_slices`").
			WillReturnResult(sqlmock.NewResult(1000, 2))
		// Region2
		mock.ExpectExec("INSERT INTO `prompt_record_regions`").
			WillReturnResult(sqlmock.NewResult(200, 1))
		// Region2 的 1 个 Slice
		mock.ExpectExec("INSERT INTO `prompt_record_region_slices`").
			WillReturnResult(sqlmock.NewResult(2000, 1))
		mock.ExpectCommit()

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		record := &model.PromptRecord{
			ExternalID: "ext-multi", Title: "多条记录",
		}
		customA := "自定义文本A"
		customB := "自定义文本B"
		customC := "自定义文本C"
		payloads := []*regionPayload{
			{
				Region: model.PromptRecordRegion{RegionID: 1, SortOrder: 0},
				Slices: []model.PromptRecordRegionSlice{
					{SliceID: 10, SortOrder: 0, Content: "原文A", TranslatedContent: "翻译A", CustomText: &customA},
					{SliceID: 20, SortOrder: 1, Content: "原文B", TranslatedContent: "翻译B", CustomText: &customB},
				},
			},
			{
				Region: model.PromptRecordRegion{RegionID: 2, SortOrder: 1},
				Slices: []model.PromptRecordRegionSlice{
					{SliceID: 30, SortOrder: 0, Content: "原文C", TranslatedContent: "翻译C", CustomText: &customC},
				},
			},
		}

		err := repo.CreateWithRegions(record, payloads)
		if err != nil {
			t.Fatalf("CreateWithRegions 失败: %v", err)
		}

		// 验证 Record ID
		if record.ID != 1 {
			t.Errorf("Record.ID = %d，预期 1", record.ID)
		}

		// Region1 验证
		r1 := &payloads[0].Region
		if r1.ID != 100 {
			t.Errorf("Region1.ID = %d，预期 100", r1.ID)
		}
		if r1.RecordID != 1 {
			t.Errorf("Region1.RecordID = %d，预期 1", r1.RecordID)
		}

		s1 := payloads[0].Slices
		if len(s1) != 2 {
			t.Fatalf("Region1 的 Slices 数量 = %d，预期 2", len(s1))
		}
		if s1[0].ID != 1000 {
			t.Errorf("Slice[0].ID = %d，预期 1000", s1[0].ID)
		}
		if s1[0].RecordRegionID != 100 {
			t.Errorf("Slice[0].RecordRegionID = %d，预期 100", s1[0].RecordRegionID)
		}
		if s1[1].ID != 1001 {
			t.Errorf("Slice[1].ID = %d，预期 1001", s1[1].ID)
		}

		// Region2 验证
		r2 := &payloads[1].Region
		if r2.ID != 200 {
			t.Errorf("Region2.ID = %d，预期 200", r2.ID)
		}
		if r2.RecordID != 1 {
			t.Errorf("Region2.RecordID = %d，预期 1", r2.RecordID)
		}

		s2 := payloads[1].Slices
		if len(s2) != 1 {
			t.Fatalf("Region2 的 Slices 数量 = %d，预期 1", len(s2))
		}
		if s2[0].ID != 2000 {
			t.Errorf("Region2 Slice[0].ID = %d，预期 2000", s2[0].ID)
		}
		if s2[0].RecordRegionID != 200 {
			t.Errorf("Region2 Slice[0].RecordRegionID = %d，预期 200", s2[0].RecordRegionID)
		}

		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRecordRepo_GetByExternalID 测试通过外部 ID 查询记录
func TestRecordRepo_GetByExternalID(t *testing.T) {
	t.Run("记录存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		mock.ExpectQuery("SELECT .* FROM `prompt_records`.*external_id").
			WillReturnRows(sqlmock.NewRows(recordColumns()).
				AddRow(1, "uuid-abc", "标题", "完整内容", now, now, nil))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		got, err := repo.GetByExternalID("uuid-abc")
		if err != nil {
			t.Fatalf("GetByExternalID 失败: %v", err)
		}
		if got.ExternalID != "uuid-abc" {
			t.Errorf("ExternalID = %q，预期 uuid-abc", got.ExternalID)
		}
		if got.Title != "标题" {
			t.Errorf("Title = %q，预期 标题", got.Title)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("记录不存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `prompt_records`.*external_id").
			WillReturnRows(sqlmock.NewRows(recordColumns()))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		_, err := repo.GetByExternalID("nonexistent")
		if err != gorm.ErrRecordNotFound {
			t.Errorf("错误类型 = %v，预期 gorm.ErrRecordNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRecordRepo_GetByID 测试按 ID 查询记录
func TestRecordRepo_GetByID(t *testing.T) {
	t.Run("记录存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		mock.ExpectQuery("SELECT .* FROM `prompt_records`").
			WillReturnRows(sqlmock.NewRows(recordColumns()).
				AddRow(42, "ext-42", "记录42", "body", now, now, nil))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		got, err := repo.GetByID(42)
		if err != nil {
			t.Fatalf("GetByID 失败: %v", err)
		}
		if got.ID != 42 {
			t.Errorf("ID = %d，预期 42", got.ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("记录不存在", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `prompt_records`").
			WillReturnRows(sqlmock.NewRows(recordColumns()))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		_, err := repo.GetByID(999)
		if err != gorm.ErrRecordNotFound {
			t.Errorf("错误类型 = %v，预期 gorm.ErrRecordNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRecordRepo_GetRegionsByRecordID 测试获取 Record 下的 Region 列表，验证按 sort_order 排序
func TestRecordRepo_GetRegionsByRecordID(t *testing.T) {
	t.Run("多Region按sort_order排序", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		mock.ExpectQuery("SELECT .* FROM `prompt_record_regions`.*record_id.*ORDER BY sort_order ASC").
			WillReturnRows(sqlmock.NewRows(recRegionCols()).
				AddRow(10, now, now, nil, 1, 5, 0).
				AddRow(20, now, now, nil, 1, 3, 1).
				AddRow(30, now, now, nil, 1, 7, 2))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		regions, err := repo.GetRegionsByRecordID(1)
		if err != nil {
			t.Fatalf("GetRegionsByRecordID 失败: %v", err)
		}
		if len(regions) != 3 {
			t.Fatalf("Regions 数量 = %d，预期 3", len(regions))
		}
		if regions[0].SortOrder != 0 || regions[1].SortOrder != 1 || regions[2].SortOrder != 2 {
			t.Errorf("SortOrder 排序错误: [%d, %d, %d]", regions[0].SortOrder, regions[1].SortOrder, regions[2].SortOrder)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("空结果", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `prompt_record_regions`.*record_id").
			WillReturnRows(sqlmock.NewRows(recRegionCols()))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		regions, err := repo.GetRegionsByRecordID(999)
		if err != nil {
			t.Fatalf("GetRegionsByRecordID 失败: %v", err)
		}
		if len(regions) != 0 {
			t.Errorf("Regions 数量 = %d，预期 0", len(regions))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRecordRepo_GetRegionSlices 测试获取 Region 下的 Slice 列表，验证按 sort_order 排序
func TestRecordRepo_GetRegionSlices(t *testing.T) {
	t.Run("多Slice按sort_order排序", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		customA := "覆盖A"
		mock.ExpectQuery("SELECT .* FROM `prompt_record_region_slices`.*record_region_id.*ORDER BY sort_order ASC").
			WillReturnRows(sqlmock.NewRows(recRegionSliceCols()).
				AddRow(100, now, now, nil, 10, 1, 0, "原文1", "翻译1", nil).
				AddRow(101, now, now, nil, 10, 2, 1, "原文2", "翻译2", &customA).
				AddRow(102, now, now, nil, 10, 3, 2, "原文3", "翻译3", nil))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		slices, err := repo.GetRegionSlices(10)
		if err != nil {
			t.Fatalf("GetRegionSlices 失败: %v", err)
		}
		if len(slices) != 3 {
			t.Fatalf("Slices 数量 = %d，预期 3", len(slices))
		}
		if slices[0].SortOrder != 0 || slices[2].SortOrder != 2 {
			t.Errorf("SortOrder 排序错误: [%d, %d, %d]", slices[0].SortOrder, slices[1].SortOrder, slices[2].SortOrder)
		}
		if slices[0].CustomText != nil {
			t.Error("slices[0].CustomText 应为 nil")
		}
		if slices[1].CustomText == nil || *slices[1].CustomText != "覆盖A" {
			t.Errorf("slices[1].CustomText = %v，预期 覆盖A", slices[1].CustomText)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("空结果", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `prompt_record_region_slices`.*record_region_id").
			WillReturnRows(sqlmock.NewRows(recRegionSliceCols()))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		slices, err := repo.GetRegionSlices(999)
		if err != nil {
			t.Fatalf("GetRegionSlices 失败: %v", err)
		}
		if len(slices) != 0 {
			t.Errorf("Slices 数量 = %d，预期 0", len(slices))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRecordRepo_GetAllRegionSlices 测试获取 Record 下所有 Slice 的扁平列表
func TestRecordRepo_GetAllRegionSlices(t *testing.T) {
	t.Run("扁平列表按sort_order排序", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		mock.ExpectQuery("SELECT .* FROM `prompt_record_region_slices`.*JOIN.*prompt_record_regions").
			WillReturnRows(sqlmock.NewRows(recRegionSliceCols()).
				AddRow(1, now, now, nil, 10, 100, 0, "a", "aa", nil).
				AddRow(2, now, now, nil, 10, 200, 1, "b", "bb", nil).
				AddRow(3, now, now, nil, 20, 300, 2, "c", "cc", nil))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		slices, err := repo.GetAllRegionSlices(1)
		if err != nil {
			t.Fatalf("GetAllRegionSlices 失败: %v", err)
		}
		if len(slices) != 3 {
			t.Fatalf("Slices 数量 = %d，预期 3", len(slices))
		}
		if slices[0].SortOrder != 0 || slices[2].SortOrder != 2 {
			t.Errorf("SortOrder 排序错误")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("空结果", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		mock.ExpectQuery("SELECT .* FROM `prompt_record_region_slices`.*JOIN.*prompt_record_regions").
			WillReturnRows(sqlmock.NewRows(recRegionSliceCols()))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		slices, err := repo.GetAllRegionSlices(999)
		if err != nil {
			t.Fatalf("GetAllRegionSlices 失败: %v", err)
		}
		if len(slices) != 0 {
			t.Errorf("Slices 数量 = %d，预期 0", len(slices))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRecordRepo_List 测试分页列表查询
func TestRecordRepo_List(t *testing.T) {
	t.Run("空列表", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		// Count 查询
		mock.ExpectQuery("SELECT count\\(.*\\) FROM `prompt_records`").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(0))
		// Find 查询
		mock.ExpectQuery("SELECT .* FROM `prompt_records`.*ORDER BY id DESC").
			WillReturnRows(sqlmock.NewRows(recordColumns()))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		records, total, err := repo.List(1, 10)
		if err != nil {
			t.Fatalf("List 失败: %v", err)
		}
		if total != 0 {
			t.Errorf("total = %d，预期 0", total)
		}
		if len(records) != 0 {
			t.Errorf("records 长度 = %d，预期 0", len(records))
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})

	t.Run("多条记录验证分页", func(t *testing.T) {
		gormDB, mock, sqlDB := newMockDB(t)
		defer sqlDB.Close()

		now := time.Now()
		// Count 返回总数 5
		mock.ExpectQuery("SELECT count\\(.*\\) FROM `prompt_records`").
			WillReturnRows(sqlmock.NewRows([]string{"count(*)"}).AddRow(5))
		// 第2页，每页2条，按 id DESC
		mock.ExpectQuery("SELECT .* FROM `prompt_records`.*ORDER BY id DESC").
			WillReturnRows(sqlmock.NewRows(recordColumns()).
				AddRow(3, "ext-3", "记录3", "c", now, now, nil).
				AddRow(2, "ext-2", "记录2", "b", now, now, nil))

		repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
		records, total, err := repo.List(2, 2)
		if err != nil {
			t.Fatalf("List 失败: %v", err)
		}
		if total != 5 {
			t.Errorf("total = %d，预期 5", total)
		}
		if len(records) != 2 {
			t.Fatalf("records 长度 = %d，预期 2", len(records))
		}
		if records[0].ID != 3 || records[1].ID != 2 {
			t.Errorf("排序错误: IDs = [%d, %d]，预期 [3, 2]", records[0].ID, records[1].ID)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Errorf("未满足的 mock 期望: %v", err)
		}
	})
}

// TestRecordRepo_Delete 测试删除记录（软删除）
func TestRecordRepo_Delete(t *testing.T) {
	gormDB, mock, sqlDB := newMockDB(t)
	defer sqlDB.Close()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_records`.*SET.*deleted_at").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	repo := &RecordRepo{db: gormDB, data: &infra.Data{DB: gormDB}}
	err := repo.Delete(1)
	if err != nil {
		t.Fatalf("Delete 失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的 mock 期望: %v", err)
	}
}
