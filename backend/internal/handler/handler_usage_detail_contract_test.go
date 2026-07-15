package handler

import (
	"bytes"
	"go/ast"
	"go/parser"
	"go/token"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func TestHandlerUsageAuditContracts(t *testing.T) {
	t.Run("count_tokens captures request body preview", func(t *testing.T) {
		fn := parseHandlerFunc(t, "gateway_handler.go", "CountTokens")
		require.True(t, funcBodyContainsCall(fn, "service", "SetUsageRequestBody"))
	})

	t.Run("anthropic openai-compatible success usage carries detail snapshot", func(t *testing.T) {
		for _, name := range []string{"gateway_handler_responses.go", "gateway_handler_chat_completions.go"} {
			t.Run(name, func(t *testing.T) {
				requireUsageInputLiteralsHaveKey(t, name, "RecordUsageInput", "DetailSnapshot")
			})
		}
	})

	t.Run("openai embeddings success usage carries detail snapshot", func(t *testing.T) {
		requireUsageInputLiteralsHaveKey(t, "openai_embeddings.go", "OpenAIRecordUsageInput", "DetailSnapshot")
	})

	t.Run("grok media captures request body preview", func(t *testing.T) {
		fn := parseHandlerFunc(t, "grok_media.go", "handleGrokMedia")
		require.True(t, funcBodyContainsCall(fn, "service", "SetUsageRequestBody"))
	})

	t.Run("grok media success usage carries detail snapshot", func(t *testing.T) {
		requireUsageInputLiteralsHaveKey(t, "grok_media.go", "OpenAIRecordUsageInput", "DetailSnapshot")
	})

	t.Run("responses websocket success usage carries detail snapshot", func(t *testing.T) {
		fn := parseHandlerFunc(t, "openai_gateway_handler.go", "ResponsesWebSocket")
		requireFuncUsageInputLiteralsHaveKey(t, "openai_gateway_handler.go", fn, "OpenAIRecordUsageInput", "DetailSnapshot")
	})

	t.Run("responses websocket terminal failover creates failed usage", func(t *testing.T) {
		body := readHandlerSource(t, "openai_gateway_handler.go")
		selectionExhaustedErr := `if lastFailoverErr != nil {
				h.submitFailoverFailedUsageLog(c, apiKey, lastFailedAccount, reqModel, true, lastFailoverErr, lastFailedDuration, lastFailedReasoningEffort, "handler.openai_gateway.responses_ws")
				closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "upstream failover exhausted")`
		selectionExhaustedNil := `if lastFailoverErr != nil {
				h.submitFailoverFailedUsageLog(c, apiKey, lastFailedAccount, reqModel, true, lastFailoverErr, lastFailedDuration, lastFailedReasoningEffort, "handler.openai_gateway.responses_ws")
				closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "upstream failover exhausted")`
		exhausted := `if switchCount >= maxAccountSwitches {
					h.submitFailoverFailedUsageLog(c, apiKey, account, reqModel, true, failoverErr, 0, lastFailedReasoningEffort, "handler.openai_gateway.responses_ws")
					closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "upstream failover exhausted")`
		fastStop := `if h.gatewayService.ShouldStopOpenAIOAuth429Failover(account, failoverErr.StatusCode, switchCount) {
					h.submitFailoverFailedUsageLog(c, apiKey, account, reqModel, true, failoverErr, 0, lastFailedReasoningEffort, "handler.openai_gateway.responses_ws")
					closeOpenAIClientWS(wsConn, coderws.StatusTryAgainLater, "upstream failover exhausted")`
		require.Contains(t, body, selectionExhaustedErr)
		require.Contains(t, body, selectionExhaustedNil)
		require.Contains(t, body, exhausted)
		require.Contains(t, body, fastStop)
	})

	t.Run("anthropic openai-compatible failover exhausted creates failed usage", func(t *testing.T) {
		for _, name := range []string{"gateway_handler_responses.go", "gateway_handler_chat_completions.go"} {
			t.Run(name, func(t *testing.T) {
				body := readHandlerSource(t, name)
				require.Contains(t, body, `h.submitFailedUsageLogFromFailover(c, apiKey, lastFailedAccount, reqModel, reqStream, fs.LastFailoverErr, lastFailedDuration, nil, "handler.gateway.`)
				require.Contains(t, body, `h.submitFailedUsageLogFromFailover(c, apiKey, account, reqModel, reqStream, fs.LastFailoverErr, forwardDuration, nil, "handler.gateway.`)
			})
		}
	})

	t.Run("compat responses handles nil selection after failover", func(t *testing.T) {
		body := readHandlerSource(t, "gateway_handler_responses.go")
		require.Contains(t, body, `if selection == nil || selection.Account == nil {`)
		require.Contains(t, body, `h.submitFailedUsageLogFromFailover(c, apiKey, lastFailedAccount, reqModel, reqStream, fs.LastFailoverErr, lastFailedDuration, nil, "handler.gateway.responses")`)
	})

	t.Run("chat partial failover creates failed usage", func(t *testing.T) {
		openAIBody := readHandlerSource(t, "openai_chat_completions.go")
		require.Contains(t, openAIBody, `if c.Writer.Size() != writerSizeBeforeForward {
						if c.Request.Context().Err() == nil {
							h.submitFailoverFailedUsageLog(c, apiKey, account, reqModel, reqStream, failoverErr, forwardDuration`)

		compatBody := readHandlerSource(t, "gateway_handler_chat_completions.go")
		require.Contains(t, compatBody, `if c.Writer.Size() != writerSizeBeforeForward {
					if c.Request.Context().Err() == nil {
						h.submitFailedUsageLogFromFailover(c, apiKey, account, reqModel, reqStream, failoverErr, forwardDuration`)
	})

	t.Run("prompt too long terminal branches create failed usage", func(t *testing.T) {
		body := readHandlerSource(t, "gateway_handler.go")
		require.Equal(t, 4, strings.Count(body, "submitPromptTooLongFailedUsage()"))
		require.Contains(t, body, "responseHeaders.Set(\"X-Request-Id\", promptTooLongErr.RequestID)")
		require.Contains(t, body, "ResponseHeaders: responseHeaders")
	})
}

func TestJSONInlineBinaryRequestSnapshotsKeepMetadata(t *testing.T) {
	imageBody := []byte(`{"model":"gpt-image-2","prompt":"replace background","size":"1536x1024","n":2,"images":[{"image_url":"data:image/png;base64,c2VjcmV0"}],"mask":{"image_url":"data:image/png;base64,bWFzaw=="}}`)

	t.Run("openai images", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		var snapshot *middleware2.UsageDetailSnapshot
		r := gin.New()
		r.Use(middleware2.UsageDetailCapture())
		r.POST("/v1/images/edits", func(c *gin.Context) {
			parsed := &service.OpenAIImagesRequest{
				Model:          "gpt-image-2",
				Prompt:         "replace background",
				Size:           "1536x1024",
				N:              2,
				InputImageURLs: []string{"data:image/png;base64,c2VjcmV0"},
				MaskImageURL:   "data:image/png;base64,bWFzaw==",
				HasMask:        true,
			}
			service.SetUsageRequestBody(c, service.RequestBodyPreviewString(imageBody))
			snapshot = buildOpenAIImagesDetailSnapshot(c, parsed)
			c.Status(http.StatusNoContent)
		})
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/v1/images/edits", bytes.NewReader(imageBody)))

		metadata := gjson.Get(snapshot.RequestBody, "preview").String()
		require.Equal(t, "gpt-image-2", gjson.Get(metadata, "model").String())
		require.Equal(t, "replace background", gjson.Get(metadata, "prompt").String())
		require.Equal(t, "1536x1024", gjson.Get(metadata, "size").String())
		require.Equal(t, int64(2), gjson.Get(metadata, "n").Int())
		require.True(t, gjson.Get(metadata, "had_source_image").Bool())
		require.True(t, gjson.Get(metadata, "had_mask").Bool())
		require.NotContains(t, snapshot.RequestBody, "data:image")
		require.NotContains(t, snapshot.RequestBody, "c2VjcmV0")
	})

	t.Run("grok media", func(t *testing.T) {
		preview := grokMediaRequestBodyPreview("application/json", imageBody, service.ParseGrokMediaRequest("application/json", imageBody))
		metadata := gjson.Get(preview, "preview").String()
		require.Equal(t, "gpt-image-2", gjson.Get(metadata, "model").String())
		require.Equal(t, "replace background", gjson.Get(metadata, "prompt").String())
		require.Equal(t, "1536x1024", gjson.Get(metadata, "size").String())
		require.Equal(t, int64(2), gjson.Get(metadata, "n").Int())
		require.True(t, gjson.Get(metadata, "had_source_image").Bool())
		require.True(t, gjson.Get(metadata, "had_mask").Bool())
		require.NotContains(t, preview, "data:image")
		require.NotContains(t, preview, "c2VjcmV0")
	})
}

