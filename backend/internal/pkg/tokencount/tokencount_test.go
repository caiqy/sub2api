package tokencount

import (
	"strings"
	"testing"
)

func TestCountTokens_Empty(t *testing.T) {
	if got := CountTokens(""); got != 0 {
		t.Errorf("CountTokens(\"\") = %d, want 0", got)
	}
}

func TestCountTokens_English(t *testing.T) {
	text := "Hello, world! This is a test of the token counter."
	got := CountTokens(text)
	if got <= 0 {
		t.Errorf("CountTokens(english) = %d, want > 0", got)
	}
	// Sanity: English text of ~50 chars should be roughly 10-15 tokens
	if got > 50 {
		t.Errorf("CountTokens(english) = %d, unexpectedly large for %d chars", got, len(text))
	}
}

func TestCountTokens_Chinese(t *testing.T) {
	text := "你好世界，这是一个测试。"
	got := CountTokens(text)
	if got <= 0 {
		t.Errorf("CountTokens(chinese) = %d, want > 0", got)
	}
}

func TestCountTokens_LargeText(t *testing.T) {
	// Simulate a large code block output (like tool_use JSON)
	text := strings.Repeat(`{"description": "深入探索账号选择算法细节", "prompt": "Continue the exploration"}`, 100)
	got := CountTokens(text)
	if got <= 0 {
		t.Errorf("CountTokens(large) = %d, want > 0", got)
	}
}

func TestEstimateTokensFallback_Empty(t *testing.T) {
	if got := EstimateTokensFallback(""); got != 0 {
		t.Errorf("EstimateTokensFallback(\"\") = %d, want 0", got)
	}
}

func TestEstimateTokensFallback_English(t *testing.T) {
	text := "Hello world this is a test"
	got := EstimateTokensFallback(text)
	// 26 chars, mostly ASCII → (26+3)/4 = 7
	if got != 7 {
		t.Errorf("EstimateTokensFallback(english) = %d, want 7", got)
	}
}

func TestEstimateTokensFallback_Chinese(t *testing.T) {
	text := "你好世界测试"
	got := EstimateTokensFallback(text)
	// 6 runes, all non-ASCII → 6
	if got != 6 {
		t.Errorf("EstimateTokensFallback(chinese) = %d, want 6", got)
	}
}

func BenchmarkCountTokens(b *testing.B) {
	text := strings.Repeat("Hello world, this is a benchmark test for token counting. ", 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		CountTokens(text)
	}
}
