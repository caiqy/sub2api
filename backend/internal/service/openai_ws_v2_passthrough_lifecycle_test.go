package service

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/Wei-Shaw/sub2api/internal/config"
	coderws "github.com/coder/websocket"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

type stagedPassthroughFrame struct {
	messageType coderws.MessageType
	payload     []byte
	err         error
}

type stagedPassthroughConn struct {
	frames    chan stagedPassthroughFrame
	writes    chan []byte
	writeErr  error
	closed    chan struct{}
	closeOnce sync.Once
}

func newStagedPassthroughConn() *stagedPassthroughConn {
	return &stagedPassthroughConn{
		frames: make(chan stagedPassthroughFrame, 4),
		writes: make(chan []byte, 4),
		closed: make(chan struct{}),
	}
}

func (c *stagedPassthroughConn) Send(payload string) {
	c.frames <- stagedPassthroughFrame{messageType: coderws.MessageText, payload: []byte(payload)}
}

func (c *stagedPassthroughConn) Fail(err error) {
	c.frames <- stagedPassthroughFrame{err: err}
}

func (c *stagedPassthroughConn) WriteJSON(context.Context, any) error { return nil }

func (c *stagedPassthroughConn) ReadMessage(ctx context.Context) ([]byte, error) {
	_, payload, err := c.ReadFrame(ctx)
	return payload, err
}

func (c *stagedPassthroughConn) Ping(context.Context) error { return nil }

func (c *stagedPassthroughConn) ReadFrame(ctx context.Context) (coderws.MessageType, []byte, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return coderws.MessageText, nil, ctx.Err()
	case <-c.closed:
		return coderws.MessageText, nil, errOpenAIWSConnClosed
	case frame := <-c.frames:
		return frame.messageType, append([]byte(nil), frame.payload...), frame.err
	}
}

func (c *stagedPassthroughConn) WriteFrame(ctx context.Context, _ coderws.MessageType, payload []byte) error {
	if c.writeErr != nil {
		return c.writeErr
	}
	if ctx == nil {
		ctx = context.Background()
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return errOpenAIWSConnClosed
	default:
	}
	var parsed any
	if err := json.Unmarshal(payload, &parsed); err != nil {
		return err
	}
	select {
	case c.writes <- append([]byte(nil), payload...):
	case <-ctx.Done():
		return ctx.Err()
	case <-c.closed:
		return errOpenAIWSConnClosed
	}
	return nil
}

func TestPassthroughLifecycle_FirstWriteFailureCallsAfterTurnOnce(t *testing.T) {
	upstream := newStagedPassthroughConn()
	upstream.writeErr = errors.New("first write failed")
	called := 0
	server, serverErr := startPassthroughLifecycleServer(t, context.Background(), newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount(), &OpenAIWSIngressHooks{
		OnOutboundRequest: func(_ int, _ []byte, model string) { require.Equal(t, "gpt-5.1", model) },
		AfterTurn: func(turn int, result *OpenAIForwardResult, err error) {
			called++
			require.Equal(t, 1, turn)
			require.Nil(t, result)
			require.Error(t, err)
		},
	})
	defer server.Close()
	client := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = client.CloseNow() }()
	select {
	case err := <-serverErr:
		require.Error(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("passthrough write failure did not return")
	}
	require.Equal(t, 1, called)
}

func TestPassthroughLifecycle_CleanUpstreamCloseBeforeOutputReturnsFailoverError(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
	}{
		{name: "eof", err: io.EOF},
		{name: "normal_close", err: coderws.CloseError{Code: coderws.StatusNormalClosure}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := newStagedPassthroughConn()
			server, serverErr := startPassthroughLifecycleServer(
				t,
				context.Background(),
				newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream),
				passthroughLifecycleAccount(),
			)
			defer server.Close()
			client := dialPassthroughLifecycleClient(t, server)
			defer func() { _ = client.CloseNow() }()
			readCtx, cancelRead := context.WithCancel(context.Background())
			defer cancelRead()
			go func() { _, _, _ = client.Read(readCtx) }()

			require.Equal(t, "response.create", gjson.GetBytes(requirePassthroughUpstreamWrite(t, upstream, time.Second), "type").String())
			upstream.Fail(tc.err)

			select {
			case err := <-serverErr:
				var failoverErr *UpstreamFailoverError
				require.ErrorAs(t, err, &failoverErr)
				require.True(t, failoverErr.ShouldRetryNextAccount())
			case <-time.After(6 * time.Second):
				t.Fatal("clean upstream close did not return from passthrough")
			}
		})
	}
}

func TestPassthroughLifecycle_ClientDisconnectDrainsSecondTurnCompletionOnce(t *testing.T) {
	upstream := newStagedPassthroughConn()
	type completion struct {
		turn   int
		result *OpenAIForwardResult
		err    error
	}
	completed := make(chan completion, 3)
	server, serverErr := startPassthroughLifecycleServer(t, context.Background(), newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount(), &OpenAIWSIngressHooks{
		AfterTurn: func(turn int, result *OpenAIForwardResult, err error) {
			completed <- completion{turn: turn, result: result, err: err}
		},
	})
	defer server.Close()

	firstPayload := `{"type":"response.create","model":"gpt-5.4-max","reasoning":{"effort":"high"}}`
	client := dialPassthroughLifecycleClientWithPayload(t, server, firstPayload)
	defer func() { _ = client.CloseNow() }()
	firstWrite := requirePassthroughUpstreamWrite(t, upstream, time.Second)
	require.Equal(t, "gpt-5.4-max", gjson.GetBytes(firstWrite, "model").String())
	require.Equal(t, "high", gjson.GetBytes(firstWrite, "reasoning.effort").String())
	upstream.Send(`{"type":"response.created","response":{"id":"resp_first","model":"gpt-5.4-max"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_first","model":"gpt-5.4-max","usage":{"input_tokens":1,"output_tokens":1}}}`)
	_, err := readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	var first completion
	select {
	case first = <-completed:
	case <-time.After(time.Second):
		t.Fatal("first passthrough turn did not complete")
	}
	require.NoError(t, first.err)
	require.Equal(t, 1, first.turn)
	require.Equal(t, "resp_first", first.result.RequestID)
	require.NotNil(t, first.result.RequestedReasoningEffort)
	require.Equal(t, "high", *first.result.RequestedReasoningEffort)
	require.NotNil(t, first.result.ReasoningEffort)
	require.Equal(t, "high", *first.result.ReasoningEffort)

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), time.Second)
	err = client.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create"}`))
	cancelWrite()
	require.NoError(t, err)
	secondWrite := requirePassthroughUpstreamWrite(t, upstream, time.Second)
	require.Equal(t, "gpt-5.4-max", gjson.GetBytes(secondWrite, "model").String())
	require.False(t, gjson.GetBytes(secondWrite, "reasoning.effort").Exists())
	upstream.Send(`{"type":"response.created","response":{"id":"resp_second","model":"gpt-5.4-max"}}`)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	require.NoError(t, client.CloseNow())
	time.Sleep(50 * time.Millisecond)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_second","model":"gpt-5.4-max","usage":{"input_tokens":2,"output_tokens":3}}}`)

	var second completion
	select {
	case second = <-completed:
	case <-time.After(time.Second):
		t.Fatal("drained second passthrough turn did not complete")
	}
	require.NoError(t, second.err)
	require.Equal(t, 2, second.turn)
	require.Equal(t, "resp_second", second.result.RequestID)
	require.Equal(t, 2, second.result.Usage.InputTokens)
	require.Equal(t, 3, second.result.Usage.OutputTokens)
	require.Nil(t, second.result.RequestedReasoningEffort)
	require.NotNil(t, second.result.ReasoningEffort)
	require.Equal(t, "xhigh", *second.result.ReasoningEffort)
	select {
	case extra := <-completed:
		t.Fatalf("unexpected duplicate completion: turn=%d err=%v", extra.turn, extra.err)
	case err := <-serverErr:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("passthrough drain did not return")
	}
	require.Empty(t, completed)
}