func TestGrokMediaRequestBodyPreviewOmitsMultipartBinary(t *testing.T) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	require.NoError(t, writer.WriteField("model", "grok-imagine-1.0"))
	require.NoError(t, writer.WriteField("prompt", "replace background"))
	imagePart, err := writer.CreateFormFile("image", "source.png")
	require.NoError(t, err)
	_, err = imagePart.Write([]byte("raw-grok-image-bytes"))
	require.NoError(t, err)
	require.NoError(t, writer.Close())

	contentType := writer.FormDataContentType()
	preview := grokMediaRequestBodyPreview(contentType, body.Bytes(), service.ParseGrokMediaRequest(contentType, body.Bytes()))
	metadata := gjson.Get(preview, "preview").String()

	require.NotContains(t, preview, "raw-grok-image-bytes")
	require.Equal(t, "grok-imagine-1.0", gjson.Get(metadata, "model").String())
	require.Empty(t, gjson.Get(metadata, "prompt").String())
	require.True(t, gjson.Get(metadata, "had_source_image").Bool())
}

func TestMultipartMetadataSnapshotsBoundLongPrompts(t *testing.T) {
	prompt := strings.Repeat(`"\\`, 2<<20)
	originalBody := []byte(strings.Repeat("b", 7<<20))

	t.Run("grok", func(t *testing.T) {
		preview := grokMediaRequestBodyPreview("multipart/form-data; boundary=x", originalBody, service.GrokMediaRequestInfo{Prompt: prompt})
		require.LessOrEqual(t, len(preview), 5<<20)
		require.Equal(t, "request_body_preview", gjson.Get(preview, "kind").String())
		require.True(t, gjson.Get(preview, "truncated").Bool())
		require.Equal(t, int64(len(originalBody)), gjson.Get(preview, "size").Int())
		metadata := gjson.Get(preview, "preview").String()
		require.True(t, gjson.Valid(metadata))
		require.Empty(t, gjson.Get(metadata, "prompt").String())
	})

	t.Run("openai images", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		var snapshot *middleware2.UsageDetailSnapshot
		r := gin.New()
		r.Use(middleware2.UsageDetailCapture())
		r.POST("/images", func(c *gin.Context) {
			snapshot = buildOpenAIImagesDetailSnapshot(c, &service.OpenAIImagesRequest{Multipart: true, Prompt: prompt})
			c.Status(http.StatusNoContent)
		})
		r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/images", bytes.NewReader(originalBody)))

		require.NotNil(t, snapshot)
		require.LessOrEqual(t, len(snapshot.RequestBody), 5<<20)
		require.Equal(t, "request_body_preview", gjson.Get(snapshot.RequestBody, "kind").String())
		require.True(t, gjson.Get(snapshot.RequestBody, "truncated").Bool())
		require.Equal(t, int64(len(originalBody)), gjson.Get(snapshot.RequestBody, "size").Int())
		metadata := gjson.Get(snapshot.RequestBody, "preview").String()
		require.True(t, gjson.Valid(metadata))
		require.Empty(t, gjson.Get(metadata, "prompt").String())
	})
}

