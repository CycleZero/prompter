package prompt

// ============================================================
// Region DTOs
// ============================================================

// CreateRegionRequest 创建类别请求
type CreateRegionRequest struct {
	Name        string `json:"name" binding:"required"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description"`
}

// UpdateRegionRequest 更新类别请求
type UpdateRegionRequest struct {
	Name        string `json:"name"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description"`
}

// RegionResponse 类别响应
type RegionResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	SortOrder   int    `json:"sort_order"`
	Description string `json:"description"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

// ============================================================
// Slice DTOs
// ============================================================

// CreateSliceRequest 创建提示词块请求
type CreateSliceRequest struct {
	Content           string   `json:"content" binding:"required"`
	TranslatedContent string   `json:"translated_content"`
	OriginLanguage    string   `json:"origin_language"`
	TargetLanguage    string   `json:"target_language"`
	RegionIDs         []uint   `json:"region_ids"` // 所属类别列表
}

// UpdateSliceRequest 更新提示词块请求
type UpdateSliceRequest struct {
	Content           string `json:"content"`
	TranslatedContent string `json:"translated_content"`
	OriginLanguage    string `json:"origin_language"`
	TargetLanguage    string `json:"target_language"`
}

// SliceResponse 提示词块响应
type SliceResponse struct {
	ID                uint   `json:"id"`
	Content           string `json:"content"`
	TranslatedContent string `json:"translated_content"`
	OriginLanguage    string `json:"origin_language"`
	TargetLanguage    string `json:"target_language"`
	CreatedAt         string `json:"created_at"`
	UpdatedAt         string `json:"updated_at"`
}

// SliceListResponse 提示词块列表响应
type SliceListResponse struct {
	List  []*SliceResponse `json:"list"`
	Total int64            `json:"total"`
}

// ============================================================
// Active Prompt (Draft) DTOs — Redis 单一活动 Prompt
// ============================================================

// ActiveSliceDTO 活动 Prompt 中某个 Region 下的单个 Slice 引用
type ActiveSliceDTO struct {
	SliceID            uint    `json:"slice_id"`
	Content            string  `json:"content"`             // 原文（供前端直观显示）
	TranslatedContent  string  `json:"translated_content"`  // 翻译文本（供前端直观显示）
	CustomText         *string `json:"custom_text"`
	SortOrder          int     `json:"sort_order"` // 用户在 Region 内拖拽的顺序
}

// ActivePromptRegionDTO 活动 Prompt 中的 Region 分组（含其下 Slices）
type ActivePromptRegionDTO struct {
	RegionID   uint             `json:"region_id"`
	RegionName string           `json:"region_name"`
	SortOrder  int              `json:"sort_order"` // 用户拖拽 Region 的顺序
	Slices     []ActiveSliceDTO `json:"slices"`
}

// UpdateActivePromptRequest 更新活动 Prompt 请求
type UpdateActivePromptRequest struct {
	Title   string                   `json:"title"`
	Regions []ActivePromptRegionDTO  `json:"regions" binding:"required"`
}

// ActivePromptResponse 活动 Prompt 响应
type ActivePromptResponse struct {
	Title     string                   `json:"title"`
	Regions   []ActivePromptRegionDTO  `json:"regions"`
	UpdatedAt string                   `json:"updated_at"`
}

// ============================================================
// Record DTOs
// ============================================================

// RecordSliceResponse 记录详情中单个 Slice 的展开（快照数据）
type RecordSliceResponse struct {
	SliceID           uint    `json:"slice_id"`
	Content           string  `json:"content"`
	TranslatedContent string  `json:"translated_content"`
	CustomText        *string `json:"custom_text"`
	SortOrder         int     `json:"sort_order"`
}

// RecordRegionResponse 记录详情中的 Region 分组
type RecordRegionResponse struct {
	RegionID   uint                  `json:"region_id"`
	RegionName string                `json:"region_name"`
	SortOrder  int                   `json:"sort_order"`
	Slices     []RecordSliceResponse `json:"slices"`
}

// RecordResponse 记录响应
type RecordResponse struct {
	ID          uint                    `json:"id"`
	ExternalID  string                  `json:"external_id"`
	Title       string                  `json:"title"`
	FullContent string                  `json:"full_content"`
	Regions     []RecordRegionResponse  `json:"regions,omitempty"`
	CreatedAt   string                  `json:"created_at"`
	UpdatedAt   string                  `json:"updated_at"`
}

// PersistRecordResponse 持久化记录响应（ComfyUI 调用返回）
type PersistRecordResponse struct {
	ID          uint   `json:"id"`
	ExternalID  string `json:"external_id"`
	Title       string `json:"title"`
	FullContent string `json:"full_content"`
	CreatedAt   string `json:"created_at"`
}

// ListRecordResponse 记录列表响应
type ListRecordResponse struct {
	List  []*RecordResponse `json:"list"`
	Total int64             `json:"total"`
	Page  int               `json:"page"`
}

// ============================================================
// SliceType DTOs
// ============================================================

// SliceTypeResponse 语义分类响应
type SliceTypeResponse struct {
	ID        uint                `json:"id"`
	Name      string              `json:"name"`
	ParentID  *uint               `json:"parent_id"`
	SortOrder int                 `json:"sort_order"`
	Children  []*SliceTypeResponse `json:"children,omitempty"`
}

// SliceTypeTreeResponse 分类树响应
type SliceTypeTreeResponse struct {
	Types []*SliceTypeResponse `json:"types"`
}

// ============================================================
// Combo Tree DTO
// ============================================================

// ComboSliceDTO 树形结构中的 Slice 节点
type ComboSliceDTO struct {
	ID                uint   `json:"id"`
	Content           string `json:"content"`
	TranslatedContent string `json:"translated_content"`
	OriginLanguage    string `json:"origin_language"`
	TargetLanguage    string `json:"target_language"`
	SortOrder         int    `json:"sort_order"`
}

// ComboRegionDTO 树形结构中的 Region 节点
type ComboRegionDTO struct {
	ID          uint            `json:"id"`
	Name        string          `json:"name"`
	SortOrder   int             `json:"sort_order"`
	Description string          `json:"description"`
	Slices      []ComboSliceDTO `json:"slices"`
}

// ComboTreeResponse 完整树响应
type ComboTreeResponse struct {
	Regions []ComboRegionDTO `json:"regions"`
}
