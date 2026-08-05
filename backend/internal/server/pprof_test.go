package server

import (
	"bytes"
	"context"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestStartPprofServerDisabled(t *testing.T) {
	instance, err := StartPprofServer(config.PprofConfig{})
	require.NoError(t, err)
	require.Nil(t, instance)
}

func TestStartPprofServerServesProfilesAndShutsDown(t *testing.T) {
	defaultMux := http.DefaultServeMux
	failingMux := http.NewServeMux()
	failingMux.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "default mux used", http.StatusInternalServerError)
	})
	http.DefaultServeMux = failingMux
	t.Cleanup(func() { http.DefaultServeMux = defaultMux })

	instance, err := StartPprofServer(config.PprofConfig{Enabled: true, Address: "127.0.0.1:0"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = instance.Shutdown(context.Background()) })

	client := &http.Client{
		Transport: &http.Transport{DisableKeepAlives: true},
		Timeout:   time.Second,
	}
	t.Cleanup(client.CloseIdleConnections)
	url := "http://" + instance.Address() + "/debug/pprof/heap"
	resp, err := client.Get(url)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.Equal(t, "application/octet-stream", resp.Header.Get("Content-Type"))
	require.Equal(t, `attachment; filename="heap"`, resp.Header.Get("Content-Disposition"))
	require.NoError(t, resp.Body.Close())

	require.NoError(t, instance.Shutdown(context.Background()))
	_, err = client.Get(url)
	require.Error(t, err)
}

func TestStartPprofServerRejectsNonLoopbackAddress(t *testing.T) {
	for _, address := range []string{"0.0.0.0:6060", "[::]:0", "192.0.2.1:0"} {
		t.Run(address, func(t *testing.T) {
			instance, err := StartPprofServer(config.PprofConfig{Enabled: true, Address: address})
			require.Nil(t, instance)
			require.ErrorContains(t, err, "loopback")
		})
	}
}

func TestStartPprofServerReturnsOccupiedPortErrorSynchronously(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = listener.Close() })

	instance, err := StartPprofServer(config.PprofConfig{Enabled: true, Address: listener.Addr().String()})
	require.Nil(t, instance)
	require.ErrorContains(t, err, "listen pprof")
}

func TestStartPprofServerLogsUnexpectedServeErrors(t *testing.T) {
	output := &lockedBuffer{}
	previousOutput := log.Writer()
	log.SetOutput(output)
	t.Cleanup(func() { log.SetOutput(previousOutput) })

	instance, err := StartPprofServer(config.PprofConfig{Enabled: true, Address: "127.0.0.1:0"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = instance.Shutdown(context.Background()) })

	require.NoError(t, instance.listener.Close())
	require.Eventually(t, func() bool {
		return strings.Contains(output.String(), "Pprof server serve failed")
	}, time.Second, 10*time.Millisecond)
}

type lockedBuffer struct {
	mu     sync.Mutex
	buffer bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buffer.String()
}
