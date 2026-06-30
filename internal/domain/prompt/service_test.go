package prompt

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"prompter/infra"
)

// ============================================================
// 测试辅助函数
// ============================================================

// testNow 统一测试时间
var testNow = time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC)

// regionColumns 返回 PromptRegion 表的 sqlmock 列定义（与 GORM 扫描顺序一致）
func regionColumns() []string {
	return []string{
		"id", "name", "sort_order", "description",
		"created_at", "updated_at", "deleted_at",
	}
}

// regionRow 创建一条 PromptRegion 行数据
func regionRow(id uint, name string, sortOrder int) *sqlmock.Rows {
	return sqlmock.NewRows(regionColumns()).
		AddRow(id, name, sortOrder, "", testNow, testNow, nil)
}

// recordRow 创建一条 PromptRecord 行数据
func recordRow(id uint, externalID, title, fullContent string) *sqlmock.Rows {
	return sqlmock.NewRows(recordColumns()).
		AddRow(id, externalID, title, fullContent, testNow, testNow, nil)
}

// newMiniredisClient 创建基于 miniredis 的 RedisClient，用于 DraftRepo 的 Redis 层
func newMiniredisClient(t *testing.T) (*miniredis.Miniredis, *infra.RedisClient) {
	t.Helper()
	s, err := miniredis.Run()
	if err != nil {
		t.Fatalf("启动 miniredis 失败: %v", err)
	}
	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	rc := infra.NewCustomRedisClient(rdb)
	return s, rc
}

// setupTestRouter 创建完整的 Gin 测试路由，使用 mock DB + miniredis。
// 所有 repo 通过手动构造（绕过 AutoMigrate），与现有测试风格一致。
func setupTestRouter(t *testing.T) (*gin.Engine, sqlmock.Sqlmock, func()) {
	t.Helper()

	gormDB, mock, sqlDB := newMockDB(t)
	mr, rc := newMiniredisClient(t)

	data := &infra.Data{DB: gormDB, RedisClient: rc}

	// 手动构造 repo（不调用 NewXxxRepo 以避免 AutoMigrate）
	regionRepo := &RegionRepo{db: gormDB, data: data}
	sliceRepo := &SliceRepo{db: gormDB, data: data}
	recordRepo := &RecordRepo{db: gormDB, data: data}
	draftRepo := &DraftRepo{data: data}
	sliceTypeRepo := &SliceTypeRepo{db: gormDB, data: data}

	regionBiz := NewRegionBiz(regionRepo)
	sliceBiz := NewSliceBiz(sliceRepo)
	draftBiz := NewDraftBiz(draftRepo)
	recordBiz := NewRecordBiz(recordRepo, sliceRepo, draftRepo)
	sliceTypeBiz := NewSliceTypeBiz(sliceTypeRepo)

	svc := NewPromptService(regionBiz, sliceBiz, draftBiz, recordBiz, sliceTypeBiz)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	api := r.Group("/api")

	// Region 路由
	regions := api.Group("/regions")
	{
		regions.POST("", svc.CreateRegion)
		regions.GET("", svc.ListRegions)
		regions.GET("/:id", svc.GetRegion)
		regions.PUT("/:id", svc.UpdateRegion)
		regions.DELETE("/:id", svc.DeleteRegion)
	}

	// Slice 路由
	slices := api.Group("/slices")
	{
		slices.POST("", svc.CreateSlice)
		slices.GET("", svc.ListSlices)
		slices.GET("/:id", svc.GetSlice)
		slices.PUT("/:id", svc.UpdateSlice)
		slices.DELETE("/:id", svc.DeleteSlice)
	}

	// 活动 Prompt 路由
	api.GET("/active-prompt", svc.GetActivePrompt)
	api.PUT("/active-prompt", svc.UpdateActivePrompt)

	// Record 路由
	api.POST("/records/:uuid", svc.PersistRecord)
	api.GET("/records", svc.ListRecords)
	api.GET("/records/:id", svc.GetRecord)
	api.DELETE("/records/:id", svc.DeleteRecord)

	// Combo + SliceType 路由
	api.GET("/combo/tree", svc.GetComboTree)
	api.GET("/slice-types", svc.GetSliceTypeTree)

	cleanup := func() {
		mr.Close()
		sqlDB.Close()
	}

	return r, mock, cleanup
}

