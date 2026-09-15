package service

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestOpenAIImagesPreparedBodyAccountMappingAndReplay(t *testing.T) {
	for _, format := range []string{"generation", "generation-stream", "json-edit", "multipart-edit"} {
		for _, mapping := range []struct{ request, channel string }{{"gpt-image-1", ""}, {"gpt-image-2", ""}, {"gpt-image-2", "gpt-image-1"}} {
			t.Run(format+"/"+mapping.request+"/"+mapping.channel, func(t *testing.T) {
				requestModel := mapping.request
				isEdit := strings.Contains(format, "edit")
				body := []byte(`{"model":"` + requestModel + `","prompt":"draw","n":2,"size":"1024x1024","quality":"high","output_format":"png","output_compression":80,"partial_images":1}`)
				c, _ := newOpenAIImagesTestContext(t, body)
				if format == "generation-stream" {
					body = append(body[:len(body)-1], []byte(`,"stream":true}`)...)
				}
				if isEdit {
					c.Request.URL.Path = "/v1/images/edits"
					body = append(body[:len(body)-1], []byte(`,"images":[{"image_url":"data:image/png;base64,AA=="}],"mask":{"image_url":"data:image/png;base64,AQ=="}}`)...)
				}
				if format == "multipart-edit" {
					var buf bytes.Buffer
					writer := multipart.NewWriter(&buf)
					for key, value := range map[string]string{"model": requestModel, "prompt": "draw", "n": "2", "size": "1024x1024", "quality": "high", "output_format": "png", "output_compression": "80", "partial_images": "1"} {
						require.NoError(t, writer.WriteField(key, value))
					}
					for i, field := range []string{"image", "mask"} {
						part, err := writer.CreateFormFile(field, field+".png")
						require.NoError(t, err)
						_, err = part.Write([]byte{byte(i)})
						require.NoError(t, err)
					}
					require.NoError(t, writer.Close())
					body = buf.Bytes()
					c.Request.Header.Set("Content-Type", writer.FormDataContentType())
				}
				svc := newOpenAIImagesTestService(nil)
				parsed, err := svc.ParseOpenAIImagesRequest(c, body)
				require.NoError(t, err)
				prepared, err := svc.PrepareOpenAIImagesOAuthBody(parsed, mapping.channel)
				require.NoError(t, err)
				handle, err := NewRequestBodyHandleFromBytes(prepared, RequestBodyHandleOptions{})
				require.NoError(t, err)
				t.Cleanup(func() { CleanupRequestBodyHandle(handle) })
				BindOpenAIRequestBodyHandle(c, handle)
				seed := parsed.FreezeStickySessionSeed()
				parsed.ReleaseText()
				// 后续账号只能重放缓存，不能依赖仍在内存中的 prompt 或上传内容。
				parsed.Uploads = nil
				parsed.MaskUpload = nil

				for _, route := range []struct {
					model    string
					fallback bool
				}{
					{"gpt-image-2", false},
					{"gpt-image-1", false},
					{"gpt-image-2", true},
				} {
					calls := 0
					svc.httpUpstream = &codexModelsHTTPUpstreamStub{do: func(req *http.Request, _ string, _ int64, _ int) (*http.Response, error) {
						calls++
						payload, err := io.ReadAll(req.Body)
						require.NoError(t, err)
						direct := route.model == "gpt-image-2" && calls == 1
						prefix := "tools.0."
						if direct {
							prefix = ""
							require.Equal(t, strings.TrimSuffix(chatgptCodexURL, "/responses")+strings.TrimPrefix(parsed.Endpoint, "/v1"), req.URL.String())
							require.False(t, gjson.GetBytes(payload, "tools").Exists())
							require.Equal(t, "draw", gjson.GetBytes(payload, "prompt").String())
						} else {
							require.Equal(t, chatgptCodexURL, req.URL.String())
							require.Equal(t, openAIImagesResponsesMainModelValue(), gjson.GetBytes(payload, "model").String())
							require.Equal(t, "draw", gjson.GetBytes(payload, "input.0.content.0.text").String())
						}
						require.Equal(t, route.model, gjson.GetBytes(payload, prefix+"model").String())
						for key, value := range map[string]string{"n": "2", "size": "1024x1024", "quality": "high", "output_format": "png", "output_compression": "80", "partial_images": "1"} {
							require.Equal(t, value, gjson.GetBytes(payload, prefix+key).String(), key)
						}
						if isEdit {
							imagePath, maskPath := "images.0.image_url", "mask.image_url"
							if !direct {
								imagePath, maskPath = "input.0.content.1.image_url", "tools.0.input_image_mask.image_url"
							}
							require.True(t, strings.HasSuffix(gjson.GetBytes(payload, imagePath).String(), ";base64,AA=="))
							require.True(t, strings.HasSuffix(gjson.GetBytes(payload, maskPath).String(), ";base64,AQ=="))
						}
						if direct && route.fallback {
							return &http.Response{StatusCode: http.StatusNotFound, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"endpoint unavailable"}}`))}, nil
						}
						if direct {
							if parsed.Stream {
								require.True(t, gjson.GetBytes(payload, "stream").Bool())
								return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"image_generation.completed\",\"b64_json\":\"aGVsbG8=\"}\n\n"))}, nil
							}
							return openAIImagesJSONResponse(), nil
						}
						return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": {"text/event-stream"}}, Body: io.NopCloser(strings.NewReader("data: {\"type\":\"response.completed\",\"response\":{\"output\":[{\"type\":\"image_generation_call\",\"result\":\"aGVsbG8=\"}]}}\n\n"))}, nil
					}}
					account := directImagesTestAccount()
					mappedModel := requestModel
					if mapping.channel != "" {
						mappedModel = mapping.channel
					}
					account.Credentials["model_mapping"] = map[string]any{mappedModel: route.model}
					result, err := svc.ForwardImages(context.Background(), c, account, nil, parsed, mapping.channel)
					require.NoError(t, err)
					require.Equal(t, route.model, result.UpstreamModel)
					require.Equal(t, seed, parsed.StickySessionSeed())
					wantCalls := 1
					if route.fallback {
						wantCalls = 2
					}
					require.Equal(t, wantCalls, calls)
					replayed, err := handle.ReadAll()
					require.NoError(t, err)
					require.Equal(t, prepared, replayed, "账号映射和回退不能修改共享缓存")
				}
			})
		}
	}
}