func (c *stagedPassthroughConn) Close() error {
	c.closeOnce.Do(func() { close(c.closed) })
	return nil
}

type stagedPassthroughDialer struct {
	conn        openAIWSClientConn
	mu          sync.Mutex
	lastHeaders http.Header
}

func (d *stagedPassthroughDialer) Dial(_ context.Context, _ string, headers http.Header, _ string) (openAIWSClientConn, int, http.Header, error) {
	d.mu.Lock()
	d.lastHeaders = headers.Clone()
	d.mu.Unlock()
	return d.conn, http.StatusSwitchingProtocols, http.Header{}, nil
}

func (d *stagedPassthroughDialer) Headers() http.Header {
	d.mu.Lock()
	defer d.mu.Unlock()
	return d.lastHeaders.Clone()
}

func newPassthroughLifecycleService(cfg *config.Config, upstream *stagedPassthroughConn) *OpenAIGatewayService {
	dialer := &stagedPassthroughDialer{conn: upstream}
	return &OpenAIGatewayService{
		cfg:                       cfg,
		httpUpstream:              &httpUpstreamRecorder{},
		cache:                     &stubGatewayCache{},
		openaiWSResolver:          NewOpenAIWSProtocolResolver(cfg),
		toolCorrector:             NewCodexToolCorrector(),
		openaiWSPassthroughDialer: dialer,
	}
}

func passthroughLifecycleConfig() *config.Config {
	cfg := &config.Config{}
	cfg.Security.URLAllowlist.Enabled = false
	cfg.Security.URLAllowlist.AllowInsecureHTTP = true
	cfg.Gateway.OpenAIFirstOutputTimeoutSeconds = 1
	cfg.Gateway.OpenAIWS.Enabled = true
	cfg.Gateway.OpenAIWS.APIKeyEnabled = true
	cfg.Gateway.OpenAIWS.ResponsesWebsocketsV2 = true
	cfg.Gateway.OpenAIWS.ModeRouterV2Enabled = true
	cfg.Gateway.OpenAIWS.IngressModeDefault = OpenAIWSIngressModeCtxPool
	cfg.Gateway.OpenAIWS.IngressInterTurnIdleTimeoutSeconds = 1
	cfg.Gateway.OpenAIWS.DialTimeoutSeconds = 3
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 1
	cfg.Gateway.OpenAIWS.WriteTimeoutSeconds = 3
	return cfg
}

func passthroughLifecycleAccount() *Account {
	return &Account{
		ID:          901,
		Name:        "passthrough-lifecycle",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"api_key": "sk-test"},
		Extra: map[string]any{
			"openai_apikey_responses_websockets_v2_mode": OpenAIWSIngressModePassthrough,
		},
	}
}

func startPassthroughLifecycleServer(
	t *testing.T,
	controlCtx context.Context,
	svc *OpenAIGatewayService,
	account *Account,
	hooks ...*OpenAIWSIngressHooks,
) (*httptest.Server, <-chan error) {
	return startPassthroughLifecycleServerWithHeaders(t, controlCtx, svc, account, nil, hooks...)
}

func startPassthroughLifecycleServerWithHeaders(
	t *testing.T,
	controlCtx context.Context,
	svc *OpenAIGatewayService,
	account *Account,
	requestHeaders http.Header,
	hooks ...*OpenAIWSIngressHooks,
) (*httptest.Server, <-chan error) {
	var hooksFactory func(*gin.Context) *OpenAIWSIngressHooks
	if len(hooks) > 0 {
		hook := hooks[0]
		hooksFactory = func(*gin.Context) *OpenAIWSIngressHooks { return hook }
	}
	return startPassthroughLifecycleServerWithRequestHeaders(t, controlCtx, svc, account, requestHeaders, hooksFactory)
}

func startPassthroughLifecycleServerWithHooks(
	t *testing.T,
	controlCtx context.Context,
	svc *OpenAIGatewayService,
	account *Account,
	hooksFactory func(*gin.Context) *OpenAIWSIngressHooks,
) (*httptest.Server, <-chan error) {
	return startPassthroughLifecycleServerWithRequestHeaders(t, controlCtx, svc, account, nil, hooksFactory)
}

