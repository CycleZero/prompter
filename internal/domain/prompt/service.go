package prompt

import (
	"errors"
	"net/http"
	"strconv"

	"prompter/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PromptService HTTP 服务层 - 处理请求解析、参数校验、响应格式化
type PromptService struct {
	regionBiz    *RegionBiz
	sliceBiz     *SliceBiz
	draftBiz     *DraftBiz
	recordBiz    *RecordBiz
	sliceTypeBiz *SliceTypeBiz
}

// NewPromptService 创建 PromptService
func NewPromptService(
	regionBiz *RegionBiz,
	sliceBiz *SliceBiz,
	draftBiz *DraftBiz,
	recordBiz *RecordBiz,
	sliceTypeBiz *SliceTypeBiz,
) *PromptService {
	return &PromptService{
		regionBiz:    regionBiz,
		sliceBiz:     sliceBiz,
		draftBiz:     draftBiz,
		recordBiz:    recordBiz,
		sliceTypeBiz: sliceTypeBiz,
	}
}

// ============================================================
// Region 处理器
// ============================================================

// CreateRegion 创建新类别
// @Summary 创建类别
// @Tags prompt
// @Accept json
// @Produce json
// @Param request body CreateRegionRequest true "创建请求"
// @Success 200 {object} RegionResponse
// @Router /api/regions [post]
func (s *PromptService) CreateRegion(c *gin.Context) {
	var req CreateRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	region, err := s.regionBiz.Create(req.Name, req.Description, req.SortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, toRegionResponse(region))
}

// ListRegions 获取类别列表
// @Summary 获取类别列表
// @Tags prompt
// @Produce json
// @Success 200 {array} RegionResponse
// @Router /api/regions [get]
func (s *PromptService) ListRegions(c *gin.Context) {
	regions, err := s.regionBiz.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	responses := make([]*RegionResponse, 0, len(regions))
	for _, r := range regions {
		responses = append(responses, toRegionResponse(r))
	}

	c.JSON(http.StatusOK, responses)
}

// GetRegion 获取类别详情
// @Summary 获取类别
// @Tags prompt
// @Produce json
// @Param id path int true "类别ID"
// @Success 200 {object} RegionResponse
// @Router /api/regions/{id} [get]
func (s *PromptService) GetRegion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 格式错误"})
		return
	}
	region, err := s.regionBiz.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, toRegionResponse(region))
}

// UpdateRegion 更新类别
// @Summary 更新类别
// @Tags prompt
// @Accept json
// @Produce json
// @Param id path int true "类别ID"
// @Param request body UpdateRegionRequest true "更新请求"
// @Success 200 {object} RegionResponse
// @Router /api/regions/{id} [put]
func (s *PromptService) UpdateRegion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 格式错误"})
		return
	}

	var req UpdateRegionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	region, err := s.regionBiz.Update(uint(id), req.Name, req.Description, req.SortOrder)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, toRegionResponse(region))
}

// DeleteRegion 删除类别
// @Summary 删除类别
// @Tags prompt
// @Produce json
// @Param id path int true "类别ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/regions/{id} [delete]
func (s *PromptService) DeleteRegion(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 格式错误"})
		return
	}

	if err := s.regionBiz.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ============================================================
// Slice 处理器
// ============================================================

// CreateSlice 创建提示词块
// @Summary 创建提示词块
// @Tags prompt
// @Accept json
// @Produce json
// @Param request body CreateSliceRequest true "创建请求"
// @Success 200 {object} SliceResponse
// @Router /api/slices [post]
func (s *PromptService) CreateSlice(c *gin.Context) {
	var req CreateSliceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	slice, err := s.sliceBiz.Create(req.Content, req.TranslatedContent, req.OriginLanguage, req.TargetLanguage, req.RegionIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusOK, toSliceResponse(slice))
}

// ListSlices 获取提示词块列表
// @Summary 获取提示词块列表
// @Tags prompt
// @Produce json
// @Param type_id query int false "语义分类ID"
// @Param region_id query int false "类别ID"
// @Success 200 {object} SliceListResponse
// @Router /api/slices [get]
func (s *PromptService) ListSlices(c *gin.Context) {
	typeIDStr := c.Query("type_id")
	regionIDStr := c.Query("region_id")

	var slices []*model.PromptSlice
	var err error

	if typeIDStr != "" {
		typeID, parseErr := strconv.ParseUint(typeIDStr, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "type_id 格式错误"})
			return
		}
		slices, err = s.sliceBiz.ListByType(uint(typeID))
	} else if regionIDStr != "" {
		regionID, parseErr := strconv.ParseUint(regionIDStr, 10, 32)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "region_id 格式错误"})
			return
		}
		slices, err = s.sliceBiz.ListByRegion(uint(regionID))
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少 type_id 或 region_id 参数"})
		return
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	responses := make([]*SliceResponse, 0, len(slices))
	for _, sl := range slices {
		responses = append(responses, toSliceResponse(sl))
	}

	c.JSON(http.StatusOK, SliceListResponse{
		List:  responses,
		Total: int64(len(slices)),
	})
}

