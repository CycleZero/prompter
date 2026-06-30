package prompt

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"prompter/infra"
	"prompter/model"
)

// ============================================================
// 测试策略说明
// ============================================================
//
// 本文件包含三类测试：
//
// 1. DraftRepo 集成测试（Redis 层）
//    - 使用 miniredis 模拟 Redis，验证 GetActive 的读写行为
//    - 不依赖真实 Redis 或 MySQL
//    - 原有测试保留，用于验证 DraftRepo 层独立性
//
// 2. Biz 透传方法测试（DB 层）
//    - 使用 go-sqlmock 模拟 *gorm.DB，验证 Biz 方法正确透传到 Repo
//    - 覆盖 RecordBiz、RegionBiz、SliceBiz 的 GetByID、List、Delete 等方法
//    - 不经过 AutoMigrate，直接构造 repo 结构体并设置未导出字段
//
// 3. PersistFromActive 核心逻辑测试
//    - 使用 sqlmock + miniredis 组合，验证部分业务路径
//    - 幂等性检查（仅需 mock GetByExternalID）
//    - NoActivePrompt 分支（mock GetByExternalID + Redis miss + DB fallback）
//    - 完整流程（多 Region/多 Slice/事务写入）需真实数据库，用 skip 占位
//
// 约束：由于 Biz 层依赖具体类型（*RecordRepo 而非接口），无法使用传统 interface mock，
// 因此采用 sqlmock 在数据库驱动层拦截 SQL 调用。

// ============================================================
// 测试辅助函数
// ============================================================

// strPtr 返回指向给定字符串的指针，用于设置可选的 CustomText 字段
func strPtr(s string) *string {
	return &s
}

// newMockDB 创建 sqlmock 支持的 *gorm.DB，用于模拟数据库操作
// 返回 *gorm.DB、sqlmock.Sqlmock 和底层的 *sql.DB（调用方负责 Close）
func newMockDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock, *sql.DB) {
	t.Helper()
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	gormDB, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
	}), &gorm.Config{})
	if err != nil {
		sqlDB.Close()
		t.Fatalf("创建 gorm.DB 失败: %v", err)
	}
	return gormDB, mock, sqlDB
}

// newRecordBizOnly 创建仅含 sqlmock DB、不含 Redis 的 RecordBiz
// 用于透传方法测试，不需要 DraftRepo 访问 Redis
func newRecordBizOnly(t *testing.T) (*RecordBiz, sqlmock.Sqlmock, func()) {
	t.Helper()
	gormDB, mock, sqlDB := newMockDB(t)
	data := &infra.Data{DB: gormDB}

	// 绕过 AutoMigrate，直接设置未导出字段
	recordRepo := &RecordRepo{db: gormDB, data: data}
	sliceRepo := &SliceRepo{db: gormDB, data: data}
	draftRepo := &DraftRepo{data: data}

	biz := NewRecordBiz(recordRepo, sliceRepo, draftRepo)
	return biz, mock, func() { sqlDB.Close() }
}