func startPassthroughLifecycleServerWithRequestHeaders(
	t *testing.T,
	controlCtx context.Context,
	svc *OpenAIGatewayService,
	account *Account,
	requestHeaders http.Header,
	hooksFactory func(*gin.Context) *OpenAIWSIngressHooks,
) (*httptest.Server, <-chan error) {
	t.Helper()
	serverErr := make(chan error, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := coderws.Accept(w, r, &coderws.AcceptOptions{CompressionMode: coderws.CompressionContextTakeover})
		if err != nil {
			serverErr <- err
			return
		}
		defer func() { _ = conn.CloseNow() }()

		msgType, firstMessage, err := ReadOpenAIWSClientMessage(
			controlCtx,
			conn,
			3*time.Second,
			coderws.StatusPolicyViolation,
			"missing first response.create message",
		)
		if err != nil {
			serverErr <- err
			return
		}
		if msgType != coderws.MessageText {
			serverErr <- errors.New("first message was not text")
			return
		}

		recorder := httptest.NewRecorder()
		ginCtx, _ := gin.CreateTestContext(recorder)
		req := r.Clone(controlCtx)
		req.Header = req.Header.Clone()
		for key, values := range requestHeaders {
			req.Header.Del(key)
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		ginCtx.Request = req
		var hooks *OpenAIWSIngressHooks
		if hooksFactory != nil {
			hooks = hooksFactory(ginCtx)
		}
		serverErr <- svc.ProxyResponsesWebSocketFromClient(controlCtx, ginCtx, conn, account, "sk-test", firstMessage, hooks)
	}))
	return server, serverErr
}

func TestPassthroughLifecycle_CyberTerminalEventsMarkBeforeAfterTurn(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name        string
		events      []string
		wantBody    string
		wantMessage string
		wantInput   int
		wantOutput  int
	}{
		{
			name: "error",
			events: []string{
				`{"type":"error","error":{"code":"cyber_policy","message":"blocked by error event"},"usage":{"input_tokens":5,"output_tokens":1}}`,
				`{"type":"response.failed","response":{"id":"resp_error","error":{"code":"cyber_policy","message":"blocked by paired failed event"},"usage":{"input_tokens":9,"output_tokens":2}}}`,
			},
			wantBody:    `"type":"error"`,
			wantMessage: "blocked by error event",
			wantInput:   5,
			wantOutput:  1,
		},
		{
			name: "response_failed",
			events: []string{
				`{"type":"response.failed","response":{"id":"resp_failed","error":{"code":"cyber_policy","message":"blocked by failed event"},"usage":{"input_tokens":9,"output_tokens":2}}}`,
			},
			wantBody:    `"type":"response.failed"`,
			wantMessage: "blocked by failed event",
			wantInput:   9,
			wantOutput:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controlCtx, cancelControl := context.WithCancelCause(context.Background())
			defer cancelControl(context.Canceled)
			upstream := newStagedPassthroughConn()
			for _, event := range tt.events {
				upstream.Send(event)
			}

			markSeen := make(chan CyberPolicyMark, 1)
			afterTurnCalls := atomic.Int32{}
			server, serverErr := startPassthroughLifecycleServerWithHooks(
				t,
				controlCtx,
				newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream),
				passthroughLifecycleAccount(),
				func(c *gin.Context) *OpenAIWSIngressHooks {
					return &OpenAIWSIngressHooks{AfterTurn: func(_ int, _ *OpenAIForwardResult, _ error) {
						afterTurnCalls.Add(1)
						if mark := GetOpsCyberPolicy(c); mark != nil {
							select {
							case markSeen <- *mark:
							default:
							}
						}
					}}
				},
			)
			defer server.Close()
			clientConn := dialPassthroughLifecycleClient(t, server)
			defer func() { _ = clientConn.CloseNow() }()

			for range tt.events {
				_, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
				require.NoError(t, err)
			}

			select {
			case mark := <-markSeen:
				require.Equal(t, "cyber_policy", mark.Code)
				require.Equal(t, tt.wantMessage, mark.Message)
				require.Contains(t, mark.Body, tt.wantBody)
				require.Equal(t, http.StatusOK, mark.UpstreamStatus)
				require.Equal(t, tt.wantInput, mark.UpstreamInTok)
				require.Equal(t, tt.wantOutput, mark.UpstreamOutTok)
			case <-time.After(3 * time.Second):
				t.Fatal("cyber mark was not visible to AfterTurn")
			}
			require.NoError(t, clientConn.Close(coderws.StatusNormalClosure, "done"))
			select {
			case <-serverErr:
			case <-time.After(3 * time.Second):
				t.Fatal("cyber passthrough test did not exit")
			}
			require.Equal(t, int32(1), afterTurnCalls.Load(), "error/response.failed pair must complete and record once")
		})
	}
}

func TestPassthroughLifecycle_NonCyberFailureKeepsAccountSideEffects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.failed","response":{"id":"resp_non_cyber","error":{"type":"authentication_error","code":"invalid_api_key","status_code":401,"message":"credential rejected"},"usage":{"input_tokens":3,"output_tokens":1}}}`)
	repo := &openAIStream403AccountRepo{}
	svc := newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream)
	svc.rateLimitService = NewRateLimitService(repo, nil, svc.cfg, nil, nil)
	account := passthroughLifecycleAccount()

	markSeen := make(chan *CyberPolicyMark, 1)
	server, serverErr := startPassthroughLifecycleServerWithHooks(
		t,
		controlCtx,
		svc,
		account,
		func(c *gin.Context) *OpenAIWSIngressHooks {
			return &OpenAIWSIngressHooks{AfterTurn: func(_ int, _ *OpenAIForwardResult, _ error) {
				markSeen <- GetOpsCyberPolicy(c)
			}}
		},
	)
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	event, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.failed", gjson.GetBytes(event, "type").String())
	select {
	case mark := <-markSeen:
		require.Nil(t, mark)
	case <-time.After(3 * time.Second):
		t.Fatal("non-cyber terminal event did not complete its turn")
	}
	require.Equal(t, 1, repo.setErrorCalls, "non-cyber credential failure must retain account failure side effects")
	require.True(t, svc.isOpenAIAccountRuntimeBlocked(account))
	require.NoError(t, clientConn.Close(coderws.StatusNormalClosure, "done"))
	select {
	case <-serverErr:
	case <-time.After(3 * time.Second):
		t.Fatal("non-cyber passthrough test did not exit")
	}
}

