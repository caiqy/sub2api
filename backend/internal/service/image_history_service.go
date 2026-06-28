package service

import (
	"bytes"
	"context"
	"io"
	"mime"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/tidwall/gjson"
)

type ImageHistoryMode string

type ImageHistoryStatus string

const (
	ImageHistoryModeGenerate ImageHistoryMode = "generate"
	ImageHistoryModeEdit     ImageHistoryMode = "edit"

	ImageHistoryStatusSuccess ImageHistoryStatus = "success"
	ImageHistoryStatusError   ImageHistoryStatus = "error"
)

type ImageHistoryListFilters struct {
	APIKeyID int64
	Mode     string
	Status   string
}

type ImageHistoryListQuery struct {
	APIKeyID int64
	Mode     string
	Status   string
	Page     int
	PageSize int
}

type ImageHistoryListItem struct {
	UsageLogID   int64
	APIKeyID     int64
	APIKeyName   string
	APIKeyMasked string
	Mode         ImageHistoryMode
	Status       ImageHistoryStatus
	Model        string
	Prompt       string
	ImageCount   int
	ImageSize    string
	ActualCost   float64
	DurationMs   *int
	CreatedAt    time.Time
}

type ImageHistoryImage struct {
	DataURL       string
	RevisedPrompt string
}

type ImageHistoryReplay struct {
	Mode                      ImageHistoryMode
	Model                     string
	Prompt                    string
	Size                      string
	Quality                   string
	Background                string
	OutputFormat              string
	Moderation                string
	N                         int
	RequiresSourceImageUpload bool
	RequiresMaskUpload        bool
}

type ImageHistoryDetail struct {
	UsageLogID     int64
	APIKeyID       int64
	APIKeyName     string
	APIKeyMasked   string
	Mode           ImageHistoryMode
	Status         ImageHistoryStatus
	Model          string
	Prompt         string
	Size           string
	Quality        string
	Background     string
	OutputFormat   string
	Moderation     string
	N              int
	HadSourceImage bool
	HadMask        bool
	Images         []ImageHistoryImage
	ErrorMessage   string
	DurationMs     *int
	Replay         ImageHistoryReplay
	CreatedAt      time.Time
}

type ImageHistoryService struct {
	usageRepo UsageLogRepository
}

func NewImageHistoryService(usageRepo UsageLogRepository) *ImageHistoryService {
	return &ImageHistoryService{usageRepo: usageRepo}
}

func (s *ImageHistoryService) List(ctx context.Context, userID int64, query ImageHistoryListQuery) ([]ImageHistoryListItem, *pagination.PaginationResult, error) {
	params := pagination.DefaultPagination()
	if query.Page > 0 {
		params.Page = query.Page
	}
	if query.PageSize > 0 {
		params.PageSize = query.PageSize
	}
	params.SortBy = "created_at"
	params.SortOrder = pagination.SortOrderDesc

	filters := ImageHistoryListFilters{
		APIKeyID: query.APIKeyID,
		Mode:     normalizeImageHistoryModeFilter(query.Mode),
		Status:   normalizeImageHistoryStatusFilter(query.Status),
	}
	logs, page, err := s.usageRepo.ListImageHistoryByUser(ctx, userID, params, filters)
	if err != nil {
		return nil, nil, err
	}

	items := make([]ImageHistoryListItem, 0, len(logs))
	for _, log := range logs {
		item := ImageHistoryListItem{
			UsageLogID: log.ID,
			APIKeyID:   log.APIKeyID,
			Mode:       imageHistoryModeFromEndpoint(log.InboundEndpoint),
			Status:     imageHistoryStatusFromCount(log.ImageCount),
			Model:      imageHistoryModel(log),
			ImageCount: log.ImageCount,
			ActualCost: log.ActualCost,
			CreatedAt:  log.CreatedAt,
		}
		if log.DurationMs != nil {
			durationMs := *log.DurationMs
			item.DurationMs = &durationMs
		}
		if log.ImageSize != nil {
			item.ImageSize = strings.TrimSpace(*log.ImageSize)
		}
		if log.APIKey != nil {
			item.APIKeyName = log.APIKey.Name
			item.APIKeyMasked = maskImageHistoryAPIKey(log.APIKey.Key)
		}
		if detailRow, detailErr := s.usageRepo.GetDetailByUsageLogID(ctx, log.ID); detailErr == nil {
			item.Prompt = parseImageHistoryRequestSnapshot(detailRow).Prompt
		}
		items = append(items, item)
	}

	return items, page, nil
}

