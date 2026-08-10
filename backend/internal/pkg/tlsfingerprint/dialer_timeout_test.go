package tlsfingerprint

import (
	"bufio"
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	utls "github.com/refraction-networking/utls"
)

const testDialDeadline = 75 * time.Millisecond

func TestDialerAddsDeadlineToBackgroundContext(t *testing.T) {
	type deadlineObservation struct {
		deadline time.Time
		ok       bool
	}
	deadlineCh := make(chan deadlineObservation, 1)
	dialer := NewDialer(nil, func(ctx context.Context, _, _ string) (net.Conn, error) {
		deadline, ok := ctx.Deadline()
		deadlineCh <- deadlineObservation{deadline: deadline, ok: ok}
		return nil, errors.New("stop after observing deadline")
	})

	_, err := dialer.DialTLSContext(context.Background(), "tcp", "example.test:443")
	if err == nil {
		t.Fatal("DialTLSContext returned nil error")
	}

	observation := <-deadlineCh
	if !observation.ok {
		t.Fatal("background dial context is missing a deadline")
	}
	deadline := observation.deadline
	remaining := time.Until(deadline)
	if remaining < 9*time.Second || remaining > 11*time.Second {
		t.Fatalf("background dial deadline remaining = %s, want about 10s", remaining)
	}
}

func TestSOCKS5ProxyDialerHonorsContextDeadline(t *testing.T) {
	addr, observed, peerClosed, release := startStalledProxy(t, false)
	proxyURL, err := url.Parse("socks5://" + addr)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), testDialDeadline)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		conn, err := NewSOCKS5ProxyDialer(nil, proxyURL).DialTLSContext(ctx, "tcp", "example.test:443")
		if conn != nil {
			_ = conn.Close()
		}
		done <- err
	}()

	awaitSignal(t, observed, "SOCKS5 handshake")
	awaitDialError(t, done, release)
	awaitSignal(t, peerClosed, "SOCKS5 connection close")
}

func TestHTTPProxyDialerCONNECTHonorsContextDeadline(t *testing.T) {
	addr, observed, peerClosed, release := startStalledProxy(t, true)
	proxyURL, err := url.Parse("http://" + addr)
	if err != nil {
		t.Fatalf("parse proxy URL: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), testDialDeadline)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		conn, err := NewHTTPProxyDialer(nil, proxyURL).DialTLSContext(ctx, "tcp", "example.test:443")
		if conn != nil {
			_ = conn.Close()
		}
		done <- err
	}()

	awaitSignal(t, observed, "HTTP CONNECT write")
	awaitDialError(t, done, release)
	awaitSignal(t, peerClosed, "HTTP CONNECT connection close")
}

func TestHTTPProxyDialerCONNECTWriteHonorsContextCancellation(t *testing.T) {
	client, proxy := net.Pipe()
	writeStarted := make(chan struct{})
	closeSignal := &closeSignalConn{Conn: client, closed: make(chan struct{})}
	conn := &writeSignalConn{Conn: closeSignal, started: writeStarted}
	dialer := &HTTPProxyDialer{
		proxyURL: &url.URL{Scheme: "http", Host: "proxy.test:80"},
		dialContext: func(context.Context, string, string) (net.Conn, error) {
			return conn, nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	finished := make(chan struct{})
	t.Cleanup(func() {
		_ = client.Close()
		_ = proxy.Close()
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Error("CONNECT write dial goroutine leaked")
		}
	})
	go func() {
		defer close(finished)
		result, err := dialer.DialTLSContext(ctx, "tcp", "example.test:443")
		if result != nil {
			_ = result.Close()
		}
		done <- err
	}()

	awaitSignal(t, writeStarted, "HTTP CONNECT request write")
	cancel()
	awaitCanceledDial(t, done, "HTTP CONNECT request write")
	awaitSignal(t, closeSignal.closed, "HTTP CONNECT request write connection close")
}

func TestHTTPProxyDialerCONNECTResponseReadHonorsContextCancellation(t *testing.T) {
	client, proxy := net.Pipe()
	readStarted := make(chan struct{})
	closeSignal := &closeSignalConn{Conn: client, closed: make(chan struct{})}
	conn := &readSignalConn{Conn: closeSignal, started: readStarted}
	dialer := &HTTPProxyDialer{
		proxyURL: &url.URL{Scheme: "http", Host: "proxy.test:80"},
		dialContext: func(context.Context, string, string) (net.Conn, error) {
			return conn, nil
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	requestRead := make(chan error, 1)
	go func() {
		req, err := http.ReadRequest(bufio.NewReader(proxy))
		if err == nil && req.Method != http.MethodConnect {
			err = errors.New("proxy received a non-CONNECT request")
		}
		requestRead <- err
	}()

	done := make(chan error, 1)
	finished := make(chan struct{})
	t.Cleanup(func() {
		_ = client.Close()
		_ = proxy.Close()
		select {
		case <-finished:
		case <-time.After(time.Second):
			t.Error("CONNECT response read dial goroutine leaked")
		}
	})
	go func() {
		defer close(finished)
		result, err := dialer.DialTLSContext(ctx, "tcp", "example.test:443")
		if result != nil {
			_ = result.Close()
		}
		done <- err
	}()

	select {
	case err := <-requestRead:
		if err != nil {
			t.Fatalf("proxy read CONNECT request: %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for proxy to read CONNECT request")
	}
	awaitSignal(t, readStarted, "HTTP CONNECT response read")
	cancel()
	awaitCanceledDial(t, done, "HTTP CONNECT response read")
	awaitSignal(t, closeSignal.closed, "HTTP CONNECT response read connection close")
}

func TestUTLSHandshakeSetsContextDeadline(t *testing.T) {
	client, server := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = server.Close()
	})
	conn := &deadlineRecordingConn{Conn: client}
	ctx, cancel := context.WithTimeout(context.Background(), testDialDeadline)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		_, err := performTLSHandshake(ctx, conn, nil, "example.test:443")
		done <- err
	}()

	awaitDialError(t, done, func() { _ = server.Close() })
	if !conn.hasNonZeroDeadline() {
		t.Fatal("uTLS handshake did not set the context deadline on its connection")
	}
}

func TestUTLSHandshakeClearsTemporaryDeadlineAfterSuccess(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.StartTLS()
	t.Cleanup(server.Close)

	rawConn, err := net.Dial("tcp", strings.TrimPrefix(server.URL, "https://"))
	if err != nil {
		t.Fatalf("dial local TLS server: %v", err)
	}
	conn := &deadlineRecordingConn{Conn: rawConn}
	t.Cleanup(func() { _ = conn.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	result, err := performTLSHandshakeWithConfig(ctx, conn, nil, "127.0.0.1:443", &utls.Config{
		ServerName:         "127.0.0.1",
		InsecureSkipVerify: true,
	})
	if err != nil {
		t.Fatalf("perform local uTLS handshake: %v", err)
	}
	t.Cleanup(func() { _ = result.Close() })

	deadlines := conn.recordedDeadlines()
	if len(deadlines) < 2 || deadlines[len(deadlines)-1] != (time.Time{}) {
		t.Fatalf("temporary connection deadline was not cleared: %v", deadlines)
	}
}

func awaitSignal(t *testing.T, ch <-chan struct{}, name string) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(time.Second):
		t.Fatalf("timed out waiting for %s", name)
	}
}

func awaitDialError(t *testing.T, done <-chan error, release func()) {
	t.Helper()
	select {
	case err := <-done:
		if err == nil {
			t.Fatal("dial unexpectedly succeeded")
		}
	case <-time.After(time.Second):
		release()
		select {
		case <-done:
		case <-time.After(time.Second):
			t.Fatal("dial goroutine remained blocked after its connection was closed")
		}
		t.Fatal("dial did not honor its context deadline")
	}
}

func awaitCanceledDial(t *testing.T, done <-chan error, phase string) {
	t.Helper()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("%s error = %v, want context.Canceled", phase, err)
		}
	case <-time.After(time.Second):
		t.Fatalf("%s remained blocked after explicit cancellation", phase)
	}
}

