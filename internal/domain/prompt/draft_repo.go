package prompt

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"prompter/infra"
	"prompter/model"

	"gorm.io/gorm"
)

const activePromptKey = "active_prompt"

// ActivePromptData 活动 Prompt 数据结构，同时用于 Redis 缓存和 MySQL 持久化
type ActivePromptData struct {
	Title     string                   `json:"title"`
	Regions   []ActivePromptRegionDTO  `json:"regions"`
	UpdatedAt string                   `json:"updated_at"`
}

// DraftRepo 活动 Prompt 数据访问层 — 双写 Redis + MySQL
// Redis 为主（低延迟读写），MySQL 为 Redis 意外清空时的兜底
type DraftRepo struct {
	data *infra.Data
}

// NewDraftRepo 创建 DraftRepo，同时对 ActivePrompt 表执行自动迁移
func NewDraftRepo(data *infra.Data) *DraftRepo {
	if data.DB != nil {
		if err := data.DB.AutoMigrate(&model.ActivePrompt{}); err != nil {
			panic(err)
		}
	}
	return &DraftRepo{data: data}
}

// GetActive 获取当前活动 Prompt
// 读取策略：Redis 优先（低延迟）→ 任何失败（含 Redis 不可用）回源 MySQL
// 只有 Redis 和 MySQL 均失败时才返回 error
func (r *DraftRepo) GetActive() (*ActivePromptData, error) {
	// 优先读 Redis
	var data ActivePromptData
	err := r.data.RedisClient.GetObject(context.Background(), activePromptKey, &data)
	if err == nil {
		return &data, nil
	}

	// Redis 不可用或未命中 → MySQL 兜底
	if r.data.DB == nil {
		return nil, nil
	}
	var ap model.ActivePrompt
	dbErr := r.data.DB.First(&ap).Error
	if errors.Is(dbErr, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if dbErr != nil {
		return nil, dbErr
	}

	// MySQL 命中 → 反序列化并尝试回写 Redis（忽略回写失败）
	if err := json.Unmarshal([]byte(ap.Data), &data); err != nil {
		return nil, err
	}
	_ = r.data.RedisClient.PutObject(context.Background(), activePromptKey, &data, 7*24*time.Hour)
	return &data, nil
}

// SetActive 写入活动 Prompt — 双写 Redis + MySQL
// 策略：Redis 尽力而为，MySQL 为准。只有 MySQL 写入失败才返回 error
func (r *DraftRepo) SetActive(data *ActivePromptData) error {
	// 先尝试 Redis（忽略失败）
	_ = r.data.RedisClient.PutObject(context.Background(), activePromptKey, data, 7*24*time.Hour)

	// MySQL 是最终保障
	if r.data.DB == nil {
		return nil
	}
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return nil
	}
	ap := model.ActivePrompt{Data: string(jsonBytes)}
	result := r.data.DB.Where(model.ActivePrompt{Model: gorm.Model{ID: 1}}).Assign(ap).FirstOrCreate(&ap)
	return result.Error
}