func TestPassthroughLifecycle_CyberSkipsFailureAccountSideEffects(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.failed","response":{"id":"resp_cyber_auth","error":{"type":"authentication_error","code":"cyber_policy","status_code":401,"message":"request blocked"}}}`)
	repo := &openAIStream403AccountRepo{}
	svc := newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream)
	svc.rateLimitService = NewRateLimitService(repo, nil, svc.cfg, nil, nil)
	account := passthroughLifecycleAccount()

	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, svc, account)
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	event, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.failed", gjson.GetBytes(event, "type").String())
	require.Zero(t, repo.setErrorCalls, "cyber_policy is request-scoped and must not cool down the account")
	require.False(t, svc.isOpenAIAccountRuntimeBlocked(account))

	require.NoError(t, clientConn.Close(coderws.StatusNormalClosure, "done"))
	select {
	case <-serverErr:
	case <-time.After(3 * time.Second):
		t.Fatal("cyber side-effect test did not exit")
	}
}

func TestPassthroughLifecycle_CloseReasonTruncationPreservesUTF8(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	originalReason := strings.Repeat("a", 119) + "界"
	upstream.Fail(NewOpenAIWSClientCloseError(coderws.StatusPolicyViolation, originalReason, errors.New("policy rejected")))

	server, serverErr := startPassthroughLifecycleServer(
		t,
		controlCtx,
		newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream),
		passthroughLifecycleAccount(),
	)
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	_, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	var closeErr coderws.CloseError
	require.ErrorAs(t, err, &closeErr)
	require.Equal(t, coderws.StatusPolicyViolation, closeErr.Code)
	require.True(t, utf8.ValidString(closeErr.Reason))
	require.LessOrEqual(t, len(closeErr.Reason), 120)
	require.Equal(t, strings.Repeat("a", 119), closeErr.Reason)

	select {
	case <-serverErr:
	case <-time.After(3 * time.Second):
		t.Fatal("passthrough close reason test did not exit")
	}
}

func dialPassthroughLifecycleClient(t *testing.T, server *httptest.Server) *coderws.Conn {
	return dialPassthroughLifecycleClientWithModel(t, server, "gpt-5.1")
}

func dialPassthroughLifecycleClientWithModel(t *testing.T, server *httptest.Server, model string) *coderws.Conn {
	return dialPassthroughLifecycleClientWithHeaders(t, server, model, nil)
}

func dialPassthroughLifecycleClientWithHeaders(t *testing.T, server *httptest.Server, model string, headers http.Header) *coderws.Conn {
	return dialPassthroughLifecycleClientWithFirstFrame(t, server, []byte(`{"type":"response.create","model":"`+model+`","stream":false}`), headers)
}

func dialPassthroughLifecycleClientWithFirstFrame(t *testing.T, server *httptest.Server, firstFrame []byte, headers http.Header) *coderws.Conn {
	t.Helper()
	dialCtx, cancelDial := context.WithTimeout(context.Background(), 3*time.Second)
	clientConn, _, err := coderws.Dial(dialCtx, "ws"+strings.TrimPrefix(server.URL, "http"), &coderws.DialOptions{HTTPHeader: headers})
	cancelDial()
	require.NoError(t, err)
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, firstFrame)
	cancelWrite()
	require.NoError(t, err)
	return clientConn
}

func dialPassthroughLifecycleClientWithPayload(t *testing.T, server *httptest.Server, payload string) *coderws.Conn {
	t.Helper()
	return dialPassthroughLifecycleClientWithFirstFrame(t, server, []byte(payload), nil)
}

