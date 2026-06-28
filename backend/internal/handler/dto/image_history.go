package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

type ImageHistoryListItem struct {
	ID           int64     `json:"id"`
	APIKeyID     int64     `json:"api_key_id"`
	APIKeyName   string    `json:"api_key_name,omitempty"`
	APIKeyMasked string    `json:"api_key_masked,omitempty"`
	Mode         string    `json:"mode"`
	Status       string    `json:"status"`
	Model        string    `json:"model"`
	Prompt       string    `json:"prompt,omitempty"`
	ImageCount   int       `json:"image_count"`
	ImageSize    string    `json:"image_size,omitempty"`
	ActualCost   float64   `json:"actual_cost"`
	DurationMs   *int      `json:"duration_ms,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type ImageHistoryImage struct {
	DataURL       string `json:"data_url"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

type ImageHistoryReplay struct {
	Mode                      string `json:"mode"`
	Model                     string `json:"model"`
	Prompt                    string `json:"prompt,omitempty"`
	Size                      string `json:"size,omitempty"`
	Quality                   string `json:"quality,omitempty"`
	Background                string `json:"background,omitempty"`
	OutputFormat              string `json:"output_format,omitempty"`
	Moderation                string `json:"moderation,omitempty"`
	N                         int    `json:"n"`
	RequiresSourceImageUpload bool   `json:"requires_source_image_upload"`
	RequiresMaskUpload        bool   `json:"requires_mask_upload"`
}

type ImageHistoryDetail struct {
	ID             int64               `json:"id"`
	APIKeyID       int64               `json:"api_key_id"`
	APIKeyName     string              `json:"api_key_name,omitempty"`
	APIKeyMasked   string              `json:"api_key_masked,omitempty"`
	Mode           string              `json:"mode"`
	Status         string              `json:"status"`
	Model          string              `json:"model"`
	Prompt         string              `json:"prompt,omitempty"`
	Size           string              `json:"size,omitempty"`
	Quality        string              `json:"quality,omitempty"`
	Background     string              `json:"background,omitempty"`
	OutputFormat   string              `json:"output_format,omitempty"`
	Moderation     string              `json:"moderation,omitempty"`
	N              int                 `json:"n"`
	HadSourceImage bool                `json:"had_source_image"`
	HadMask        bool                `json:"had_mask"`
	Images         []ImageHistoryImage `json:"images,omitempty"`
	ErrorMessage   string              `json:"error_message,omitempty"`
	DurationMs     *int                `json:"duration_ms,omitempty"`
	Replay         ImageHistoryReplay  `json:"replay"`
	CreatedAt      time.Time           `json:"created_at"`
}

func ImageHistoryListItemFromService(item *service.ImageHistoryListItem) *ImageHistoryListItem {
	if item == nil {
		return nil
	}
	return &ImageHistoryListItem{
		ID:           item.UsageLogID,
		APIKeyID:     item.APIKeyID,
		APIKeyName:   item.APIKeyName,
		APIKeyMasked: item.APIKeyMasked,
		Mode:         string(item.Mode),
		Status:       string(item.Status),
		Model:        item.Model,
		Prompt:       item.Prompt,
		ImageCount:   item.ImageCount,
		ImageSize:    item.ImageSize,
		ActualCost:   item.ActualCost,
		DurationMs:   item.DurationMs,
		CreatedAt:    item.CreatedAt,
	}
}

func ImageHistoryDetailFromService(detail *service.ImageHistoryDetail) *ImageHistoryDetail {
	if detail == nil {
		return nil
	}
	images := make([]ImageHistoryImage, 0, len(detail.Images))
	for i := range detail.Images {
		images = append(images, ImageHistoryImage{
			DataURL:       detail.Images[i].DataURL,
			RevisedPrompt: detail.Images[i].RevisedPrompt,
		})
	}
	return &ImageHistoryDetail{
		ID:             detail.UsageLogID,
		APIKeyID:       detail.APIKeyID,
		APIKeyName:     detail.APIKeyName,
		APIKeyMasked:   detail.APIKeyMasked,
		Mode:           string(detail.Mode),
		Status:         string(detail.Status),
		Model:          detail.Model,
		Prompt:         detail.Prompt,
		Size:           detail.Size,
		Quality:        detail.Quality,
		Background:     detail.Background,
		OutputFormat:   detail.OutputFormat,
		Moderation:     detail.Moderation,
		N:              detail.N,
		HadSourceImage: detail.HadSourceImage,
		HadMask:        detail.HadMask,
		Images:         images,
		ErrorMessage:   detail.ErrorMessage,
		DurationMs:     detail.DurationMs,
		Replay: ImageHistoryReplay{
			Mode:                      string(detail.Replay.Mode),
			Model:                     detail.Replay.Model,
			Prompt:                    detail.Replay.Prompt,
			Size:                      detail.Replay.Size,
			Quality:                   detail.Replay.Quality,
			Background:                detail.Replay.Background,
			OutputFormat:              detail.Replay.OutputFormat,
			Moderation:                detail.Replay.Moderation,
			N:                         detail.Replay.N,
			RequiresSourceImageUpload: detail.Replay.RequiresSourceImageUpload,
			RequiresMaskUpload:        detail.Replay.RequiresMaskUpload,
		},
		CreatedAt: detail.CreatedAt,
	}
}