// newRecordBizWithRedis 创建含 sqlmock DB + miniredis 的 RecordBiz
// 用于 PersistFromActive 部分路径测试：DraftRepo 可访问 Redis
func newRecordBizWithRedis(t *testing.T) (*RecordBiz, sqlmock.Sqlmock, *miniredis.Miniredis, func()) {
	t.Helper()
	gormDB, mock, sqlDB := newMockDB(t)

	s, err := miniredis.Run()
	if err != nil {
		sqlDB.Close()
		t.Fatalf("启动 miniredis 失败: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	rc := infra.NewCustomRedisClient(rdb)

	data := &infra.Data{DB: gormDB, RedisClient: rc}
	recordRepo := &RecordRepo{db: gormDB, data: data}
	sliceRepo := &SliceRepo{db: gormDB, data: data}
	draftRepo := &DraftRepo{data: data}

	biz := NewRecordBiz(recordRepo, sliceRepo, draftRepo)
	return biz, mock, s, func() {
		s.Close()
		sqlDB.Close()
	}
}

// newRegionBizOnly 创建仅含 sqlmock DB 的 RegionBiz
func newRegionBizOnly(t *testing.T) (*RegionBiz, sqlmock.Sqlmock, func()) {
	t.Helper()
	gormDB, mock, sqlDB := newMockDB(t)
	data := &infra.Data{DB: gormDB}

	regionRepo := &RegionRepo{db: gormDB, data: data}
	biz := NewRegionBiz(regionRepo)
	return biz, mock, func() { sqlDB.Close() }
}

// newSliceBizOnly 创建仅含 sqlmock DB 的 SliceBiz
func newSliceBizOnly(t *testing.T) (*SliceBiz, sqlmock.Sqlmock, func()) {
	t.Helper()
	gormDB, mock, sqlDB := newMockDB(t)
	data := &infra.Data{DB: gormDB}

	sliceRepo := &SliceRepo{db: gormDB, data: data}
	biz := NewSliceBiz(sliceRepo)
	return biz, mock, func() { sqlDB.Close() }
}

// ============================================================
// DraftRepo 集成测试（Redis 层） — 原有测试
// ============================================================

// TestPersistFromActive_DraftRepoWithMiniredis 验证 RecordBiz.PersistFromActive
// 所依赖的 DraftRepo Redis 读取路径。SetActive 内部调用 PutObject，后者使用 SetEx
// 并传入 0 过期时间（非法参数），因此我们绕过 SetActive/SetEx，直接通过
// redis SET 命令写入数据，然后验证 GetActive 能正确读取。
//
// 注意：本测试仅覆盖 DraftRepo 的 Redis 读取行为，RecordBiz 的完整持久化
// 流程需配合真实数据库，参见文件末尾的集成测试占位。
func TestPersistFromActive_DraftRepoWithMiniredis(t *testing.T) {
	// 启动 miniredis 服务器
	s, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// 将 miniredis 包装为 infra.RedisClient
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	rc := infra.NewCustomRedisClient(rdb)

	// 构建 Data：仅填充 RedisClient，DB 为 nil
	// DraftRepo 不使用 DB，RecordBiz/SliceRepo 依赖真实 *gorm.DB 因此不在此测试
	data := &infra.Data{RedisClient: rc}
	repo := NewDraftRepo(data)

	ctx := context.Background()
	key := activePromptKey

	// --- 子测试 1：空 Redis 时 GetActive 返回 nil ---
	t.Run("GetActive_Empty_Returns_Nil", func(t *testing.T) {
		active, err := repo.GetActive()
		if err != nil {
			t.Fatalf("非预期错误: %v", err)
		}
		if active != nil {
			t.Fatal("空 Redis 时期望 nil，实际非 nil")
		}
	})

	// --- 子测试 2：写入合法 JSON 后 GetActive 正确读取 ---
	t.Run("GetActive_Reads_Stored_JSON", func(t *testing.T) {
		input := ActivePromptData{
			Title: "test title",
			Regions: []ActivePromptRegionDTO{
				{
					RegionID: 1, RegionName: "人物", SortOrder: 0,
					Slices: []ActiveSliceDTO{
						{SliceID: 5, CustomText: nil, SortOrder: 0},
						{SliceID: 12, CustomText: strPtr("custom"), SortOrder: 1},
					},
				},
			},
			UpdatedAt: "2024-01-01T00:00:00Z",
		}
		raw, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("json.Marshal 失败: %v", err)
		}

		// 直接通过 redis SET 写入（绕过 SetEx 的 0 过期时间问题）
		if err := rdb.Set(ctx, key, string(raw), 0).Err(); err != nil {
			t.Fatalf("redis SET 失败: %v", err)
		}

		got, err := repo.GetActive()
		if err != nil {
			t.Fatalf("GetActive 失败: %v", err)
		}
		if got == nil {
			t.Fatal("写入数据后 GetActive 期望非 nil")
		}

		// 验证 Title
		if got.Title != "test title" {
			t.Errorf("Title: 得到 %q 期望 %q", got.Title, "test title")
		}

		// 验证 UpdatedAt
		if got.UpdatedAt != "2024-01-01T00:00:00Z" {
			t.Errorf("UpdatedAt: 得到 %q 期望 %q", got.UpdatedAt, "2024-01-01T00:00:00Z")
		}

		// 验证 Regions 数量
		if len(got.Regions) != 1 {
			t.Fatalf("Regions 数量: 得到 %d 期望 1", len(got.Regions))
		}
		r := got.Regions[0]
		if r.RegionID != 1 || r.RegionName != "人物" {
			t.Errorf("Region: 得到 id=%d name=%q 期望 id=1 name=人物", r.RegionID, r.RegionName)
		}
		if len(r.Slices) != 2 {
			t.Fatalf("Slices 数量: 得到 %d 期望 2", len(r.Slices))
		}

		// Slice 0 — CustomText 为 nil
		if r.Slices[0].SliceID != 5 {
			t.Errorf("slice[0].SliceID: 得到 %d 期望 5", r.Slices[0].SliceID)
		}
		if r.Slices[0].CustomText != nil {
			t.Error("slice[0].CustomText: 期望 nil")
		}

		// Slice 1 — CustomText 非 nil
		if r.Slices[1].SliceID != 12 {
			t.Errorf("slice[1].SliceID: 得到 %d 期望 12", r.Slices[1].SliceID)
		}
		if r.Slices[1].CustomText == nil {
			t.Fatal("slice[1].CustomText: 期望非 nil")
		}
		if *r.Slices[1].CustomText != "custom" {
			t.Errorf("slice[1].CustomText: 得到 %q 期望 %q", *r.Slices[1].CustomText, "custom")
		}
		if r.Slices[1].SortOrder != 1 {
			t.Errorf("slice[1].SortOrder: 得到 %d 期望 1", r.Slices[1].SortOrder)
		}
	})

	// --- 子测试 3：非法 JSON 时 GetActive 回退到 DB 并返回 nil ---
	t.Run("GetActive_Invalid_JSON_Fallback_Nil", func(t *testing.T) {
		if err := rdb.Set(ctx, key, "not-valid-json", 0).Err(); err != nil {
			t.Fatalf("redis SET 失败: %v", err)
		}

		got, err := repo.GetActive()
		if err != nil {
			t.Fatalf("非预期错误: %v", err)
		}
		if got != nil {
			t.Fatal("Redis 含非法 JSON 且 DB 不可用时，期望 nil")
		}
	})

	// --- 子测试 4：覆盖 Redis 数据后 GetActive 反映最新值 ---
	t.Run("GetActive_Reflects_Overwrite", func(t *testing.T) {
		first := ActivePromptData{
			Title: "first title",
			Regions: []ActivePromptRegionDTO{
				{RegionID: 1, RegionName: "人物", SortOrder: 0,
					Slices: []ActiveSliceDTO{{SliceID: 1, SortOrder: 0}}},
			},
		}
		raw1, _ := json.Marshal(first)
		if err := rdb.Set(ctx, key, string(raw1), 0).Err(); err != nil {
			t.Fatalf("redis SET 第一次失败: %v", err)
		}

		second := ActivePromptData{
			Title: "second title",
			Regions: []ActivePromptRegionDTO{
				{RegionID: 3, RegionName: "武器", SortOrder: 0,
					Slices: []ActiveSliceDTO{
						{SliceID: 7, SortOrder: 0},
						{SliceID: 8, SortOrder: 1},
					}},
			},
		}
		raw2, _ := json.Marshal(second)
		if err := rdb.Set(ctx, key, string(raw2), 0).Err(); err != nil {
			t.Fatalf("redis SET 第二次失败: %v", err)
		}

		got, err := repo.GetActive()
		if err != nil {
			t.Fatalf("覆盖后 GetActive 失败: %v", err)
		}

		if got.Title != "second title" {
			t.Errorf("覆盖后 Title: 得到 %q 期望 %q", got.Title, "second title")
		}
		if len(got.Regions) != 1 || len(got.Regions[0].Slices) != 2 {
			t.Fatalf("覆盖后 Regions/Slices: 得到 %d regions, %d slices",
				len(got.Regions), len(got.Regions[0].Slices))
		}
		if got.Regions[0].Slices[0].SliceID != 7 {
			t.Errorf("slice[0].SliceID: 得到 %d 期望 7", got.Regions[0].Slices[0].SliceID)
		}
	})
}

// ============================================================
// RecordBiz 透传方法测试 — 使用 sqlmock 验证参数正确传递
// ============================================================

// TestRecordBiz_GetByID_Success 测试通过 ID 成功获取记录
func TestRecordBiz_GetByID_Success(t *testing.T) {
	biz, mock, cleanup := newRecordBizOnly(t)
	defer cleanup()

	now := time.Now()
	// recordColumns 顺序: id, external_id, title, full_content, created_at, updated_at, deleted_at
	mock.ExpectQuery("SELECT \\* FROM `prompt_records`").
		WillReturnRows(sqlmock.NewRows(recordColumns()).
			AddRow(1, "uuid-1", "测试标题", "完整内容", now, now, nil))

	record, err := biz.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID 返回错误: %v", err)
	}
	if record.ID != 1 {
		t.Errorf("期望 ID=1, 得到 %d", record.ID)
	}
	if record.Title != "测试标题" {
		t.Errorf("期望 Title=\"测试标题\", 得到 %q", record.Title)
	}
	if record.ExternalID != "uuid-1" {
		t.Errorf("期望 ExternalID=\"uuid-1\", 得到 %q", record.ExternalID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRecordBiz_GetByID_NotFound 测试记录不存在时返回 gorm.ErrRecordNotFound
func TestRecordBiz_GetByID_NotFound(t *testing.T) {
	biz, mock, cleanup := newRecordBizOnly(t)
	defer cleanup()

	mock.ExpectQuery("SELECT \\* FROM `prompt_records`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := biz.GetByID(999)
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("期望 gorm.ErrRecordNotFound, 得到 %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRecordBiz_List 测试分页列表 — 验证 total 和列表正确透传
func TestRecordBiz_List(t *testing.T) {
	biz, mock, cleanup := newRecordBizOnly(t)
	defer cleanup()

	// Count 查询
	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `prompt_records`").WillReturnRows(
		sqlmock.NewRows([]string{"count(*)"}).AddRow(2),
	)
	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `prompt_records`").
		WillReturnRows(sqlmock.NewRows(recordColumns()).
			AddRow(1, "uuid-1", "标题一", "内容一", now, now, nil))

	records, total, err := biz.List(1, 10)
	if err != nil {
		t.Fatalf("List 返回错误: %v", err)
	}
	if total != 2 {
		t.Errorf("期望 total=2, 得到 %d", total)
	}
	if len(records) != 1 {
		t.Fatalf("期望 1 条记录, 得到 %d", len(records))
	}
	if records[0].ID != 1 {
		t.Errorf("期望记录 ID=1, 得到 %d", records[0].ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRecordBiz_List_MultiplePages 测试第二页分页 — 验证 offset 计算正确
func TestRecordBiz_List_MultiplePages(t *testing.T) {
	biz, mock, cleanup := newRecordBizOnly(t)
	defer cleanup()

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `prompt_records`").WillReturnRows(
		sqlmock.NewRows([]string{"count(*)"}).AddRow(25),
	)
	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `prompt_records`").
		WillReturnRows(sqlmock.NewRows(recordColumns()).
			AddRow(21, "uuid-21", "标题21", "内容21", now, now, nil))

	records, total, err := biz.List(3, 10)
	if err != nil {
		t.Fatalf("List 返回错误: %v", err)
	}
	if total != 25 {
		t.Errorf("期望 total=25, 得到 %d", total)
	}
	if len(records) != 1 {
		t.Fatalf("期望 1 条记录, 得到 %d", len(records))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRecordBiz_Delete 测试删除记录 — 软删除透传
func TestRecordBiz_Delete(t *testing.T) {
	biz, mock, cleanup := newRecordBizOnly(t)
	defer cleanup()

	// GORM 软删除生成 UPDATE SET deleted_at=?
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_records` SET `deleted_at`=").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := biz.Delete(1)
	if err != nil {
		t.Fatalf("Delete 返回错误: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRecordBiz_GetRegionsByRecordID 测试获取记录的 Region 列表
func TestRecordBiz_GetRegionsByRecordID(t *testing.T) {
	biz, mock, cleanup := newRecordBizOnly(t)
	defer cleanup()

	// GORM 列顺序: id, created_at, updated_at, deleted_at, record_id, region_id, sort_order
	now := time.Now()
	cols := []string{
		"id", "created_at", "updated_at", "deleted_at",
		"record_id", "region_id", "sort_order",
	}
	mock.ExpectQuery("SELECT \\* FROM `prompt_record_regions`").
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow(1, now, now, nil, 100, 10, 0).
			AddRow(2, now, now, nil, 100, 20, 1))

	regions, err := biz.GetRegionsByRecordID(100)
	if err != nil {
		t.Fatalf("GetRegionsByRecordID 返回错误: %v", err)
	}
	if len(regions) != 2 {
		t.Fatalf("期望 2 个 Region, 得到 %d", len(regions))
	}
	if regions[0].RegionID != 10 {
		t.Errorf("期望第一个 RegionID=10, 得到 %d", regions[0].RegionID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRecordBiz_GetRegionSlices 测试获取 Region 下的 Slice 列表
func TestRecordBiz_GetRegionSlices(t *testing.T) {
	biz, mock, cleanup := newRecordBizOnly(t)
	defer cleanup()

	now := time.Now()
	customText := "自定义文本"
	cols := []string{
		"id", "created_at", "updated_at", "deleted_at",
		"record_region_id", "slice_id", "sort_order",
		"content", "translated_content", "custom_text",
	}
	mock.ExpectQuery("SELECT \\* FROM `prompt_record_region_slices`").
		WillReturnRows(sqlmock.NewRows(cols).
			AddRow(1, now, now, nil, 5, 100, 0, "原文一", "翻译一", nil).
			AddRow(2, now, now, nil, 5, 200, 1, "原文二", "翻译二", &customText))

	slices, err := biz.GetRegionSlices(5)
	if err != nil {
		t.Fatalf("GetRegionSlices 返回错误: %v", err)
	}
	if len(slices) != 2 {
		t.Fatalf("期望 2 个 Slice, 得到 %d", len(slices))
	}
	if slices[0].SliceID != 100 {
		t.Errorf("期望第一个 SliceID=100, 得到 %d", slices[0].SliceID)
	}
	if slices[1].CustomText == nil || *slices[1].CustomText != "自定义文本" {
		t.Error("期望第二个 Slice CustomText=\"自定义文本\"")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// ============================================================
// RecordBiz.PersistFromActive 部分路径测试
// ============================================================

// TestRecordBiz_PersistFromActive_Idempotent 测试幂等性：
// GetByExternalID 返回已存在记录时，直接返回该记录而不创建新记录
func TestRecordBiz_PersistFromActive_Idempotent(t *testing.T) {
	biz, mock, cleanup := newRecordBizOnly(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `prompt_records`").
		WillReturnRows(sqlmock.NewRows(recordColumns()).
			AddRow(42, "test-uuid", "已存在标题", "已存在内容", now, now, nil))

	record, err := biz.PersistFromActive("test-uuid")
	if err != nil {
		t.Fatalf("PersistFromActive 返回错误: %v", err)
	}
	if record.ID != 42 {
		t.Errorf("期望返回已存在记录 ID=42, 得到 %d", record.ID)
	}
	if record.ExternalID != "test-uuid" {
		t.Errorf("期望 ExternalID=\"test-uuid\", 得到 %q", record.ExternalID)
	}
	if record.Title != "已存在标题" {
		t.Errorf("期望 Title=\"已存在标题\", 得到 %q", record.Title)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRecordBiz_PersistFromActive_NoActivePrompt 测试无活动 Prompt 场景：
// GetByExternalID 未找到记录 → DraftRepo.GetActive 返回 nil → 返回 ErrNoActivePrompt
func TestRecordBiz_PersistFromActive_NoActivePrompt(t *testing.T) {
	biz, mock, _, cleanup := newRecordBizWithRedis(t)
	defer cleanup()

	// 步骤 1：GetByExternalID 返回 ErrRecordNotFound
	mock.ExpectQuery("SELECT \\* FROM `prompt_records`").
		WillReturnError(gorm.ErrRecordNotFound)

	// 步骤 2：draftRepo.GetActive 的 DB 回退查询也返回 ErrRecordNotFound
	mock.ExpectQuery("SELECT \\* FROM `active_prompts`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := biz.PersistFromActive("test-uuid")
	if err == nil {
		t.Fatal("期望返回 ErrNoActivePrompt，实际为 nil")
	}
	if !errors.Is(err, ErrNoActivePrompt) {
		t.Errorf("期望 ErrNoActivePrompt, 得到 %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRecordBiz_PersistFromActive_EmptyRegions 测试活动 Prompt 存在但 Regions 为空的场景
func TestRecordBiz_PersistFromActive_EmptyRegions(t *testing.T) {
	biz, mock, _, cleanup := newRecordBizWithRedis(t)
	defer cleanup()

	// 步骤 1：GetByExternalID → 未找到
	mock.ExpectQuery("SELECT \\* FROM `prompt_records`").
		WillReturnError(gorm.ErrRecordNotFound)

	// 步骤 2：draftRepo.GetActive DB 回退 — 返回 Regions 为空的 ActivePrompt
	emptyData := ActivePromptData{
		Title:     "空 Regions",
		Regions:   []ActivePromptRegionDTO{},
		UpdatedAt: "2024-01-01T00:00:00Z",
	}
	raw, err := json.Marshal(emptyData)
	if err != nil {
		t.Fatalf("json.Marshal 失败: %v", err)
	}
	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `active_prompts`").
		WillReturnRows(sqlmock.NewRows([]string{
			"id", "created_at", "updated_at", "deleted_at", "data",
		}).AddRow(1, now, now, nil, string(raw)))

	_, err = biz.PersistFromActive("test-uuid")
	if err == nil {
		t.Fatal("期望返回 ErrNoActivePrompt，实际为 nil")
	}
	if !errors.Is(err, ErrNoActivePrompt) {
		t.Errorf("期望 ErrNoActivePrompt, 得到 %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// ============================================================
// PersistFromActive 完整流程集成测试（需真实数据库）
// ============================================================

// TestRecordBiz_PersistFromActive_Full_Integration 完整流程集成测试
// 覆盖 CustomText 覆盖、SortOrder 排序、AssemblePrompt 拼接、事务写入
// 需要真实数据库环境，当前跳过
func TestRecordBiz_PersistFromActive_Full_Integration(t *testing.T) {
	t.Skip("需要真实数据库环境：完整 PersistFromActive 流程涉及多表事务写入，" +
		"当前 sqlmock+miniredis 组合可覆盖部分路径，" +
		"完整验证请使用真实 MySQL 数据库")
}

// ============================================================
// RegionBiz 透传方法测试 — 使用 sqlmock 验证参数正确传递
// ============================================================

// TestRegionBiz_GetByID_Success 测试通过 ID 成功获取类别
func TestRegionBiz_GetByID_Success(t *testing.T) {
	biz, mock, cleanup := newRegionBizOnly(t)
	defer cleanup()

	now := time.Now()
	// regionColumns 顺序: id, name, sort_order, description, created_at, updated_at, deleted_at
	mock.ExpectQuery("SELECT \\* FROM `prompt_regions`").
		WillReturnRows(sqlmock.NewRows(regionColumns()).
			AddRow(1, "人物", 0, "角色描述", now, now, nil))

	region, err := biz.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID 返回错误: %v", err)
	}
	if region.ID != 1 {
		t.Errorf("期望 ID=1, 得到 %d", region.ID)
	}
	if region.Name != "人物" {
		t.Errorf("期望 Name=\"人物\", 得到 %q", region.Name)
	}
	if region.SortOrder != 0 {
		t.Errorf("期望 SortOrder=0, 得到 %d", region.SortOrder)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRegionBiz_GetByID_NotFound 测试类别不存在时返回错误
func TestRegionBiz_GetByID_NotFound(t *testing.T) {
	biz, mock, cleanup := newRegionBizOnly(t)
	defer cleanup()

	mock.ExpectQuery("SELECT \\* FROM `prompt_regions`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := biz.GetByID(999)
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("期望 gorm.ErrRecordNotFound, 得到 %v", err)
	}
}

// TestRegionBiz_List 测试获取类别列表
func TestRegionBiz_List(t *testing.T) {
	biz, mock, cleanup := newRegionBizOnly(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `prompt_regions`").
		WillReturnRows(sqlmock.NewRows(regionColumns()).
			AddRow(1, "人物", 0, "角色相关", now, now, nil).
			AddRow(2, "场景", 1, "环境相关", now, now, nil).
			AddRow(3, "风格", 2, "画风相关", now, now, nil))

	regions, err := biz.List()
	if err != nil {
		t.Fatalf("List 返回错误: %v", err)
	}
	if len(regions) != 3 {
		t.Fatalf("期望 3 个 Region, 得到 %d", len(regions))
	}
	if regions[0].Name != "人物" {
		t.Errorf("期望第一个 Name=\"人物\", 得到 %q", regions[0].Name)
	}
	if regions[2].Name != "风格" {
		t.Errorf("期望第三个 Name=\"风格\", 得到 %q", regions[2].Name)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRegionBiz_Create 测试创建类别 — 验证 INSERT 透传
func TestRegionBiz_Create(t *testing.T) {
	biz, mock, cleanup := newRegionBizOnly(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `prompt_regions`").
		WillReturnResult(sqlmock.NewResult(5, 1))
	mock.ExpectCommit()

	region, err := biz.Create("新类别", "这是描述", 3)
	if err != nil {
		t.Fatalf("Create 返回错误: %v", err)
	}
	if region.ID != 5 {
		t.Errorf("期望 ID=5, 得到 %d", region.ID)
	}
	if region.Name != "新类别" {
		t.Errorf("期望 Name=\"新类别\", 得到 %q", region.Name)
	}
	if region.SortOrder != 3 {
		t.Errorf("期望 SortOrder=3, 得到 %d", region.SortOrder)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRegionBiz_Update 测试更新类别 — 先查后改后保存
func TestRegionBiz_Update(t *testing.T) {
	biz, mock, cleanup := newRegionBizOnly(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `prompt_regions`").
		WillReturnRows(sqlmock.NewRows(regionColumns()).
			AddRow(1, "旧名称", 0, "旧描述", now, now, nil))

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_regions`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	region, err := biz.Update(1, "新名称", "新描述", 5)
	if err != nil {
		t.Fatalf("Update 返回错误: %v", err)
	}
	if region.Name != "新名称" {
		t.Errorf("期望更新后 Name=\"新名称\", 得到 %q", region.Name)
	}
	if region.SortOrder != 5 {
		t.Errorf("期望更新后 SortOrder=5, 得到 %d", region.SortOrder)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestRegionBiz_Delete 测试删除类别 — 软删除透传
func TestRegionBiz_Delete(t *testing.T) {
	biz, mock, cleanup := newRegionBizOnly(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_regions` SET `deleted_at`=").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := biz.Delete(1)
	if err != nil {
		t.Fatalf("Delete 返回错误: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// ============================================================
// SliceBiz 透传方法测试 — 使用 sqlmock 验证参数正确传递
// ============================================================

// sliceBizColumns 返回 PromptSlice 表的列名（用于 SliceBiz 测试）
func sliceBizColumns() []string {
	return []string{
		"id", "created_at", "updated_at", "deleted_at",
		"type_id", "content", "translated_content",
		"origin_language", "target_language",
	}
}

// TestSliceBiz_GetByID_Success 测试通过 ID 成功获取提示词块
func TestSliceBiz_GetByID_Success(t *testing.T) {
	biz, mock, cleanup := newSliceBizOnly(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `prompt_slices`").
		WillReturnRows(sqlmock.NewRows(sliceBizColumns()).
			AddRow(1, now, now, nil, nil, "原始内容", "translated", "chinese", "english"))

	slice, err := biz.GetByID(1)
	if err != nil {
		t.Fatalf("GetByID 返回错误: %v", err)
	}
	if slice.ID != 1 {
		t.Errorf("期望 ID=1, 得到 %d", slice.ID)
	}
	if slice.Content != "原始内容" {
		t.Errorf("期望 Content=\"原始内容\", 得到 %q", slice.Content)
	}
	if slice.TranslatedContent != "translated" {
		t.Errorf("期望 TranslatedContent=\"translated\", 得到 %q", slice.TranslatedContent)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestSliceBiz_GetByID_NotFound 测试提示词块不存在时返回错误
func TestSliceBiz_GetByID_NotFound(t *testing.T) {
	biz, mock, cleanup := newSliceBizOnly(t)
	defer cleanup()

	mock.ExpectQuery("SELECT \\* FROM `prompt_slices`").
		WillReturnError(gorm.ErrRecordNotFound)

	_, err := biz.GetByID(999)
	if err == nil {
		t.Fatal("期望返回错误，实际为 nil")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("期望 gorm.ErrRecordNotFound, 得到 %v", err)
	}
}

// TestSliceBiz_ListByRegion 测试按 Region 查询 Slice 列表（通过 Record 层级关联）
func TestSliceBiz_ListByRegion(t *testing.T) {
	biz, mock, cleanup := newSliceBizOnly(t)
	defer cleanup()

	now := time.Now()
	// ListByRegion 使用 JOIN 查询，匹配 DISTINCT 子句（无 backtick 的表别名 .*）
	mock.ExpectQuery("SELECT DISTINCT prompt_slices\\.\\* FROM `prompt_slices`").
		WillReturnRows(sqlmock.NewRows(sliceBizColumns()).
			AddRow(1, now, now, nil, nil, "内容A", "transA", "chinese", "english").
			AddRow(2, now, now, nil, nil, "内容B", "transB", "chinese", "english"))

	slices, err := biz.ListByRegion(1)
	if err != nil {
		t.Fatalf("ListByRegion 返回错误: %v", err)
	}
	if len(slices) != 2 {
		t.Fatalf("期望 2 个 Slice, 得到 %d", len(slices))
	}
	if slices[0].Content != "内容A" {
		t.Errorf("期望第一个 Content=\"内容A\", 得到 %q", slices[0].Content)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestSliceBiz_ListByType 测试按语义分类查询 Slice 列表
func TestSliceBiz_ListByType(t *testing.T) {
	biz, mock, cleanup := newSliceBizOnly(t)
	defer cleanup()

	now := time.Now()
	typeID := uint(5)
	mock.ExpectQuery("SELECT \\* FROM `prompt_slices`").
		WillReturnRows(sqlmock.NewRows(sliceBizColumns()).
			AddRow(10, now, now, nil, &typeID, "发型", "hairstyle", "chinese", "english"))

	slices, err := biz.ListByType(5)
	if err != nil {
		t.Fatalf("ListByType 返回错误: %v", err)
	}
	if len(slices) != 1 {
		t.Fatalf("期望 1 个 Slice, 得到 %d", len(slices))
	}
	if slices[0].ID != 10 {
		t.Errorf("期望 ID=10, 得到 %d", slices[0].ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestSliceBiz_Create 测试创建提示词块 — 验证 INSERT 透传
func TestSliceBiz_Create(t *testing.T) {
	biz, mock, cleanup := newSliceBizOnly(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `prompt_slices`").
		WillReturnResult(sqlmock.NewResult(7, 1))
	mock.ExpectCommit()

	slice, err := biz.Create("新提示词", "new prompt", "chinese", "english", []uint{1, 2})
	if err != nil {
		t.Fatalf("Create 返回错误: %v", err)
	}
	if slice.ID != 7 {
		t.Errorf("期望 ID=7, 得到 %d", slice.ID)
	}
	if slice.Content != "新提示词" {
		t.Errorf("期望 Content=\"新提示词\", 得到 %q", slice.Content)
	}
	if slice.OriginLanguage != model.Chinese {
		t.Errorf("期望 OriginLanguage=chinese, 得到 %v", slice.OriginLanguage)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestSliceBiz_Update 测试更新提示词块 — 先查后改后保存
func TestSliceBiz_Update(t *testing.T) {
	biz, mock, cleanup := newSliceBizOnly(t)
	defer cleanup()

	now := time.Now()
	mock.ExpectQuery("SELECT \\* FROM `prompt_slices`").
		WillReturnRows(sqlmock.NewRows(sliceBizColumns()).
			AddRow(1, now, now, nil, nil, "旧内容", "old", "chinese", "english"))

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_slices`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	slice, err := biz.Update(1, "新内容", "new", "english", "chinese")
	if err != nil {
		t.Fatalf("Update 返回错误: %v", err)
	}
	if slice.Content != "新内容" {
		t.Errorf("期望 Content=\"新内容\", 得到 %q", slice.Content)
	}
	if slice.TranslatedContent != "new" {
		t.Errorf("期望 TranslatedContent=\"new\", 得到 %q", slice.TranslatedContent)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}

// TestSliceBiz_Delete 测试删除提示词块 — 软删除透传
func TestSliceBiz_Delete(t *testing.T) {
	biz, mock, cleanup := newSliceBizOnly(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_slices` SET `deleted_at`=").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	err := biz.Delete(1)
	if err != nil {
		t.Fatalf("Delete 返回错误: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("mock 期望未满足: %v", err)
	}
}
