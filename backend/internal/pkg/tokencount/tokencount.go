// Package tokencount provides output token estimation for upstream responses
// that do not report accurate output_tokens (e.g., some Anthropic-compatible proxies).
//
// It uses tiktoken-go/tokenizer with the o200k_base encoding (GPT-4o family)
// as a reasonable cross-model approximation. The encoder is lazily initialized once
// on first use via sync.Once and reused for all subsequent calls (thread-safe).
package tokencount

import (
	"sync"

	"github.com/tiktoken-go/tokenizer"
)

var (
	encoder  tokenizer.Codec
	initOnce sync.Once
	initErr  error
)

// ensureEncoder lazily initialises the shared encoder.
func ensureEncoder() (tokenizer.Codec, error) {
	initOnce.Do(func() {
		encoder, initErr = tokenizer.Get(tokenizer.O200kBase)
	})
	return encoder, initErr
}

// CountTokens returns the token count of the given text using the o200k_base
// encoding. It uses Codec.Count() which only returns the count without
// allocating the full token slice, reducing memory overhead.
// If the encoder fails to initialise it falls back to a simple
// heuristic (see EstimateTokensFallback).
func CountTokens(text string) int {
	if text == "" {
		return 0
	}
	enc, err := ensureEncoder()
	if err != nil {
		return EstimateTokensFallback(text)
	}
	count, err := enc.Count(text)
	if err != nil {
		return EstimateTokensFallback(text)
	}
	return count
}

// EstimateTokensFallback provides a rough token count without a real tokenizer.
// English-heavy text: ~4 chars/token; CJK-heavy text: ~1 rune/token.
// This mirrors the logic already used in estimateTokensForText elsewhere in
// the codebase.
func EstimateTokensFallback(s string) int {
	if s == "" {
		return 0
	}
	runes := []rune(s)
	if len(runes) == 0 {
		return 0
	}
	ascii := 0
	for _, r := range runes {
		if r <= 0x7f {
			ascii++
		}
	}
	asciiRatio := float64(ascii) / float64(len(runes))
	if asciiRatio >= 0.8 {
		return (len(runes) + 3) / 4
	}
	return len(runes)
}
