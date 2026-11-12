package service

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestWriteOpenAIImagesStreamEvent_MultilineUsage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	usage := []byte("{\r\n  \"input_tokens\": 1557,\r\n  \"output_tokens\": 20,\r\n  \"details\": {\"note\": \"line\\nbreak\", \"large\": 9007199254740993}\r\n}")
	for _, prefix := range []string{"image_generation", "image_edit"} {
		for _, source := range []string{"constructed", "native"} {
			t.Run(prefix+"/"+source, func(t *testing.T) {
				eventName := prefix + ".completed"
				payload := buildOpenAIImagesStreamCompletedPayload(eventName, openAIResponsesImageResult{Result: "AA=="}, "b64_json", 1710000000, usage)
				if source == "native" {
					payload = []byte(`{"type":"` + eventName + `","b64_json":"AA==","usage":` + string(usage) + `,"model":"gpt-image-2"}`)
				}
				rec := httptest.NewRecorder()
				c, _ := gin.CreateTestContext(rec)
				err := (&OpenAIGatewayService{}).writeOpenAIImagesStreamEvent(c, c.Writer, eventName, payload)
				require.NoError(t, err)
				events := parseOpenAIImageTestSSEEvents(rec.Body.String())
				require.Len(t, events, 1)
				require.Equal(t, eventName, events[0].Name)
				var decoded map[string]json.RawMessage
				require.NoError(t, json.Unmarshal([]byte(events[0].Data), &decoded), "SSE data must contain the entire completed JSON")
				require.Equal(t, 3, strings.Count(rec.Body.String(), "\n"))
				require.Equal(t, "AA==", gjson.Get(events[0].Data, "b64_json").String())
				require.JSONEq(t, string(usage), string(decoded["usage"]))
				require.Equal(t, "9007199254740993", gjson.Get(events[0].Data, "usage.details.large").Raw)
			})
		}
	}
}