func startStalledProxy(t *testing.T, readCONNECT bool) (string, <-chan struct{}, <-chan struct{}, func()) {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	observed := make(chan struct{})
	peerClosed := make(chan struct{})
	stop := make(chan struct{})
	done := make(chan struct{})
	var (
		mu   sync.Mutex
		conn net.Conn
		once sync.Once
	)

	go func() {
		defer close(done)
		accepted, err := listener.Accept()
		if err != nil {
			return
		}
		mu.Lock()
		conn = accepted
		mu.Unlock()
		defer func() { _ = accepted.Close() }()

		if readCONNECT {
			req, err := http.ReadRequest(bufio.NewReader(accepted))
			if err != nil || req.Method != http.MethodConnect {
				return
			}
		} else {
			buf := make([]byte, 1)
			if _, err := accepted.Read(buf); err != nil {
				return
			}
		}
		close(observed)
		_ = accepted.SetReadDeadline(time.Now().Add(time.Second))
		for {
			if _, err := accepted.Read(make([]byte, 1)); err != nil {
				close(peerClosed)
				break
			}
		}
		<-stop
	}()

	release := func() {
		once.Do(func() {
			close(stop)
			_ = listener.Close()
			mu.Lock()
			if conn != nil {
				_ = conn.Close()
			}
			mu.Unlock()
		})
		<-done
	}
	t.Cleanup(release)
	return listener.Addr().String(), observed, peerClosed, release
}

type deadlineRecordingConn struct {
	net.Conn
	mu        sync.Mutex
	deadlines []time.Time
}

type writeSignalConn struct {
	net.Conn
	started chan struct{}
	once    sync.Once
}

func (c *writeSignalConn) Write(p []byte) (int, error) {
	c.once.Do(func() { close(c.started) })
	return c.Conn.Write(p)
}

type closeSignalConn struct {
	net.Conn
	closed chan struct{}
	once   sync.Once
}

func (c *closeSignalConn) Close() error {
	c.once.Do(func() { close(c.closed) })
	return c.Conn.Close()
}

type readSignalConn struct {
	net.Conn
	started chan struct{}
	once    sync.Once
}

func (c *readSignalConn) Read(p []byte) (int, error) {
	c.once.Do(func() { close(c.started) })
	return c.Conn.Read(p)
}

func (c *deadlineRecordingConn) SetDeadline(deadline time.Time) error {
	c.mu.Lock()
	c.deadlines = append(c.deadlines, deadline)
	c.mu.Unlock()
	return c.Conn.SetDeadline(deadline)
}

func (c *deadlineRecordingConn) hasNonZeroDeadline() bool {
	for _, deadline := range c.recordedDeadlines() {
		if !deadline.IsZero() {
			return true
		}
	}
	return false
}

func (c *deadlineRecordingConn) recordedDeadlines() []time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return append([]time.Time(nil), c.deadlines...)
}