func requireUsageInputLiteralsHaveKey(t *testing.T, name, literalName, key string) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
	require.NoError(t, err)

	var missing []token.Position
	ast.Inspect(file, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok || !isServiceCompositeLiteral(literal.Type, literalName) {
			return true
		}
		if !compositeLiteralHasKey(literal, key) {
			missing = append(missing, fset.Position(literal.Lbrace))
		}
		return true
	})
	require.Empty(t, missing, "%s literals must carry %s", literalName, key)
}

func requireFuncUsageInputLiteralsHaveKey(t *testing.T, name string, fn *ast.FuncDecl, literalName, key string) {
	t.Helper()
	fset := token.NewFileSet()
	missing := usageInputLiteralsMissingKey(fset, fn.Body, literalName, key)
	require.Empty(t, missing, "%s %s literals must carry %s", name, literalName, key)
}

func usageInputLiteralsMissingKey(fset *token.FileSet, node ast.Node, literalName, key string) []token.Position {
	var missing []token.Position
	ast.Inspect(node, func(node ast.Node) bool {
		literal, ok := node.(*ast.CompositeLit)
		if !ok || !isServiceCompositeLiteral(literal.Type, literalName) {
			return true
		}
		if !compositeLiteralHasKey(literal, key) {
			missing = append(missing, fset.Position(literal.Lbrace))
		}
		return true
	})
	return missing
}

func parseHandlerFunc(t *testing.T, name, funcName string) *ast.FuncDecl {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filepath.Join(".", name), nil, 0)
	require.NoError(t, err)
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && fn.Name.Name == funcName {
			return fn
		}
	}
	t.Fatalf("function %s not found in %s", funcName, name)
	return nil
}

func funcBodyContainsCall(fn *ast.FuncDecl, pkg, name string) bool {
	found := false
	ast.Inspect(fn.Body, func(node ast.Node) bool {
		call, ok := node.(*ast.CallExpr)
		if !ok {
			return true
		}
		selector, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || selector.Sel.Name != name {
			return true
		}
		ident, ok := selector.X.(*ast.Ident)
		found = ok && ident.Name == pkg
		return !found
	})
	return found
}

func isServiceCompositeLiteral(expr ast.Expr, name string) bool {
	selector, ok := expr.(*ast.SelectorExpr)
	if !ok || selector.Sel.Name != name {
		return false
	}
	pkg, ok := selector.X.(*ast.Ident)
	return ok && pkg.Name == "service"
}

func readHandlerSource(t *testing.T, name string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(".", name))
	require.NoError(t, err)
	return strings.ReplaceAll(string(body), "\r\n", "\n")
}