func TestPassthroughLifecycle_OAuthFingerprintRewritesInitialAndLaterResponseCreateFrames(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	upstream := newStagedPassthroughConn()
	svc := newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream)
	svc.cfg.Gateway.OpenAIWS.OAuthEnabled = true
	account := &Account{
		ID:          902,
		Name:        "oauth-passthrough-fingerprint",
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "chatgpt-acc"},
		Extra: map[string]any{
			"codex_fingerprint_mode":                    "session",
			codexFingerprintSeedExtraKey:                testCodexFingerprintSeed,
			"openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModePassthrough,
		},
	}
	server, serverErr := startPassthroughLifecycleServer(t, ctx, svc, account)
	defer server.Close()
	client := dialPassthroughLifecycleClientWithFirstFrame(t, server, []byte(`{"type":"response.create","model":"gpt-5.1","stream":false,"client_metadata":{"session_id":"initial-session","x-codex-installation-id":"initial-installation","thread_id":"initial-thread","x-codex-turn-metadata":"{\"installation_id\":\"initial-installation\",\"session_id\":\"initial-session\",\"thread_id\":\"initial-thread\",\"turn_id\":\"initial-turn\"}"}}`), http.Header{
		"session-id":              []string{"client-session"},
		"x-codex-installation-id": []string{"client-installation"},
		"x-codex-turn-metadata":   []string{`{"installation_id":"client-installation","session_id":"client-session","thread_id":"client-thread","turn_id":"client-turn"}`},
		"authorization":           []string{"Bearer inbound-must-not-forward"},
	})
	defer func() { _ = client.CloseNow() }()

	first := requirePassthroughUpstreamWrite(t, upstream, time.Second)
	dialer, ok := svc.getOpenAIWSPassthroughDialer().(*stagedPassthroughDialer)
	require.True(t, ok, "test service must use staged passthrough dialer")
	headers := dialer.Headers()
	require.Equal(t, headers.Get("session_id"), gjson.GetBytes(first, "client_metadata.session_id").String())
	require.Equal(t, headers.Get("x-codex-installation-id"), gjson.GetBytes(first, "client_metadata.x-codex-installation-id").String())
	require.Equal(t, headers.Get("thread-id"), gjson.GetBytes(first, "client_metadata.thread_id").String())
	var headerMetadata map[string]any
	require.NoError(t, json.Unmarshal([]byte(headers.Get("x-codex-turn-metadata")), &headerMetadata))
	var firstBodyMetadata map[string]any
	require.NoError(t, json.Unmarshal([]byte(gjson.GetBytes(first, "client_metadata.x-codex-turn-metadata").String()), &firstBodyMetadata))
	require.Equal(t, headerMetadata["turn_id"], firstBodyMetadata["turn_id"], "handshake and initial frame must share one attempt ID set")

	upstream.Send(`{"type":"response.created","response":{"id":"resp_first","model":"gpt-5.1"}}`)
	_, err := readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), time.Second)
	err = client.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.cancel","response_id":"resp_first","client_metadata":{"session_id":"cancel-session"}}`))
	cancelWrite()
	require.NoError(t, err)
	cancelFrame := requirePassthroughUpstreamWrite(t, upstream, time.Second)
	require.Equal(t, "cancel-session", gjson.GetBytes(cancelFrame, "client_metadata.session_id").String(), "non-response.create frames must remain unchanged")

	upstream.Send(`{"type":"response.completed","response":{"id":"resp_first","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	writeCtx, cancelWrite = context.WithTimeout(context.Background(), time.Second)
	err = client.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1","client_metadata":{"session_id":"later-session","x-codex-installation-id":"later-installation","thread_id":"later-thread"}}`))
	cancelWrite()
	require.NoError(t, err)
	later := requirePassthroughUpstreamWrite(t, upstream, time.Second)
	require.Equal(t, headers.Get("session_id"), gjson.GetBytes(later, "client_metadata.session_id").String())
	require.Equal(t, headers.Get("x-codex-installation-id"), gjson.GetBytes(later, "client_metadata.x-codex-installation-id").String())
	require.Equal(t, headers.Get("thread-id"), gjson.GetBytes(later, "client_metadata.thread_id").String())

	upstream.Send(`{"type":"response.created","response":{"id":"resp_later","model":"gpt-5.1"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_later","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
	<-serverErr
}

func TestPassthroughLifecycle_OAuthOffPreservesBothClientSessionHeaderForms(t *testing.T) {
	for _, tc := range []struct {
		name   string
		header http.Header
		key    string
		value  string
	}{
		{name: "hyphen", header: http.Header{"Session-Id": []string{"client-hyphen"}}, key: "session-id", value: "client-hyphen"},
		{name: "underscore", header: http.Header{"Session_id": []string{"client-underscore"}}, key: "session_id", value: "client-underscore"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			upstream := newStagedPassthroughConn()
			svc := newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream)
			svc.cfg.Gateway.OpenAIWS.OAuthEnabled = true
			account := &Account{
				ID: 903, Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Schedulable: true, Concurrency: 1,
				Credentials: map[string]any{"access_token": "oauth-token", "chatgpt_account_id": "chatgpt-acc"},
				Extra: map[string]any{
					"codex_fingerprint_mode":                    "off",
					"openai_oauth_responses_websockets_v2_mode": OpenAIWSIngressModePassthrough,
				},
			}
			tc.header.Set("authorization", "Bearer inbound-must-not-forward")
			server, serverErr := startPassthroughLifecycleServerWithHeaders(t, context.Background(), svc, account, tc.header)
			defer server.Close()
			client := dialPassthroughLifecycleClientWithHeaders(t, server, "gpt-5.1", nil)
			defer func() { _ = client.CloseNow() }()

			first := requirePassthroughUpstreamWrite(t, upstream, time.Second)
			dialer, ok := svc.getOpenAIWSPassthroughDialer().(*stagedPassthroughDialer)
			require.True(t, ok, "test service must use staged passthrough dialer")
			headers := dialer.Headers()
			require.Equal(t, tc.value, headers.Get(tc.key))
			require.Equal(t, "Bearer sk-test", headers.Get("authorization"))
			require.Equal(t, "", gjson.GetBytes(first, "client_metadata.session_id").String())
			upstream.Send(`{"type":"response.created","response":{"id":"resp_off","model":"gpt-5.1"}}`)
			upstream.Send(`{"type":"response.completed","response":{"id":"resp_off","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`)
			_, err := readPassthroughLifecycleFrame(t, client, time.Second)
			require.NoError(t, err)
			_, err = readPassthroughLifecycleFrame(t, client, time.Second)
			require.NoError(t, err)
			_ = client.Close(coderws.StatusNormalClosure, "done")
			select {
			case <-serverErr:
			case <-time.After(2 * time.Second):
				t.Fatal("passthrough off relay did not exit")
			}
		})
	}
}

func readPassthroughLifecycleFrame(t *testing.T, clientConn *coderws.Conn, timeout time.Duration) ([]byte, error) {
	t.Helper()
	readCtx, cancelRead := context.WithTimeout(context.Background(), timeout)
	_, payload, err := clientConn.Read(readCtx)
	cancelRead()
	return payload, err
}

func requirePassthroughUpstreamWrite(t *testing.T, upstream *stagedPassthroughConn, timeout time.Duration) []byte {
	t.Helper()
	select {
	case payload := <-upstream.writes:
		return payload
	case <-time.After(timeout):
		t.Fatal("passthrough request was not forwarded upstream")
		return nil
	}
}