func (s *ImageHistoryService) GetDetail(ctx context.Context, userID int64, usageLogID int64) (*ImageHistoryDetail, error) {
	log, err := s.usageRepo.GetByID(ctx, usageLogID)
	if err != nil {
		return nil, err
	}
	if log.UserID != userID {
		return nil, ErrUsageLogNotFound
	}
	if !isImageHistoryEndpoint(log.InboundEndpoint) {
		return nil, ErrUsageLogNotFound
	}

	detailRow, err := s.usageRepo.GetDetailByUsageLogID(ctx, usageLogID)
	if err != nil {
		return nil, err
	}

	parsedRequest := parseImageHistoryRequestSnapshot(detailRow)
	images := parseImageHistoryResponseImages(detailRow, parsedRequest.OutputFormat)
	result := &ImageHistoryDetail{
		UsageLogID:     log.ID,
		APIKeyID:       log.APIKeyID,
		Mode:           imageHistoryModeFromEndpoint(log.InboundEndpoint),
		Status:         imageHistoryStatusFromCount(log.ImageCount),
		Model:          imageHistoryModel(*log),
		Prompt:         parsedRequest.Prompt,
		Size:           parsedRequest.Size,
		Quality:        parsedRequest.Quality,
		Background:     parsedRequest.Background,
		OutputFormat:   parsedRequest.OutputFormat,
		Moderation:     parsedRequest.Moderation,
		N:              parsedRequest.N,
		HadSourceImage: parsedRequest.HadSourceImage,
		HadMask:        parsedRequest.HadMask,
		Images:         images,
		ErrorMessage:   parseImageHistoryErrorMessage(detailRow),
		DurationMs:     log.DurationMs,
		CreatedAt:      log.CreatedAt,
	}
	if log.APIKey != nil {
		result.APIKeyName = log.APIKey.Name
		result.APIKeyMasked = maskImageHistoryAPIKey(log.APIKey.Key)
	}
	result.Replay = ImageHistoryReplay{
		Mode:                      result.Mode,
		Model:                     result.Model,
		Prompt:                    result.Prompt,
		Size:                      result.Size,
		Quality:                   result.Quality,
		Background:                result.Background,
		OutputFormat:              result.OutputFormat,
		Moderation:                result.Moderation,
		N:                         result.N,
		RequiresSourceImageUpload: result.Mode == ImageHistoryModeEdit,
		RequiresMaskUpload:        result.HadMask,
	}

	return result, nil
}

type parsedImageHistoryRequest struct {
	Prompt         string
	Size           string
	Quality        string
	Background     string
	OutputFormat   string
	Moderation     string
	N              int
	HadSourceImage bool
	HadMask        bool
}

func normalizeImageHistoryModeFilter(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case string(ImageHistoryModeGenerate):
		return string(ImageHistoryModeGenerate)
	case string(ImageHistoryModeEdit):
		return string(ImageHistoryModeEdit)
	default:
		return ""
	}
}

func normalizeImageHistoryStatusFilter(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case string(ImageHistoryStatusSuccess):
		return string(ImageHistoryStatusSuccess)
	case string(ImageHistoryStatusError):
		return string(ImageHistoryStatusError)
	default:
		return ""
	}
}

func imageHistoryModeFromEndpoint(endpoint *string) ImageHistoryMode {
	if endpoint == nil {
		return ImageHistoryModeGenerate
	}
	switch normalizeOpenAIImagesEndpointPath(*endpoint) {
	case openAIImagesEditsEndpoint:
		return ImageHistoryModeEdit
	default:
		return ImageHistoryModeGenerate
	}
}

func isImageHistoryEndpoint(endpoint *string) bool {
	switch normalizeOpenAIImagesEndpointPath(strings.TrimSpace(valueOrEmpty(endpoint))) {
	case openAIImagesGenerationsEndpoint, openAIImagesEditsEndpoint:
		return true
	default:
		return false
	}
}

func imageHistoryStatusFromCount(imageCount int) ImageHistoryStatus {
	if imageCount > 0 {
		return ImageHistoryStatusSuccess
	}
	return ImageHistoryStatusError
}

func imageHistoryModel(log UsageLog) string {
	if strings.TrimSpace(log.RequestedModel) != "" {
		return strings.TrimSpace(log.RequestedModel)
	}
	return strings.TrimSpace(log.Model)
}

