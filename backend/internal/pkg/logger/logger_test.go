package logger

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestClose_LegacyPrintfFallsBackToStdLog(t *testing.T) {
	Close()

	logR, logW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create log pipe: %v", err)
	}
	prevWriter := log.Writer()
	prevFlags := log.Flags()
	prevPrefix := log.Prefix()
	log.SetOutput(logW)
	log.SetFlags(0)
	log.SetPrefix("")

	t.Cleanup(func() {
		Close()
		log.SetOutput(prevWriter)
		log.SetFlags(prevFlags)
		log.SetPrefix(prevPrefix)
		_ = logR.Close()
		_ = logW.Close()
	})

	err = Init(InitOptions{
		Level:  "debug",
		Format: "json",
		Output: OutputOptions{ToStdout: false, ToFile: false},
	})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	Close()
	log.SetOutput(logW)
	log.SetFlags(0)
	log.SetPrefix("")
	LegacyPrintf("service.test", "fallback after close")
	_ = logW.Close()

	logBytes, _ := io.ReadAll(logR)
	if !strings.Contains(string(logBytes), "fallback after close") {
		t.Fatalf("LegacyPrintf should fallback to std log after Close, got: %s", string(logBytes))
	}
}

func TestClose_CurrentLevelResetsToDefault(t *testing.T) {
	Close()
	t.Cleanup(Close)

	err := Init(InitOptions{
		Level:  "debug",
		Format: "json",
		Output: OutputOptions{ToStdout: false, ToFile: false},
	})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if got := CurrentLevel(); got != "debug" {
		t.Fatalf("CurrentLevel()=%s want debug before Close", got)
	}

	Close()

	if got := CurrentLevel(); got != "info" {
		t.Fatalf("CurrentLevel()=%s want info after Close", got)
	}
}

func TestClose_SlogFallsBackToPreviousDefault(t *testing.T) {
	Close()
	t.Cleanup(Close)

	var buf bytes.Buffer
	prevDefault := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, nil)))
	t.Cleanup(func() {
		slog.SetDefault(prevDefault)
	})

	err := Init(InitOptions{
		Level:  "info",
		Format: "json",
		Output: OutputOptions{ToStdout: false, ToFile: false},
	})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	Close()
	slog.Info("slog fallback after close")

	if !strings.Contains(buf.String(), "slog fallback after close") {
		t.Fatalf("slog should fallback to previous default after Close, got: %s", buf.String())
	}
}

func TestReconfigure_TracksPreviousClosersUntilClose(t *testing.T) {
	Close()
	t.Cleanup(Close)

	tmpDir := t.TempDir()
	firstPath := filepath.Join(tmpDir, "first.log")
	secondPath := filepath.Join(tmpDir, "second.log")

	err := Init(InitOptions{
		Level:  "info",
		Format: "json",
		Output: OutputOptions{ToStdout: false, ToFile: true, FilePath: firstPath},
	})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if got := len(resourceClose); got != 1 {
		t.Fatalf("len(resourceClose)=%d want 1 after first Init", got)
	}

	err = Reconfigure(func(opts *InitOptions) error {
		opts.Output.FilePath = secondPath
		return nil
	})
	if err != nil {
		t.Fatalf("Reconfigure() error: %v", err)
	}

	if got := len(resourceClose); got != 2 {
		t.Fatalf("len(resourceClose)=%d want 2 after Reconfigure", got)
	}

	Close()
	if got := len(resourceClose); got != 0 {
		t.Fatalf("len(resourceClose)=%d want 0 after Close", got)
	}
}

func TestReconfigure_PreviousLoggerRemainsUsable(t *testing.T) {
	Close()
	t.Cleanup(Close)

	tmpDir := t.TempDir()
	firstPath := filepath.Join(tmpDir, "first.log")
	secondPath := filepath.Join(tmpDir, "second.log")

	err := Init(InitOptions{
		Level:       "info",
		Format:      "json",
		ServiceName: "sub2api",
		Environment: "test",
		Output:      OutputOptions{ToStdout: false, ToFile: true, FilePath: firstPath},
	})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	oldLogger := L().Named("old")

	err = Reconfigure(func(opts *InitOptions) error {
		opts.Output.FilePath = secondPath
		return nil
	})
	if err != nil {
		t.Fatalf("Reconfigure() error: %v", err)
	}

	oldLogger.Info("written by old logger after reconfigure")
	L().Info("written by new logger after reconfigure")
	Close()

	firstBytes, err := os.ReadFile(firstPath)
	if err != nil {
		t.Fatalf("read first log file: %v", err)
	}
	secondBytes, err := os.ReadFile(secondPath)
	if err != nil {
		t.Fatalf("read second log file: %v", err)
	}

	if !strings.Contains(string(firstBytes), "written by old logger after reconfigure") {
		t.Fatalf("old logger should keep writing to original file, got: %s", string(firstBytes))
	}
	if !strings.Contains(string(secondBytes), "written by new logger after reconfigure") {
		t.Fatalf("new logger should write to reconfigured file, got: %s", string(secondBytes))
	}
	if err := os.Remove(firstPath); err != nil {
		t.Fatalf("first log file should be removable after Close: %v", err)
	}
	if err := os.Remove(secondPath); err != nil {
		t.Fatalf("second log file should be removable after Close: %v", err)
	}
}