// newRequest 创建 JSON 请求的快捷方法
func newRequest(method, path string, body any) *http.Request {
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	req.Header.Set("Content-Type", "application/json")
	return req
}

// parseJSON 解析 httptest 响应中的 JSON body
func parseJSON[T any](t *testing.T, w *httptest.ResponseRecorder) T {
	t.Helper()
	var result T
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("JSON 解析失败: %v\n原始响应: %s", err, w.Body.String())
	}
	return result
}

// parseJSONList 解析 JSON 数组响应
func parseJSONList[T any](t *testing.T, w *httptest.ResponseRecorder) []T {
	t.Helper()
	var result []T
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("JSON 数组解析失败: %v\n原始响应: %s", err, w.Body.String())
	}
	return result
}

// checkMock 验证所有 sqlmock 期望均被满足
func checkMock(t *testing.T, mock sqlmock.Sqlmock) {
	t.Helper()
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的 mock 期望: %v", err)
	}
}

// ============================================================
// Region 端点测试
// ============================================================

func TestCreateRegion_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `prompt_regions`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	body := CreateRegionRequest{
		Name:        "人物",
		Description: "人物外观特征",
		SortOrder:   0,
	}

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPost, "/api/regions", body)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[*RegionResponse](t, w)
	if resp.ID != 1 {
		t.Errorf("期望 ID=1，得到 %d", resp.ID)
	}
	if resp.Name != "人物" {
		t.Errorf("期望 Name='人物'，得到 '%s'", resp.Name)
	}
	if resp.SortOrder != 0 {
		t.Errorf("期望 SortOrder=0，得到 %d", resp.SortOrder)
	}

	checkMock(t, mock)
}

func TestCreateRegion_BadRequest(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/regions", nil)
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[map[string]string](t, w)
	if resp["error"] != "请求参数错误" {
		t.Errorf("期望错误消息为'请求参数错误'，得到 '%s'", resp["error"])
	}

	checkMock(t, mock)
}

func TestListRegions_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .* FROM `prompt_regions`.*ORDER BY sort_order ASC").
		WillReturnRows(
			regionRow(1, "人物", 0).
				AddRow(2, "风格", 1, "", testNow, testNow, nil),
		)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/regions", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSONList[*RegionResponse](t, w)
	if len(resp) != 2 {
		t.Fatalf("期望 2 条记录，得到 %d", len(resp))
	}
	if resp[0].Name != "人物" || resp[0].SortOrder != 0 {
		t.Errorf("第 1 条: Name='%s' SortOrder=%d，期望 人物/0", resp[0].Name, resp[0].SortOrder)
	}
	if resp[1].Name != "风格" || resp[1].SortOrder != 1 {
		t.Errorf("第 2 条: Name='%s' SortOrder=%d，期望 风格/1", resp[1].Name, resp[1].SortOrder)
	}

	checkMock(t, mock)
}

func TestGetRegion_NotFound(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .* FROM `prompt_regions`").
		WillReturnRows(sqlmock.NewRows(regionColumns()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/regions/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望状态码 404，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[map[string]string](t, w)
	if resp["error"] != "记录不存在" {
		t.Errorf("期望错误消息为'记录不存在'，得到 '%s'", resp["error"])
	}

	checkMock(t, mock)
}

func TestDeleteRegion_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_regions`.*SET.*deleted_at").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/regions/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[map[string]string](t, w)
	if resp["message"] != "删除成功" {
		t.Errorf("期望消息为'删除成功'，得到 '%s'", resp["message"])
	}

	checkMock(t, mock)
}

func TestUpdateRegion_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .* FROM `prompt_regions`").
		WillReturnRows(regionRow(1, "旧名称", 0))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_regions`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	body := UpdateRegionRequest{
		Name:        "新名称",
		Description: "更新后的描述",
		SortOrder:   5,
	}

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPut, "/api/regions/1", body)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[*RegionResponse](t, w)
	if resp.Name != "新名称" {
		t.Errorf("期望 Name='新名称'，得到 '%s'", resp.Name)
	}

	checkMock(t, mock)
}