func maskImageHistoryAPIKey(key string) string {
	trimmed := strings.TrimSpace(key)
	if len(trimmed) <= 10 {
		return trimmed
	}
	return trimmed[:8] + "..." + trimmed[len(trimmed)-4:]
}

func parseImageHistoryRequestSnapshot(detail *UsageLogDetail) parsedImageHistoryRequest {
	if detail == nil {
		return parsedImageHistoryRequest{N: 1}
	}
	body := firstNonEmptyImageHistoryValue(detail.RequestBody, detail.UpstreamRequestBody)
	contentType := extractImageHistoryContentType(firstNonEmptyImageHistoryValue(detail.RequestHeaders, detail.UpstreamRequestHeaders))
	return parseImageHistoryRequestBody(contentType, body)
}

func extractImageHistoryContentType(headers string) string {
	for _, line := range strings.Split(headers, "\n") {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(parts[0]), "content-type") {
			return strings.TrimSpace(parts[1])
		}
	}
	return ""
}

func parseImageHistoryRequestBody(contentType string, body string) parsedImageHistoryRequest {
	out := parsedImageHistoryRequest{N: 1}
	if strings.TrimSpace(body) == "" {
		return out
	}

	if multipartBoundary := detectImageHistoryMultipartBoundary(contentType, body); multipartBoundary != "" {
		if multipartOut, ok := parseImageHistoryMultipartBody(multipartBoundary, body); ok {
			return multipartOut
		}
	}

	parsed := gjson.Parse(body)
	out.Prompt = strings.TrimSpace(parsed.Get("prompt").String())
	out.Size = strings.TrimSpace(parsed.Get("size").String())
	out.Quality = strings.TrimSpace(parsed.Get("quality").String())
	out.Background = strings.TrimSpace(parsed.Get("background").String())
	out.OutputFormat = strings.TrimSpace(parsed.Get("output_format").String())
	out.Moderation = strings.TrimSpace(parsed.Get("moderation").String())
	out.HadSourceImage = parsed.Get("had_source_image").Bool() || parsed.Get("hadSourceImage").Bool()
	out.HadMask = parsed.Get("had_mask").Bool() || parsed.Get("hadMask").Bool()
	if n := int(parsed.Get("n").Int()); n > 0 {
		out.N = n
	}
	return out
}

func parseImageHistoryMultipartBody(boundary string, body string) (parsedImageHistoryRequest, bool) {
	out := parsedImageHistoryRequest{N: 1}
	reader := multipart.NewReader(bytes.NewReader([]byte(body)), boundary)
	sawPart := false
	for {
		part, partErr := reader.NextPart()
		if partErr != nil {
			break
		}
		sawPart = true
		name := imageHistoryMultipartFieldName(part)
		payload, _ := io.ReadAll(part)
		if imageHistoryMultipartIsFileField(name, part, payload) {
			switch name {
			case "image":
				out.HadSourceImage = true
			case "mask":
				out.HadMask = true
			}
			continue
		}
		value := strings.TrimSpace(string(payload))
		switch name {
		case "prompt":
			out.Prompt = value
		case "size":
			out.Size = value
		case "quality":
			out.Quality = value
		case "background":
			out.Background = value
		case "output_format":
			out.OutputFormat = value
		case "moderation":
			out.Moderation = value
		case "n":
			if parsed, parseErr := strconv.Atoi(value); parseErr == nil && parsed > 0 {
				out.N = parsed
			}
		}
	}
	return out, sawPart
}

func imageHistoryMultipartFieldName(part *multipart.Part) string {
	if part == nil {
		return ""
	}
	if name := strings.TrimSpace(part.FormName()); name != "" {
		return name
	}
	mediaType, params, err := mime.ParseMediaType(part.Header.Get("Content-Disposition"))
	if err == nil && strings.EqualFold(mediaType, "form-data") {
		return strings.TrimSpace(params["name"])
	}
	return ""
}

func imageHistoryMultipartIsFileField(name string, part *multipart.Part, payload []byte) bool {
	switch name {
	case "image", "mask":
		if part != nil && strings.TrimSpace(part.FileName()) != "" {
			return true
		}
		if part != nil && strings.HasPrefix(strings.ToLower(strings.TrimSpace(part.Header.Get("Content-Type"))), "image/") {
			return true
		}
		return len(payload) > 0
	default:
		return part != nil && strings.TrimSpace(part.FileName()) != ""
	}
}

