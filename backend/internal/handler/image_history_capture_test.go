package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/handler/dto"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestImageHistoryCapture_MetadataSurvivesReleasedRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	prompt := strings.Repeat("星🌟", 150)
	image, err := base64.StdEncoding.DecodeString("iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVQIHWP4z8DwHwAFgAI/ScLbtAAAAABJRU5ErkJggg==")
	require.NoError(t, err)
	for _, test := range []struct {
		name      string
		multipart bool
		failed    bool
	}{{"multipart-success", true, false}, {"multipart-failed", true, true}, {"json-edit-success", false, false}, {"json-edit-failed", false, true}} {
		t.Run(test.name, func(t *testing.T) {
			var body bytes.Buffer
			contentType := "application/json"
			if test.multipart {
				writer := multipart.NewWriter(&body)
				for name, value := range map[string]string{"model": "gpt-image-2", "prompt": prompt, "size": "1536x1024", "quality": "high", "background": "transparent", "output_format": "webp", "moderation": "low", "n": "2"} {
					require.NoError(t, writer.WriteField(name, value))
				}
				for _, field := range []string{"image[]", "mask"} {
					part, err := writer.CreateFormFile(field, "private.png")
					require.NoError(t, err)
					_, err = part.Write(append(bytes.Clone(image), []byte("PRIVATE-SOURCE-BYTES")...))
					require.NoError(t, err)
				}
				require.NoError(t, writer.Close())
				contentType = writer.FormDataContentType()
			} else {
				dataURL := "data:image/png;base64," + base64.StdEncoding.EncodeToString(append(bytes.Clone(image), []byte("PRIVATE-SOURCE-BYTES")...))
				require.NoError(t, json.NewEncoder(&body).Encode(map[string]any{
					"model": "gpt-image-2", "prompt": prompt, "size": "1536x1024", "quality": "high", "background": "transparent", "output_format": "webp", "moderation": "low", "n": 2,
					"images": []map[string]string{{"image_url": dataURL}}, "mask": map[string]string{"image_url": dataURL},
				}))
			}
			size := int64(body.Len())
			stream := "event: image_edit.partial_image\ndata: {\"b64_json\":\"UEFSVElBTA==\"}\n\nevent: image_edit.completed\ndata: {\"url\":\"https://example.invalid/final.webp\",\"output_format\":\"webp\"}\n\n"
			var snapshot *middleware2.UsageDetailSnapshot
			var parsed *service.OpenAIImagesRequest
			router := gin.New()
			router.Use(middleware2.UsageDetailCapture())
			router.POST("/v1/images/edits", func(c *gin.Context) {
				var coordinator *requestBodyCoordinator
				var err error
				if test.multipart {
					coordinator, err = newMultipartRequestBody(c.Request, 0)
					require.NoError(t, err)
					parsed, err = (&service.OpenAIGatewayService{}).ParseOpenAIImagesMultipartForm(c, coordinator.form)
				} else {
					coordinator, err = newJSONRequestBody(c.Request)
					require.NoError(t, err)
					raw, readErr := coordinator.ReadRaw()
					require.NoError(t, readErr)
					parsed, err = (&service.OpenAIGatewayService{}).ParseOpenAIImagesRequest(c, raw)
				}
				require.NoError(t, err)
				defer coordinator.Cleanup()
				service.SetUsageOriginalRequestBody(c, "PRIVATE-ORIGINAL-PREVIEW")
				captureOpenAIImagesRequestMetadata(c, parsed, &size)
				coordinator.ReleaseMultipartValues()
				parsed.ReleaseText()
				require.Empty(t, parsed.Prompt)
				require.Empty(t, parsed.InputImageURLs)
				require.Empty(t, parsed.MaskImageURL)
				if test.failed {
					c.Data(http.StatusBadGateway, "application/json", []byte(`{"error":{"message":"upstream failed"}}`))
				} else {
					c.Header("Content-Type", "text/event-stream")
					_, err = c.Writer.WriteString(stream)
					require.NoError(t, err)
				}
				snapshot = middleware2.BuildUsageDetailSnapshot(c)
			})
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/v1/images/edits", &body)
			request.Header.Set("Content-Type", contentType)
			router.ServeHTTP(recorder, request)
			require.NotNil(t, snapshot)
			require.NotNil(t, snapshot.RequestBodySize)
			require.Equal(t, size, *snapshot.RequestBodySize)
			require.NotContains(t, snapshot.RequestBody, "PRIVATE-")
			require.NotContains(t, snapshot.RequestBody, "private.png")
			require.NotContains(t, snapshot.RequestBody, "data:image")
			var envelope struct {
				Preview string `json:"preview"`
			}
			require.NoError(t, json.Unmarshal([]byte(snapshot.RequestBody), &envelope))
			var metadata struct {
				Prompt    string `json:"prompt"`
				HadSource bool   `json:"had_source_image"`
				HadMask   bool   `json:"had_mask"`
			}
			require.NoError(t, json.Unmarshal([]byte(envelope.Preview), &metadata))
			require.Equal(t, prompt, metadata.Prompt)
			require.True(t, metadata.HadSource)
			require.True(t, metadata.HadMask)
			if test.failed {
				require.Contains(t, snapshot.ResponseHeaders, "502")
				require.Contains(t, snapshot.ResponseHeaders, "application/json")
				require.JSONEq(t, `{"error":{"message":"upstream failed"}}`, snapshot.ResponseBody)
				return
			}
			require.Equal(t, stream, snapshot.ResponseBody)
			require.Contains(t, snapshot.ResponseHeaders, "text/event-stream")
			repo := &imageHistoryHandlerRepoStub{
				logs:    []service.UsageLog{{ID: 1, UserID: 7, HasDetail: true, ImageCount: 1, InboundEndpoint: imageHistoryStringPtr("/v1/images/edits")}},
				details: map[int64]*service.UsageLogDetail{1: {RequestBody: snapshot.RequestBody, RequestHeaders: snapshot.RequestHeaders, ResponseBody: snapshot.ResponseBody}},
			}
			svc := service.NewImageHistoryService(repo)
			items, _, err := svc.List(context.Background(), 7, service.ImageHistoryListQuery{})
			require.NoError(t, err)
			require.True(t, dto.ImageHistoryListItemFromService(&items[0]).SummaryAvailable)
			require.Len(t, []rune(items[0].Prompt), 256)
			detail, err := svc.GetDetail(context.Background(), 7, 1)
			require.NoError(t, err)
			require.Equal(t, prompt, detail.Prompt)
			require.Equal(t, parsed.Size, detail.Size)
			require.Equal(t, parsed.Quality, detail.Quality)
			require.Equal(t, parsed.Background, detail.Background)
			require.Equal(t, parsed.Moderation, detail.Moderation)
			require.Equal(t, parsed.N, detail.N)
			require.True(t, detail.Replay.RequiresSourceImageUpload)
			require.True(t, detail.Replay.RequiresMaskUpload)
			wire := dto.ImageHistoryDetailFromService(detail)
			require.Len(t, wire.Images, 1)
			raw, err := json.Marshal(wire.Images[0])
			require.NoError(t, err)
			require.JSONEq(t, `{"url":"https://example.invalid/final.webp"}`, string(raw))
		})
	}
}
