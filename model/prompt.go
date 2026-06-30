package model

import "gorm.io/gorm"

// Language 提示词语言类型
type Language string

const (
	Chinese Language = "chinese"
	English Language = "english"
)

// PromptRegion 提示词类别/分区（如：主题、风格、质量词、负面词）
type PromptRegion struct {
	gorm.Model
	Name        string `gorm:"type:varchar(100);uniqueIndex;not null;comment:类别名称" json:"name"`
	SortOrder   int    `gorm:"type:int;default:0;comment:排序" json:"sort_order"`
	Description string `gorm:"type:text;comment:描述" json:"description"`
}

// PromptSlice 提示词块（可复用的文本片段）
type PromptSlice struct {
	gorm.Model
	TypeID            *uint    `gorm:"index;comment:语义分类ID" json:"type_id"`
	Content           string   `gorm:"type:text;not null;comment:原文内容" json:"content"`
	TranslatedContent string   `gorm:"type:text;comment:翻译后内容" json:"translated_content"`
	OriginLanguage    Language `gorm:"type:varchar(20);default:chinese;comment:源语言" json:"origin_language"`
	TargetLanguage    Language `gorm:"type:varchar(20);default:english;comment:目标语言" json:"target_language"`
}

// SliceType 语义分类（外部标签体系的分类，如 WeiLin 的"人物→头发"、"画面→画质"）
type SliceType struct {
	gorm.Model
	Name      string `gorm:"type:varchar(100);uniqueIndex;not null;comment:分类名称" json:"name"`
	ParentID  *uint  `gorm:"index;comment:父分类ID(二级分类)" json:"parent_id"`
	SortOrder int    `gorm:"type:int;default:0;comment:排序" json:"sort_order"`
}

type PromptRecord struct {
	gorm.Model
	ExternalID  string `gorm:"type:varchar(64);uniqueIndex;not null;comment:ComfyUI传入的UUID" json:"external_id"`
	Title       string `gorm:"type:varchar(255);comment:标题" json:"title"`
	FullContent string `gorm:"type:longtext;comment:保存时的完整文本快照" json:"full_content"`
}

// ActivePrompt 活动 Prompt 的 MySQL 持久化备份（单行表，id=1，Redis 丢失时兜底）
type ActivePrompt struct {
	gorm.Model
	Data string `gorm:"type:longtext;comment:ActivePromptData的JSON序列化" json:"data"`
}

// PromptRecordRegion 记录中的 Region 分组（Record 下的一级实体）
// 每个 Region 代表用户在界面上看到的一个可拖拽分块
type PromptRecordRegion struct {
	gorm.Model
	RecordID  uint `gorm:"index;not null;comment:所属记录ID" json:"record_id"`
	RegionID  uint `gorm:"not null;comment:用户分组的Region标识" json:"region_id"`
	SortOrder int  `gorm:"type:int;default:0;comment:Region在Record中的显示顺序" json:"sort_order"`
}

// PromptRecordRegionSlice 记录中某个 Region 下的 Slice 引用（快照）
// 保存持久化时刻的 Slice 原文和翻译，即使后续 Slice 被修改，Record 内容不变
type PromptRecordRegionSlice struct {
	gorm.Model
	RecordRegionID    uint    `gorm:"index;not null;comment:所属RecordRegion的ID" json:"record_region_id"`
	SliceID           uint    `gorm:"not null;comment:引用的Slice ID" json:"slice_id"`
	SortOrder         int     `gorm:"type:int;default:0;comment:Slice在Region内的顺序" json:"sort_order"`
	Content           string  `gorm:"type:text;comment:持久化时的原文快照" json:"content"`
	TranslatedContent string  `gorm:"type:text;comment:持久化时的翻译快照" json:"translated_content"`
	CustomText        *string `gorm:"type:text;comment:用户覆盖的文本(NULL=使用原文)" json:"custom_text"`
}