func TestInit_DualOutput(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "logs", "sub2api.log")

	origStdout := os.Stdout
	origStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	t.Cleanup(func() {
		Close()
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = stdoutR.Close()
		_ = stderrR.Close()
		_ = stdoutW.Close()
		_ = stderrW.Close()
	})

	err = Init(InitOptions{
		Level:       "debug",
		Format:      "json",
		ServiceName: "sub2api",
		Environment: "test",
		Output: OutputOptions{
			ToStdout: true,
			ToFile:   true,
			FilePath: logPath,
		},
		Rotation: RotationOptions{
			MaxSizeMB:  10,
			MaxBackups: 2,
			MaxAgeDays: 1,
		},
		Sampling: SamplingOptions{Enabled: false},
	})
	if err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	L().Info("dual-output-info")
	L().Warn("dual-output-warn")
	Sync()

	_ = stdoutW.Close()
	_ = stderrW.Close()
	stdoutBytes, _ := io.ReadAll(stdoutR)
	stderrBytes, _ := io.ReadAll(stderrR)
	stdoutText := string(stdoutBytes)
	stderrText := string(stderrBytes)

	if !strings.Contains(stdoutText, "dual-output-info") {
		t.Fatalf("stdout missing info log: %s", stdoutText)
	}
	if !strings.Contains(stderrText, "dual-output-warn") {
		t.Fatalf("stderr missing warn log: %s", stderrText)
	}

	fileBytes, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read log file: %v", err)
	}
	fileText := string(fileBytes)
	if !strings.Contains(fileText, "dual-output-info") || !strings.Contains(fileText, "dual-output-warn") {
		t.Fatalf("file missing logs: %s", fileText)
	}
}

func TestInit_FileOutputFailureDowngrade(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	_, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	stderrR, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	t.Cleanup(func() {
		Close()
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = stdoutW.Close()
		_ = stderrR.Close()
		_ = stderrW.Close()
	})

	err = Init(InitOptions{
		Level:  "info",
		Format: "json",
		Output: OutputOptions{
			ToStdout: true,
			ToFile:   true,
			FilePath: filepath.Join(os.DevNull, "logs", "sub2api.log"),
		},
		Rotation: RotationOptions{
			MaxSizeMB:  10,
			MaxBackups: 1,
			MaxAgeDays: 1,
		},
	})
	if err != nil {
		t.Fatalf("Init() should downgrade instead of failing, got: %v", err)
	}

	_ = stderrW.Close()
	stderrBytes, _ := io.ReadAll(stderrR)
	if !strings.Contains(string(stderrBytes), "日志文件输出初始化失败") {
		t.Fatalf("stderr should contain fallback warning, got: %s", string(stderrBytes))
	}
}

func TestInit_CallerShouldPointToCallsite(t *testing.T) {
	origStdout := os.Stdout
	origStderr := os.Stderr
	stdoutR, stdoutW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe: %v", err)
	}
	_, stderrW, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stderr pipe: %v", err)
	}
	os.Stdout = stdoutW
	os.Stderr = stderrW
	t.Cleanup(func() {
		Close()
		os.Stdout = origStdout
		os.Stderr = origStderr
		_ = stdoutR.Close()
		_ = stdoutW.Close()
		_ = stderrW.Close()
	})

	if err := Init(InitOptions{
		Level:       "info",
		Format:      "json",
		ServiceName: "sub2api",
		Environment: "test",
		Caller:      true,
		Output: OutputOptions{
			ToStdout: true,
			ToFile:   false,
		},
		Sampling: SamplingOptions{Enabled: false},
	}); err != nil {
		t.Fatalf("Init() error: %v", err)
	}

	L().Info("caller-check")
	Sync()
	_ = stdoutW.Close()
	logBytes, _ := io.ReadAll(stdoutR)

	var line string
	for _, item := range strings.Split(string(logBytes), "\n") {
		if strings.Contains(item, "caller-check") {
			line = item
			break
		}
	}
	if line == "" {
		t.Fatalf("log output missing caller-check: %s", string(logBytes))
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err != nil {
		t.Fatalf("parse log json failed: %v, line=%s", err, line)
	}
	caller, _ := payload["caller"].(string)
	if !strings.Contains(caller, "logger_test.go:") {
		t.Fatalf("caller should point to this test file, got: %s", caller)
	}
}