// GetSlice 获取提示词块详情
// @Summary 获取提示词块
// @Tags prompt
// @Produce json
// @Param id path int true "提示词块ID"
// @Success 200 {object} SliceResponse
// @Router /api/slices/{id} [get]
func (s *PromptService) GetSlice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 格式错误"})
		return
	}
	slice, err := s.sliceBiz.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, toSliceResponse(slice))
}

// UpdateSlice 更新提示词块
// @Summary 更新提示词块
// @Tags prompt
// @Accept json
// @Produce json
// @Param id path int true "提示词块ID"
// @Param request body UpdateSliceRequest true "更新请求"
// @Success 200 {object} SliceResponse
// @Router /api/slices/{id} [put]
func (s *PromptService) UpdateSlice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 格式错误"})
		return
	}

	var req UpdateSliceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	slice, err := s.sliceBiz.Update(uint(id), req.Content, req.TranslatedContent, req.OriginLanguage, req.TargetLanguage)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, toSliceResponse(slice))
}

// DeleteSlice 删除提示词块
// @Summary 删除提示词块
// @Tags prompt
// @Produce json
// @Param id path int true "提示词块ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/slices/{id} [delete]
func (s *PromptService) DeleteSlice(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 格式错误"})
		return
	}

	if err := s.sliceBiz.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ============================================================
// 活动 Prompt（草稿）处理器
// ============================================================

// GetActivePrompt 获取当前活动 Prompt
// @Summary 获取活动 Prompt
// @Tags prompt
// @Produce json
// @Success 200 {object} ActivePromptResponse
// @Router /api/active-prompt [get]
func (s *PromptService) GetActivePrompt(c *gin.Context) {
	active, err := s.draftBiz.GetActive()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	if active == nil {
		c.JSON(http.StatusOK, ActivePromptResponse{})
		return
	}

	// 补全 Slice 的 Content 和 TranslatedContent（兼容老数据或前端遗漏）
	for i := range active.Regions {
		for j := range active.Regions[i].Slices {
			dto := &active.Regions[i].Slices[j]
			if dto.Content == "" {
				if sl, err := s.sliceBiz.GetByID(dto.SliceID); err == nil {
					dto.Content = sl.Content
					dto.TranslatedContent = sl.TranslatedContent
				}
			}
		}
	}

	c.JSON(http.StatusOK, ActivePromptResponse{
		Title:     active.Title,
		Regions:   active.Regions,
		UpdatedAt: active.UpdatedAt,
	})
}

// UpdateActivePrompt 更新活动 Prompt
// @Summary 更新活动 Prompt
// @Tags prompt
// @Accept json
// @Produce json
// @Param request body UpdateActivePromptRequest true "更新请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/active-prompt [put]
func (s *PromptService) UpdateActivePrompt(c *gin.Context) {
	var req UpdateActivePromptRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数错误"})
		return
	}

	if err := s.draftBiz.SetActive(req.Title, req.Regions); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// ============================================================
// 记录处理器
// ============================================================

// PersistRecord 持久化活动 Prompt 为记录
// @Summary 持久化记录
// @Tags prompt
// @Produce json
// @Param uuid path string true "ComfyUI生成的UUID"
// @Success 200 {object} PersistRecordResponse
// @Router /api/records/{uuid} [post]
func (s *PromptService) PersistRecord(c *gin.Context) {
	uuid := c.Param("uuid")
	record, err := s.recordBiz.PersistFromActive(uuid)
	if err != nil {
		if errors.Is(err, ErrNoActivePrompt) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "没有活动的Prompt"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "持久化失败"})
		return
	}

	c.JSON(http.StatusOK, toPersistRecordResponse(record))
}

// GetRecord 获取记录详情
// @Summary 获取记录详情
// @Tags prompt
// @Produce json
// @Param id path int true "记录ID"
// @Success 200 {object} RecordResponse
// @Router /api/records/{id} [get]
func (s *PromptService) GetRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 格式错误"})
		return
	}

	record, err := s.recordBiz.GetByID(uint(id))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	// 获取 Record 下的 Region 列表（已按 SortOrder 升序排列）
	recordRegions, err := s.recordBiz.GetRegionsByRecordID(record.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询 Region 失败"})
		return
	}

	// 逐 Region 重建嵌套响应结构（Region → Slice 树）
	regions := make([]RecordRegionResponse, 0, len(recordRegions))
	for _, rr := range recordRegions {
		// 查询 Region 名称（源表 PromptRegion）
		regionName := ""
		if r, err := s.regionBiz.GetByID(rr.RegionID); err == nil {
			regionName = r.Name
		}

		// 查询该 Region 下的所有 Slice（已按 SortOrder 升序排列）
		regionSliceRefs, err := s.recordBiz.GetRegionSlices(rr.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询切片失败"})
			return
		}

		// 构建 Slice 响应列表（解析 Slice 原文，处理自定义文本覆盖）
		sliceResponses := make([]RecordSliceResponse, 0, len(regionSliceRefs))
		for _, rs := range regionSliceRefs {
			// 从源表查询 Slice 原文
			content := ""
			customText := rs.CustomText
			if sl, err := s.sliceBiz.GetByID(rs.SliceID); err == nil {
				content = sl.Content
			}
			// 若有自定义文本覆盖，使用自定义文本
			if customText != nil {
				content = *customText
			}

			sliceResponses = append(sliceResponses, RecordSliceResponse{
				SliceID:    rs.SliceID,
				Content:    content,
				CustomText: customText,
				SortOrder:  rs.SortOrder,
			})
		}

		regions = append(regions, RecordRegionResponse{
			RegionID:   rr.RegionID,
			RegionName: regionName,
			SortOrder:  rr.SortOrder,
			Slices:     sliceResponses,
		})
	}

	c.JSON(http.StatusOK, RecordResponse{
		ID:          record.ID,
		ExternalID:  record.ExternalID,
		Title:       record.Title,
		FullContent: record.FullContent,
		Regions:     regions,
		CreatedAt:   record.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   record.UpdatedAt.Format("2006-01-02 15:04:05"),
	})
}