func TestPassthroughLifecycle_ResponsesLiteFirstFramePinsParallelToolCalls(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_lite","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`)
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClientWithPayload(t, server, `{
		"type":"response.create","model":"gpt-5.1","stream":false,
		"parallel_tool_calls":true,
		"client_metadata":{"ws_request_header_x_openai_internal_codex_responses_lite":"true"}
	}`)
	defer func() { _ = clientConn.CloseNow() }()

	upstreamBody := requirePassthroughUpstreamWrite(t, upstream, 3*time.Second)
	require.Equal(t, gjson.False, gjson.GetBytes(upstreamBody, "parallel_tool_calls").Type, string(upstreamBody))

	event, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.completed", gjson.GetBytes(event, "type").String())
	require.NoError(t, clientConn.Close(coderws.StatusNormalClosure, "done"))
	select {
	case <-serverErr:
	case <-time.After(3 * time.Second):
		t.Fatal("Lite 首帧测试等待 passthrough 退出超时")
	}
}

func TestOpenAIWSPassthroughTurnLifecycle_SerializesTerminalCommitAndNextTurn(t *testing.T) {
	clientFrameConn := &openAIWSClientFrameConn{interTurnStarted: make(chan struct{}, 1)}
	clientFrameConn.markTurnCompleted()
	lifecycle := newOpenAIWSPassthroughTurnLifecycle(true)
	lifecycle.beginTerminalWrite()

	admitted := make(chan bool, 1)
	go func() {
		admitted <- lifecycle.beginResponseCreate(clientFrameConn.markTurnStarted)
	}()
	select {
	case <-admitted:
		t.Fatal("next response.create was admitted before terminal commit completed")
	case <-time.After(50 * time.Millisecond):
	}

	lifecycle.finishTerminalWrite(true, clientFrameConn.markTurnCompleted)
	select {
	case ok := <-admitted:
		require.True(t, ok)
	case <-time.After(time.Second):
		t.Fatal("next response.create remained blocked after terminal commit")
	}
	require.False(t, clientFrameConn.waitingForNextTurn.Load(), "accepted next turn must win over terminal idle state")

	lifecycle = newOpenAIWSPassthroughTurnLifecycle(true)
	lifecycle.beginTerminalWrite()
	admitted = make(chan bool, 1)
	go func() {
		admitted <- lifecycle.beginResponseCreate(nil)
	}()
	lifecycle.finishTerminalWrite(false, func() {
		t.Error("failed terminal write must not commit idle state")
	})
	require.False(t, <-admitted, "failed terminal write must keep the current turn in flight")
}

func TestPassthroughLifecycle_AppliesAccountMappingAfterLaterRequestRewrite(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_first","model":"gpt-5.1"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_first","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`)
	svc := newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream)
	account := passthroughLifecycleAccount()
	account.Credentials["model_mapping"] = map[string]any{"routed-model": "account-model"}
	hooks := &OpenAIWSIngressHooks{
		RewriteRequest: func(turn int, payload []byte, originalModel string) (OpenAIWSRequestRewrite, error) {
			if turn < 2 || originalModel != "public-model" {
				return OpenAIWSRequestRewrite{Payload: payload, OriginalModel: originalModel}, nil
			}
			return OpenAIWSRequestRewrite{Payload: ReplaceModelInBody(payload, "routed-model"), OriginalModel: "routed-model"}, nil
		},
	}
	server, serverErr := startPassthroughLifecycleServer(t, ctx, svc, account, hooks)
	defer server.Close()
	client := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = client.CloseNow() }()

	require.Equal(t, "gpt-5.1", gjson.GetBytes(requirePassthroughUpstreamWrite(t, upstream, time.Second), "model").String())
	_, err := readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), time.Second)
	err = client.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"public-model","previous_response_id":"resp_first"}`))
	cancelWrite()
	require.NoError(t, err)
	require.Equal(t, "account-model", gjson.GetBytes(requirePassthroughUpstreamWrite(t, upstream, time.Second), "model").String())

	upstream.Send(`{"type":"response.created","response":{"id":"resp_second","model":"account-model"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_second","model":"account-model","usage":{"input_tokens":1,"output_tokens":1}}}`)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
	<-serverErr
}

func TestPassthroughLifecycle_SessionUpdateModelLessTurnUsesMappedFastPolicyFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	upstream := newStagedPassthroughConn()
	svc := newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream)
	settings := &OpenAIFastPolicySettings{Rules: []OpenAIFastPolicyRule{{
		ServiceTier:    OpenAIFastTierPriority,
		Action:         BetaPolicyActionFilter,
		Scope:          BetaPolicyScopeAll,
		ModelWhitelist: []string{"gpt-5.5-account"},
		FallbackAction: BetaPolicyActionPass,
	}}}
	rawSettings, err := json.Marshal(settings)
	require.NoError(t, err)
	svc.settingService = NewSettingService(&openAIFastPolicyRepoStub{values: map[string]string{
		SettingKeyOpenAIFastPolicySettings: string(rawSettings),
	}}, svc.cfg)
	account := passthroughLifecycleAccount()
	account.Credentials["model_mapping"] = map[string]any{
		"gpt-client":  "gpt-client",
		"gpt-channel": "gpt-account",
		"gpt-5.5":     "gpt-5.5-account",
	}
	outboundModels := make(chan string, 2)
	hooks := &OpenAIWSIngressHooks{
		MapRequestModel: func(_ int, model string) (string, error) {
			if model == "gpt-client" {
				return "gpt-channel", nil
			}
			return model, nil
		},
		OnOutboundRequest: func(_ int, _ []byte, model string) {
			outboundModels <- model
		},
	}
	server, serverErr := startPassthroughLifecycleServer(t, ctx, svc, account, hooks)
	defer server.Close()
	client := dialPassthroughLifecycleClientWithModel(t, server, "gpt-client")
	defer func() { _ = client.CloseNow() }()

	first := requirePassthroughUpstreamWrite(t, upstream, time.Second)
	require.Equal(t, "gpt-channel", gjson.GetBytes(first, "model").String())
	upstream.Send(`{"type":"response.created","response":{"id":"resp_first","model":"gpt-channel"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_first","model":"gpt-channel","usage":{"input_tokens":1,"output_tokens":1}}}`)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)

	writeCtx, cancelWrite := context.WithTimeout(context.Background(), time.Second)
	err = client.Write(writeCtx, coderws.MessageText, []byte(`{"type":"session.update","session":{"model":"gpt-5.5"}}`))
	cancelWrite()
	require.NoError(t, err)
	sessionUpdate := requirePassthroughUpstreamWrite(t, upstream, time.Second)
	require.Equal(t, "gpt-5.5", gjson.GetBytes(sessionUpdate, "session.model").String())

	writeCtx, cancelWrite = context.WithTimeout(context.Background(), time.Second)
	err = client.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","service_tier":"priority"}`))
	cancelWrite()
	require.NoError(t, err)
	followup := requirePassthroughUpstreamWrite(t, upstream, time.Second)
	require.False(t, gjson.GetBytes(followup, "model").Exists())
	require.False(t, gjson.GetBytes(followup, "service_tier").Exists())
	require.Equal(t, "gpt-channel", <-outboundModels)
	require.Equal(t, "gpt-5.5", <-outboundModels)

	upstream.Send(`{"type":"response.created","response":{"id":"resp_second","model":"gpt-5.5"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_second","model":"gpt-5.5","usage":{"input_tokens":1,"output_tokens":1}}}`)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	_, err = readPassthroughLifecycleFrame(t, client, time.Second)
	require.NoError(t, err)
	require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
	<-serverErr
}

func TestPassthroughLifecycle_LeaseLossSendsRetryClose(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_lease","model":"gpt-5.1"}}`)
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	event, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.created", gjson.GetBytes(event, "type").String())
	cancelControl(ErrOpenAIWSIngressLeaseLost)

	_, err = readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	var closeErr coderws.CloseError
	require.ErrorAs(t, err, &closeErr)
	require.Equal(t, coderws.StatusTryAgainLater, closeErr.Code)
	require.Equal(t, "websocket ingress capacity lease lost; please reconnect", closeErr.Reason)
	select {
	case <-serverErr:
	case <-time.After(3 * time.Second):
		t.Fatal("passthrough lease-loss reader did not exit")
	}
}