// ============================================================
// Slice 端点测试
// ============================================================

func TestCreateSlice_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `prompt_slices`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	body := CreateSliceRequest{
		Content:           "masterpiece",
		TranslatedContent: "杰作",
		OriginLanguage:    "english",
		TargetLanguage:    "chinese",
		RegionIDs:         []uint{1, 2},
	}

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPost, "/api/slices", body)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[*SliceResponse](t, w)
	if resp.ID != 1 {
		t.Errorf("期望 ID=1，得到 %d", resp.ID)
	}
	if resp.Content != "masterpiece" {
		t.Errorf("期望 Content='masterpiece'，得到 '%s'", resp.Content)
	}
	if resp.TranslatedContent != "杰作" {
		t.Errorf("期望 TranslatedContent='杰作'，得到 '%s'", resp.TranslatedContent)
	}
	if resp.OriginLanguage != "english" {
		t.Errorf("期望 OriginLanguage='english'，得到 '%s'", resp.OriginLanguage)
	}

	checkMock(t, mock)
}

func TestListSlices_ByRegionID(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT DISTINCT.*FROM `prompt_slices`.*JOIN.*prompt_record_region_slices.*JOIN.*prompt_record_regions").
		WillReturnRows(
			sqlmock.NewRows(sliceColumns()).
				AddRow(5, testNow, testNow, nil, nil, "masterpiece", "杰作", "english", "chinese").
				AddRow(6, testNow, testNow, nil, nil, "best quality", "最高质量", "english", "chinese"),
		)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/slices?region_id=1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[SliceListResponse](t, w)
	if resp.Total != 2 {
		t.Fatalf("期望 Total=2，得到 %d", resp.Total)
	}
	if len(resp.List) != 2 {
		t.Fatalf("期望 2 条，得到 %d", len(resp.List))
	}
	if resp.List[0].ID != 5 {
		t.Errorf("第 1 条 ID 期望 5，得到 %d", resp.List[0].ID)
	}

	checkMock(t, mock)
}

func TestListSlices_MissingRegionID(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/slices", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[map[string]string](t, w)
	if resp["error"] != "缺少 type_id 或 region_id 参数" {
		t.Errorf("期望错误消息为'缺少 type_id 或 region_id 参数'，得到 '%s'", resp["error"])
	}

	checkMock(t, mock)
}

