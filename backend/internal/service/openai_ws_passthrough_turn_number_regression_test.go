package service

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
)

func TestPassthroughLifecycle_NextTurnNumberWaitsForTerminalCommit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	upstream := newStagedPassthroughConn()
	commitPaused, releaseCommit := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(releaseCommit) }) }
	var terminalWrites atomic.Int32
	requestedTurns := make(chan int, 2)
	outboundTurns := make(chan int, 3)
	hooks := &OpenAIWSIngressHooks{
		OnClientRequest: func(turn int, _ []byte) { requestedTurns <- turn },
		BeforeOutboundRequest: func(turn int, _ []byte, _ string) error {
			outboundTurns <- turn
			return nil
		},
		OnClientResponse: func(payload []byte, _ error) {
			if openAIWSPassthroughIsTerminalOutput(payload) && terminalWrites.Add(1) == 2 {
				close(commitPaused)
				<-releaseCommit
			}
		},
	}
	server, serverErr := startPassthroughLifecycleServer(t, ctx, newPassthroughLifecycleService(passthroughLifecycleConfig(), upstream), passthroughLifecycleAccount(), hooks)
	defer server.Close()
	client := dialPassthroughLifecycleClient(t, server)
	defer func() { _ = client.CloseNow() }()
	defer release()

	for turn := 1; turn <= 3; turn++ {
		if turn > 1 {
			writeCtx, cancelWrite := context.WithTimeout(ctx, time.Second)
			err := client.Write(writeCtx, coderws.MessageText, []byte(`{"type":"response.create","model":"gpt-5.1"}`))
			cancelWrite()
			require.NoError(t, err)
		}
		if turn == 3 {
			// Observe the real adapter waiting on terminal commit, without sleeping
			// or replacing its lifecycle with a test implementation.
			stack := make([]byte, 1<<20)
			require.Eventually(t, func() bool {
				n := runtime.Stack(stack, true)
				return bytes.Contains(stack[:n], []byte("(*openAIWSPassthroughTurnLifecycle).beginResponseCreate"))
			}, time.Second, time.Millisecond, "next client frame did not reach the terminal commit gate")
			release()
		}
		requirePassthroughUpstreamWrite(t, upstream, time.Second)
		upstream.Send(fmt.Sprintf(`{"type":"response.created","response":{"id":"resp_turn_%d","model":"gpt-5.1"}}`, turn))
		upstream.Send(fmt.Sprintf(`{"type":"response.completed","response":{"id":"resp_turn_%d","model":"gpt-5.1","usage":{"input_tokens":1,"output_tokens":1}}}`, turn))
		for range 2 {
			_, err := readPassthroughLifecycleFrame(t, client, time.Second)
			require.NoError(t, err)
		}
		if turn == 2 {
			select {
			case <-commitPaused:
			case <-time.After(time.Second):
				t.Fatal("second terminal was not paused before its commit")
			}
		}
	}
	require.Len(t, requestedTurns, 2)
	require.Equal(t, []int{2, 3}, []int{<-requestedTurns, <-requestedTurns})
	require.Len(t, outboundTurns, 3)
	require.Equal(t, []int{1, 2, 3}, []int{<-outboundTurns, <-outboundTurns, <-outboundTurns})
	require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
	require.NoError(t, <-serverErr)
}