func TestPassthroughLifecycle_CompletedTurnStartsInterTurnIdle(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_idle","model":"gpt-5.1"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_idle","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`)
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	event, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.created", gjson.GetBytes(event, "type").String())
	event, err = readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.completed", gjson.GetBytes(event, "type").String())
	_, err = readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	var closeErr coderws.CloseError
	require.ErrorAs(t, err, &closeErr)
	require.Equal(t, coderws.StatusNormalClosure, closeErr.Code)
	require.Equal(t, "websocket idle timeout", closeErr.Reason)
	select {
	case <-serverErr:
	case <-time.After(3 * time.Second):
		t.Fatal("passthrough idle reader did not exit")
	}
}

func TestPassthroughLifecycle_ActiveTurnInactivityUsesReadTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_active","model":"gpt-5.1"}}`)
	upstream.Send(`{"type":"response.output_text.delta","response_id":"resp_active","delta":"hello"}`)
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	created, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.created", gjson.GetBytes(created, "type").String())
	delta, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.output_text.delta", gjson.GetBytes(delta, "type").String())
	_, err = readPassthroughLifecycleFrame(t, clientConn, 2500*time.Millisecond)
	var websocketCloseErr coderws.CloseError
	require.ErrorAs(t, err, &websocketCloseErr)
	require.Equal(t, coderws.StatusGoingAway, websocketCloseErr.Code)
	require.Equal(t, "upstream websocket read timeout; please reconnect", websocketCloseErr.Reason)
	select {
	case err := <-serverErr:
		var closeErr *OpenAIWSClientCloseError
		require.ErrorAs(t, err, &closeErr)
		require.Equal(t, coderws.StatusGoingAway, closeErr.StatusCode())
		require.Equal(t, "upstream websocket read timeout; please reconnect", closeErr.Reason())
	case <-time.After(2500 * time.Millisecond):
		t.Fatal("passthrough active turn remained unbounded after upstream activity stopped")
	}
}

func TestPassthroughLifecycle_PreambleAllowsPromptClientCancel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	cfg := passthroughLifecycleConfig()
	cfg.Gateway.OpenAIFirstOutputTimeoutSeconds = 3
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_cancel","model":"gpt-5.1"}}`)
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(cfg, upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()
	require.Equal(t, "response.create", gjson.GetBytes(requirePassthroughUpstreamWrite(t, upstream, time.Second), "type").String())

	created, err := readPassthroughLifecycleFrame(t, clientConn, time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.created", gjson.GetBytes(created, "type").String())
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.cancel","response_id":"resp_cancel"}`))
	cancelWrite()
	require.NoError(t, err)
	cancelFrame := requirePassthroughUpstreamWrite(t, upstream, 500*time.Millisecond)
	require.Equal(t, "response.cancel", gjson.GetBytes(cancelFrame, "type").String())

	require.NoError(t, clientConn.Close(coderws.StatusNormalClosure, "done"))
	select {
	case <-serverErr:
	case <-time.After(3 * time.Second):
		t.Fatal("passthrough cancel test did not exit")
	}
}

func TestPassthroughLifecycle_RejectsOverlappingResponseCreate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	cfg := passthroughLifecycleConfig()
	cfg.Gateway.OpenAIFirstOutputTimeoutSeconds = 3
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_overlap_first","model":"gpt-5.1"}}`)
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(cfg, upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()
	require.Equal(t, "response.create", gjson.GetBytes(requirePassthroughUpstreamWrite(t, upstream, time.Second), "type").String())

	created, err := readPassthroughLifecycleFrame(t, clientConn, time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.created", gjson.GetBytes(created, "type").String())
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1"}`))
	cancelWrite()
	require.NoError(t, err)

	_, err = readPassthroughLifecycleFrame(t, clientConn, time.Second)
	var websocketCloseErr coderws.CloseError
	require.ErrorAs(t, err, &websocketCloseErr)
	require.Equal(t, coderws.StatusPolicyViolation, websocketCloseErr.Code)
	require.Equal(t, "overlapping response.create is not supported", websocketCloseErr.Reason)
	select {
	case err := <-serverErr:
		var closeErr *OpenAIWSClientCloseError
		require.ErrorAs(t, err, &closeErr)
		require.Equal(t, coderws.StatusPolicyViolation, closeErr.StatusCode())
		require.Equal(t, "overlapping response.create is not supported", closeErr.Reason())
	case <-time.After(3 * time.Second):
		t.Fatal("overlapping response.create did not terminate passthrough")
	}
}

func TestPassthroughLifecycle_ActiveTurnActivityRefreshesReadTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_active_refresh","model":"gpt-5.1"}}`)
	upstream.Send(`{"type":"response.output_text.delta","response_id":"resp_active_refresh","delta":"one"}`)
	go func() {
		for _, event := range []string{
			`{"type":"response.output_text.delta","response_id":"resp_active_refresh","delta":"two"}`,
			`{"type":"response.output_text.delta","response_id":"resp_active_refresh","delta":"three"}`,
			`{"type":"response.completed","response":{"id":"resp_active_refresh","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":3}}}`,
		} {
			timer := time.NewTimer(600 * time.Millisecond)
			<-timer.C
			timer.Stop()
			upstream.Send(event)
		}
	}()
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	for _, wantType := range []string{
		"response.created",
		"response.output_text.delta",
		"response.output_text.delta",
		"response.output_text.delta",
		"response.completed",
	} {
		frame, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
		require.NoError(t, err)
		require.Equal(t, wantType, gjson.GetBytes(frame, "type").String())
	}
	require.NoError(t, clientConn.Close(coderws.StatusNormalClosure, "done"))
	select {
	case <-serverErr:
	case <-time.After(3 * time.Second):
		t.Fatal("passthrough active-turn refresh test did not exit")
	}
}

func TestPassthroughLifecycle_TerminalSwitchesToInterTurnIdleTimeout(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	cfg := passthroughLifecycleConfig()
	cfg.Gateway.OpenAIWS.ReadTimeoutSeconds = 1
	cfg.Gateway.OpenAIWS.IngressInterTurnIdleTimeoutSeconds = 2
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_idle_first","model":"gpt-5.1"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_idle_first","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`)
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(cfg, upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()
	require.Equal(t, "response.create", gjson.GetBytes(requirePassthroughUpstreamWrite(t, upstream, 3*time.Second), "type").String())

	created, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "resp_idle_first", gjson.GetBytes(created, "response.id").String())
	completed, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "resp_idle_first", gjson.GetBytes(completed, "response.id").String())
	time.Sleep(1300 * time.Millisecond)
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1","previous_response_id":"resp_idle_first"}`))
	cancelWrite()
	require.NoError(t, err)
	require.Equal(t, "response.create", gjson.GetBytes(requirePassthroughUpstreamWrite(t, upstream, 3*time.Second), "type").String())
	upstream.Send(`{"type":"response.created","response":{"id":"resp_idle_second","model":"gpt-5.1"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_idle_second","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`)
	created, err = readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "resp_idle_second", gjson.GetBytes(created, "response.id").String())
	completed, err = readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "resp_idle_second", gjson.GetBytes(completed, "response.id").String())
	_, err = readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	var websocketCloseErr coderws.CloseError
	require.ErrorAs(t, err, &websocketCloseErr)
	require.Equal(t, coderws.StatusNormalClosure, websocketCloseErr.Code)
	require.Equal(t, "websocket idle timeout", websocketCloseErr.Reason)

	select {
	case err := <-serverErr:
		var closeErr *OpenAIWSClientCloseError
		require.ErrorAs(t, err, &closeErr)
		require.Equal(t, coderws.StatusNormalClosure, closeErr.StatusCode())
		require.Equal(t, "websocket idle timeout", closeErr.Reason())
	case <-time.After(3 * time.Second):
		t.Fatal("passthrough terminal turn did not use inter-turn idle timeout")
	}
}