// ListRecords 获取记录列表
// @Summary 获取记录列表
// @Tags prompt
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} ListRecordResponse
// @Router /api/records [get]
func (s *PromptService) ListRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	records, total, err := s.recordBiz.List(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	responses := make([]*RecordResponse, 0, len(records))
	for _, r := range records {
		responses = append(responses, &RecordResponse{
			ID:          r.ID,
			ExternalID:  r.ExternalID,
			Title:       r.Title,
			FullContent: r.FullContent,
			CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04:05"),
			UpdatedAt:   r.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	c.JSON(http.StatusOK, ListRecordResponse{
		List:  responses,
		Total: total,
		Page:  page,
	})
}

// DeleteRecord 删除记录
// @Summary 删除记录
// @Tags prompt
// @Produce json
// @Param id path int true "记录ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/records/{id} [delete]
func (s *PromptService) DeleteRecord(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID 格式错误"})
		return
	}

	if err := s.recordBiz.Delete(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// ============================================================
// SliceType 处理器
// ============================================================

// GetSliceTypeTree 获取语义分类树
// @Summary 获取语义分类树
// @Tags prompt
// @Produce json
// @Success 200 {object} SliceTypeTreeResponse
// @Router /api/slice-types [get]
func (s *PromptService) GetSliceTypeTree(c *gin.Context) {
	types, err := s.sliceTypeBiz.GetTree()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	c.JSON(http.StatusOK, SliceTypeTreeResponse{Types: types})
}

// ============================================================
// Combo 处理器
// ============================================================

// GetComboTree 获取类别与提示词块树形结构
// @Summary 获取组合树
// @Tags prompt
// @Produce json
// @Success 200 {object} ComboTreeResponse
// @Router /api/combo/tree [get]
func (s *PromptService) GetComboTree(c *gin.Context) {
	regions, err := s.regionBiz.List()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	regionDTOs := make([]ComboRegionDTO, 0, len(regions))
	for _, region := range regions {
		slices, err := s.sliceBiz.ListByRegion(region.ID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "查询切片失败"})
			return
		}

		sliceDTOs := make([]ComboSliceDTO, 0, len(slices))
		for _, sl := range slices {
			sliceDTOs = append(sliceDTOs, ComboSliceDTO{
				ID:                sl.ID,
				Content:           sl.Content,
				TranslatedContent: sl.TranslatedContent,
				OriginLanguage:    string(sl.OriginLanguage),
				TargetLanguage:    string(sl.TargetLanguage),
				SortOrder:         0,
			})
		}

		regionDTOs = append(regionDTOs, ComboRegionDTO{
			ID:          region.ID,
			Name:        region.Name,
			SortOrder:   region.SortOrder,
			Description: region.Description,
			Slices:      sliceDTOs,
		})
	}

	c.JSON(http.StatusOK, ComboTreeResponse{
		Regions: regionDTOs,
	})
}

// ============================================================
// 辅助函数
// ============================================================

// toRegionResponse 将 model.PromptRegion 转换为 RegionResponse
func toRegionResponse(r *model.PromptRegion) *RegionResponse {
	return &RegionResponse{
		ID:          r.ID,
		Name:        r.Name,
		SortOrder:   r.SortOrder,
		Description: r.Description,
		CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   r.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// toSliceResponse 将 model.PromptSlice 转换为 SliceResponse
func toSliceResponse(s *model.PromptSlice) *SliceResponse {
	return &SliceResponse{
		ID:                s.ID,
		Content:           s.Content,
		TranslatedContent: s.TranslatedContent,
		OriginLanguage:    string(s.OriginLanguage),
		TargetLanguage:    string(s.TargetLanguage),
		CreatedAt:         s.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:         s.UpdatedAt.Format("2006-01-02 15:04:05"),
	}
}

// toPersistRecordResponse 将 model.PromptRecord 转换为 PersistRecordResponse
func toPersistRecordResponse(r *model.PromptRecord) *PersistRecordResponse {
	return &PersistRecordResponse{
		ID:          r.ID,
		ExternalID:  r.ExternalID,
		Title:       r.Title,
		FullContent: r.FullContent,
		CreatedAt:   r.CreatedAt.Format("2006-01-02 15:04:05"),
	}
}