func detectImageHistoryMultipartBoundary(contentType string, body string) string {
	if mediaType, params, err := mime.ParseMediaType(contentType); err == nil && strings.EqualFold(mediaType, "multipart/form-data") {
		if boundary := strings.TrimSpace(params["boundary"]); boundary != "" {
			return boundary
		}
	}
	if boundary := extractImageHistoryBoundaryToken(contentType); boundary != "" {
		return boundary
	}
	return extractImageHistoryBoundaryFromBody(body)
}

func extractImageHistoryBoundaryToken(contentType string) string {
	trimmed := strings.TrimSpace(contentType)
	if trimmed == "" {
		return ""
	}
	idx := strings.Index(strings.ToLower(trimmed), "boundary=")
	if idx < 0 {
		return ""
	}
	boundary := strings.TrimSpace(trimmed[idx+len("boundary="):])
	boundary = strings.Trim(boundary, `"'`)
	if cut := strings.IndexAny(boundary, ";, "); cut >= 0 {
		boundary = boundary[:cut]
	}
	return strings.TrimSpace(boundary)
}

func extractImageHistoryBoundaryFromBody(body string) string {
	trimmed := strings.TrimSpace(body)
	if !strings.HasPrefix(trimmed, "--") {
		return ""
	}
	for _, sep := range []string{"\r\n", "\n"} {
		if idx := strings.Index(trimmed, sep); idx > 2 {
			return strings.TrimSpace(strings.TrimPrefix(trimmed[:idx], "--"))
		}
	}
	return ""
}

func parseImageHistoryResponseImages(detail *UsageLogDetail, outputFormat string) []ImageHistoryImage {
	body := imageHistoryResponseBody(detail)
	if strings.TrimSpace(body) == "" {
		return nil
	}

	data := gjson.Get(body, "data")
	if !data.Exists() || !data.IsArray() {
		return nil
	}

	images := make([]ImageHistoryImage, 0, len(data.Array()))
	for _, item := range data.Array() {
		b64 := strings.TrimSpace(item.Get("b64_json").String())
		if b64 == "" {
			continue
		}
		images = append(images, ImageHistoryImage{
			DataURL:       "data:" + imageHistoryResponseMimeType(item, outputFormat) + ";base64," + b64,
			RevisedPrompt: strings.TrimSpace(item.Get("revised_prompt").String()),
		})
	}
	return images
}

func imageHistoryResponseMimeType(item gjson.Result, outputFormat string) string {
	for _, candidate := range []string{
		strings.TrimSpace(item.Get("mime_type").String()),
		strings.TrimSpace(item.Get("content_type").String()),
		strings.TrimSpace(item.Get("output_format").String()),
		strings.TrimSpace(outputFormat),
	} {
		if mimeType := normalizeImageHistoryMimeType(candidate); mimeType != "" {
			return mimeType
		}
	}
	return "image/png"
}

func normalizeImageHistoryMimeType(value string) string {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" {
		return ""
	}
	if strings.HasPrefix(trimmed, "image/") {
		if cut := strings.Index(trimmed, ";"); cut >= 0 {
			return strings.TrimSpace(trimmed[:cut])
		}
		return trimmed
	}
	switch trimmed {
	case "jpg":
		trimmed = "jpeg"
	}
	if mimeType := mime.TypeByExtension("." + trimmed); mimeType != "" {
		if cut := strings.Index(mimeType, ";"); cut >= 0 {
			return strings.TrimSpace(mimeType[:cut])
		}
		return strings.TrimSpace(mimeType)
	}
	return ""
}

func parseImageHistoryErrorMessage(detail *UsageLogDetail) string {
	body := imageHistoryResponseBody(detail)
	if strings.TrimSpace(body) == "" {
		return ""
	}
	for _, path := range []string{"error.message", "error", "message"} {
		value := strings.TrimSpace(gjson.Get(body, path).String())
		if value != "" {
			return value
		}
	}
	return ""
}

func imageHistoryResponseBody(detail *UsageLogDetail) string {
	if detail == nil {
		return ""
	}
	return firstNonEmptyImageHistoryValue(detail.ResponseBody, detail.UpstreamResponseBody)
}

func firstNonEmptyImageHistoryValue(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func valueOrEmpty(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