func TestPassthroughLifecycle_FirstOutputTimeoutRemainsBounded(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	select {
	case err := <-serverErr:
		var failoverErr *UpstreamFailoverError
		require.ErrorAs(t, err, &failoverErr)
		require.Equal(t, http.StatusGatewayTimeout, failoverErr.StatusCode)
		require.Contains(t, string(failoverErr.ResponseBody), "first_output_timeout")
	case <-time.After(2500 * time.Millisecond):
		t.Fatal("passthrough first output was left unbounded")
	}
}

func TestPassthroughLifecycle_ResponseCreatedTimeoutClosesWithoutFailover(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_preamble","model":"gpt-5.1"}}`)
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	created, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.created", gjson.GetBytes(created, "type").String())
	_, err = readPassthroughLifecycleFrame(t, clientConn, 2500*time.Millisecond)
	var websocketCloseErr coderws.CloseError
	require.ErrorAs(t, err, &websocketCloseErr)
	require.Equal(t, coderws.StatusGoingAway, websocketCloseErr.Code)
	require.Equal(t, "upstream produced no semantic output; please reconnect", websocketCloseErr.Reason)
	select {
	case err := <-serverErr:
		var failoverErr *UpstreamFailoverError
		require.NotErrorAs(t, err, &failoverErr)
		var closeErr *OpenAIWSClientCloseError
		require.ErrorAs(t, err, &closeErr)
		require.Equal(t, coderws.StatusGoingAway, closeErr.StatusCode())
		require.Equal(t, "upstream produced no semantic output; please reconnect", closeErr.Reason())
	case <-time.After(2500 * time.Millisecond):
		t.Fatal("response.created timeout did not close the passthrough connection")
	}
}

func TestPassthroughLifecycle_SecondTurnTimeoutIsNotFailoverSafe(t *testing.T) {
	gin.SetMode(gin.TestMode)
	controlCtx, cancelControl := context.WithCancelCause(context.Background())
	defer cancelControl(context.Canceled)
	upstream := newStagedPassthroughConn()
	upstream.Send(`{"type":"response.created","response":{"id":"resp_first","model":"gpt-5.1"}}`)
	upstream.Send(`{"type":"response.completed","response":{"id":"resp_first","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`)
	server, serverErr := startPassthroughLifecycleServer(t, controlCtx, newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount())
	defer server.Close()
	clientConn := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = clientConn.CloseNow() }()

	created, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.created", gjson.GetBytes(created, "type").String())
	completed, err := readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.completed", gjson.GetBytes(completed, "type").String())
	writeCtx, cancelWrite := context.WithTimeout(context.Background(), 3*time.Second)
	err = clientConn.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1","previous_response_id":"resp_first"}`))
	cancelWrite()
	require.NoError(t, err)
	upstream.Send(`{"type":"response.created","response":{"id":"resp_second","model":"gpt-5.1"}}`)

	created, err = readPassthroughLifecycleFrame(t, clientConn, 3*time.Second)
	require.NoError(t, err)
	require.Equal(t, "response.created", gjson.GetBytes(created, "type").String())
	_, err = readPassthroughLifecycleFrame(t, clientConn, 2500*time.Millisecond)
	var websocketCloseErr coderws.CloseError
	require.ErrorAs(t, err, &websocketCloseErr)
	require.Equal(t, coderws.StatusGoingAway, websocketCloseErr.Code)
	require.Equal(t, "upstream produced no semantic output; please reconnect", websocketCloseErr.Reason)
	select {
	case err := <-serverErr:
		var failoverErr *UpstreamFailoverError
		require.NotErrorAs(t, err, &failoverErr, "handler must not replay the initial request on another account for a later-turn timeout")
		var closeErr *OpenAIWSClientCloseError
		require.ErrorAs(t, err, &closeErr)
		require.Equal(t, coderws.StatusGoingAway, closeErr.StatusCode())
	case <-time.After(2500 * time.Millisecond):
		t.Fatal("second turn first semantic output was left unbounded")
	}
}

func TestOpenAIWSSessionModelEchoQueuePreservesUpdateOrder(t *testing.T) {
	var queue openAIWSSessionModelEchoQueue
	require.True(t, queue.push(""))
	require.True(t, queue.push("model-a"))
	require.True(t, queue.push("model-b"))
	model, queued := queue.pop()
	require.True(t, queued)
	require.Empty(t, model)
	model, queued = queue.pop()
	require.True(t, queued)
	require.Equal(t, "model-a", model)
	model, queued = queue.pop()
	require.True(t, queued)
	require.Equal(t, "model-b", model)
	_, queued = queue.pop()
	require.False(t, queued)
	for i := 0; i < openAIWSSessionModelEchoQueueLimit; i++ {
		require.True(t, queue.push("model"))
	}
	require.False(t, queue.push("over-limit"))
}
