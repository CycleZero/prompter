package prompt

import "time"

// DraftBiz 业务逻辑层 - 处理 Redis 活动 Prompt 的读写
type DraftBiz struct {
	draftRepo *DraftRepo
}

func NewDraftBiz(draftRepo *DraftRepo) *DraftBiz {
	return &DraftBiz{draftRepo: draftRepo}
}

// GetActive 获取当前活动 Prompt，无活动时返回 nil
func (b *DraftBiz) GetActive() (*ActivePromptData, error) {
	return b.draftRepo.GetActive()
}

// SetActive 更新当前活动 Prompt
func (b *DraftBiz) SetActive(title string, regions []ActivePromptRegionDTO) error {
	data := &ActivePromptData{
		Title:     title,
		Regions:   regions,
		UpdatedAt: time.Now().UTC().Format("2006-01-02 15:04:05"),
	}
	return b.draftRepo.SetActive(data)
}