func TestUpdateSlice_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .* FROM `prompt_slices`").
		WillReturnRows(sqlmock.NewRows(sliceColumns()).
			AddRow(1, testNow, testNow, nil, nil, "old", "旧", "english", "chinese"))
	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_slices`").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	body := UpdateSliceRequest{
		Content:           "updated content",
		TranslatedContent: "更新后内容",
		OriginLanguage:    "english",
		TargetLanguage:    "chinese",
	}

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPut, "/api/slices/1", body)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[*SliceResponse](t, w)
	if resp.Content != "updated content" {
		t.Errorf("期望 Content='updated content'，得到 '%s'", resp.Content)
	}

	checkMock(t, mock)
}

// ============================================================
// 活动 Prompt 端点测试
// ============================================================

func TestGetActivePrompt_Empty(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .* FROM `active_prompts`").
		WillReturnError(gorm.ErrRecordNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/active-prompt", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[ActivePromptResponse](t, w)
	if resp.Title != "" {
		t.Errorf("期望空 Title，得到 '%s'", resp.Title)
	}
	if resp.Regions != nil {
		t.Errorf("期望 Regions 为 nil，实际有 %d 条", len(resp.Regions))
	}

	checkMock(t, mock)
}

func TestUpdateActivePrompt_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	// FirstOrCreate: 先 SELECT 找记录（带 deleted_at IS NULL），找不到则事务内 INSERT
	mock.ExpectQuery("SELECT .* FROM `active_prompts`").
		WillReturnRows(sqlmock.NewRows(
			[]string{"id", "created_at", "updated_at", "deleted_at", "data"},
		))
	mock.ExpectBegin()
	mock.ExpectExec("INSERT INTO `active_prompts`").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	body := UpdateActivePromptRequest{
		Title: "测试标题",
		Regions: []ActivePromptRegionDTO{
			{
				RegionID:   1,
				RegionName: "人物",
				SortOrder:  0,
				Slices: []ActiveSliceDTO{
					{SliceID: 5, SortOrder: 0},
				},
			},
		},
	}

	w := httptest.NewRecorder()
	req := newRequest(http.MethodPut, "/api/active-prompt", body)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[map[string]string](t, w)
	if resp["status"] != "ok" {
		t.Errorf("期望 status='ok'，得到 '%s'", resp["status"])
	}

	checkMock(t, mock)
}

// ============================================================
// Record 端点测试
// ============================================================

func TestPersistRecord_NoActivePrompt(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .* FROM `prompt_records`.*external_id.*=.*").
		WillReturnError(gorm.ErrRecordNotFound)
	mock.ExpectQuery("SELECT .* FROM `active_prompts`").
		WillReturnError(gorm.ErrRecordNotFound)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/records/test-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("期望状态码 400，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[map[string]string](t, w)
	if resp["error"] != "没有活动的Prompt" {
		t.Errorf("期望错误消息为'没有活动的Prompt'，得到 '%s'", resp["error"])
	}

	checkMock(t, mock)
}

func TestPersistRecord_Idempotent(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .* FROM `prompt_records`.*external_id.*").
		WillReturnRows(
			recordRow(42, "existing-uuid", "已有记录", "full content"),
		)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/records/existing-uuid", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[*PersistRecordResponse](t, w)
	if resp.ID != 42 {
		t.Errorf("期望 ID=42，得到 %d", resp.ID)
	}
	if resp.ExternalID != "existing-uuid" {
		t.Errorf("期望 ExternalID='existing-uuid'，得到 '%s'", resp.ExternalID)
	}
	if resp.Title != "已有记录" {
		t.Errorf("期望 Title='已有记录'，得到 '%s'", resp.Title)
	}

	checkMock(t, mock)
}

func TestListRecords_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT count\\(\\*\\) FROM `prompt_records`").
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
	mock.ExpectQuery("SELECT \\* FROM `prompt_records`.*ORDER BY id DESC.*").
		WillReturnRows(
			recordRow(2, "uuid-2", "记录2", "内容2").
				AddRow(1, "uuid-1", "记录1", "内容1", testNow, testNow, nil),
		)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/records?page=1&page_size=10", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[ListRecordResponse](t, w)
	if resp.Total != 2 {
		t.Errorf("期望 Total=2，得到 %d", resp.Total)
	}
	if resp.Page != 1 {
		t.Errorf("期望 Page=1，得到 %d", resp.Page)
	}
	if len(resp.List) != 2 {
		t.Fatalf("期望 2 条记录，得到 %d", len(resp.List))
	}
	if resp.List[0].ExternalID != "uuid-2" {
		t.Errorf("第 1 条 ExternalID 期望 'uuid-2'，得到 '%s'", resp.List[0].ExternalID)
	}

	checkMock(t, mock)
}

func TestGetRecord_NotFound(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .* FROM `prompt_records`").
		WillReturnRows(sqlmock.NewRows(recordColumns()))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/records/999", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("期望状态码 404，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[map[string]string](t, w)
	if resp["error"] != "记录不存在" {
		t.Errorf("期望错误消息为'记录不存在'，得到 '%s'", resp["error"])
	}

	checkMock(t, mock)
}

func TestDeleteRecord_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectBegin()
	mock.ExpectExec("UPDATE `prompt_records`.*SET.*deleted_at").
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectCommit()

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodDelete, "/api/records/1", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[map[string]string](t, w)
	if resp["message"] != "删除成功" {
		t.Errorf("期望消息为'删除成功'，得到 '%s'", resp["message"])
	}

	checkMock(t, mock)
}

// ============================================================
// Combo + SliceType 端点测试
// ============================================================

func TestGetComboTree_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT .* FROM `prompt_regions`.*ORDER BY sort_order ASC").
		WillReturnRows(
			regionRow(1, "人物", 0).
				AddRow(2, "风格", 1, "", testNow, testNow, nil),
		)
	mock.ExpectQuery("SELECT DISTINCT.*FROM `prompt_slices`.*JOIN.*prompt_record_region_slices.*JOIN.*prompt_record_regions").
		WillReturnRows(sqlmock.NewRows(sliceColumns()).
			AddRow(10, testNow, testNow, nil, nil, "masterpiece", "杰作", "english", "chinese"))
	mock.ExpectQuery("SELECT DISTINCT.*FROM `prompt_slices`.*JOIN.*prompt_record_region_slices.*JOIN.*prompt_record_regions").
		WillReturnRows(sqlmock.NewRows(sliceColumns()).
			AddRow(20, testNow, testNow, nil, nil, "digital art", "数字艺术", "english", "chinese").
			AddRow(21, testNow, testNow, nil, nil, "oil painting", "油画", "english", "chinese"))

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/combo/tree", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[ComboTreeResponse](t, w)
	if len(resp.Regions) != 2 {
		t.Fatalf("期望 2 个 Region，得到 %d", len(resp.Regions))
	}
	if resp.Regions[0].Name != "人物" {
		t.Errorf("第 1 个 Region Name 期望 '人物'，得到 '%s'", resp.Regions[0].Name)
	}
	if len(resp.Regions[0].Slices) != 1 {
		t.Errorf("第 1 个 Region 期望 1 个 Slice，得到 %d", len(resp.Regions[0].Slices))
	}
	if len(resp.Regions[1].Slices) != 2 {
		t.Errorf("第 2 个 Region 期望 2 个 Slice，得到 %d", len(resp.Regions[1].Slices))
	}

	checkMock(t, mock)
}

func TestGetSliceTypeTree_Success(t *testing.T) {
	r, mock, cleanup := setupTestRouter(t)
	defer cleanup()

	mock.ExpectQuery("SELECT \\* FROM `slice_types`.*ORDER BY sort_order ASC").
		WillReturnRows(
			sqlmock.NewRows(sliceTypeColumns()).
				AddRow(1, testNow, testNow, nil, "人物", nil, 0).
				AddRow(2, testNow, testNow, nil, "头发", uint(1), 0).
				AddRow(3, testNow, testNow, nil, "眼睛", uint(1), 1).
				AddRow(4, testNow, testNow, nil, "画面", nil, 1),
		)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/slice-types", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("期望状态码 200，得到 %d: %s", w.Code, w.Body.String())
	}

	resp := parseJSON[SliceTypeTreeResponse](t, w)
	if len(resp.Types) != 2 {
		t.Fatalf("期望 2 个根分类，得到 %d", len(resp.Types))
	}

	// 按名称查找根节点（map 迭代顺序不确定）
	var rootPerson, rootScene *SliceTypeResponse
	for _, r := range resp.Types {
		switch r.Name {
		case "人物":
			rootPerson = r
		case "画面":
			rootScene = r
		}
	}
	if rootPerson == nil {
		t.Fatal("未找到根节点 '人物'")
	}
	if rootScene == nil {
		t.Fatal("未找到根节点 '画面'")
	}

	if len(rootPerson.Children) != 2 {
		t.Errorf("'人物' 期望 2 个子节点，得到 %d", len(rootPerson.Children))
	}
	if len(rootPerson.Children) >= 2 {
		if rootPerson.Children[0].Name != "头发" {
			t.Errorf("第 1 个子节点期望 '头发'，得到 '%s'", rootPerson.Children[0].Name)
		}
		if rootPerson.Children[1].Name != "眼睛" {
			t.Errorf("第 2 个子节点期望 '眼睛'，得到 '%s'", rootPerson.Children[1].Name)
		}
	}
	if rootScene.Name != "画面" {
		t.Errorf("根节点 Name 期望 '画面'，得到 '%s'", rootScene.Name)
	}

	checkMock(t, mock)
}
