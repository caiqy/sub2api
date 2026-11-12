package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"
)

const SettingKeyInputContextLimit = "input_context_limit"

type InputContextLimitSettings struct {
	Enabled     bool             `json:"enabled"`
	GroupLimits map[string]int32 `json:"group_limits"`
}

func (v *InputContextLimitSettings) UnmarshalJSON(data []byte) error {
	type plain InputContextLimitSettings
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	if fields == nil {
		return fmt.Errorf("settings must be an object")
	}
	if raw, ok := fields["enabled"]; ok && string(raw) == "null" {
		return fmt.Errorf("enabled must be boolean")
	}
	var limits map[string]json.RawMessage
	if raw, ok := fields["group_limits"]; ok {
		if string(raw) == "null" {
			return fmt.Errorf("group_limits must be an object")
		}
		if err := json.Unmarshal(raw, &limits); err != nil {
			return err
		}
		for id, n := range limits {
			if string(n) == "null" {
				return fmt.Errorf("group %s limit must be an integer", id)
			}
		}
	}
	return json.Unmarshal(data, (*plain)(v))
}

type cachedInputContextLimit struct {
	settings InputContextLimitSettings
	expires  time.Time
	version  int64
}

func cloneInputContextLimit(v InputContextLimitSettings) InputContextLimitSettings {
	m := make(map[string]int32, len(v.GroupLimits))
	for k, n := range v.GroupLimits {
		m[k] = n
	}
	v.GroupLimits = m
	return v
}
func (s *SettingService) GetInputContextLimitSettings(ctx context.Context) (*InputContextLimitSettings, error) {
	v := &InputContextLimitSettings{GroupLimits: map[string]int32{}}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyInputContextLimit)
	if errors.Is(err, ErrSettingNotFound) {
		return v, nil
	}
	if err != nil {
		return nil, err
	}
	if raw != "" {
		if err = json.Unmarshal([]byte(raw), v); err != nil {
			return nil, fmt.Errorf("decode input context limit: %w", err)
		}
	}
	if v.GroupLimits == nil {
		v.GroupLimits = map[string]int32{}
	}
	for id, n := range v.GroupLimits {
		parsed, e := strconv.ParseInt(id, 10, 64)
		if e != nil || parsed <= 0 || n < 0 {
			return nil, fmt.Errorf("invalid input context limit for group %q", id)
		}
	}
	return v, nil
}
func (s *SettingService) SetInputContextLimitSettings(ctx context.Context, v *InputContextLimitSettings) error {
	if v == nil {
		return fmt.Errorf("settings are required")
	}
	next := cloneInputContextLimit(*v)
	for id, n := range next.GroupLimits {
		parsed, err := strconv.ParseInt(id, 10, 64)
		if err != nil || parsed <= 0 || strconv.FormatInt(parsed, 10) != id || n < 0 {
			return fmt.Errorf("invalid group or limit: %q", id)
		}
		if s.defaultSubGroupReader == nil {
			return fmt.Errorf("group validation unavailable")
		}
		g, err := s.defaultSubGroupReader.GetByID(ctx, parsed)
		if err != nil {
			return fmt.Errorf("validate input context group %s: %w", id, err)
		}
		if g == nil {
			return fmt.Errorf("group %s does not exist", id)
		}
	}
	raw, err := json.Marshal(next)
	if err != nil {
		return err
	}
	s.inputContextLimitMu.Lock()
	defer s.inputContextLimitMu.Unlock()
	if err = s.settingRepo.Set(ctx, SettingKeyInputContextLimit, string(raw)); err != nil {
		return err
	}
	s.inputContextLimitCache.Store(&cachedInputContextLimit{next, time.Now().Add(60 * time.Second), time.Now().UnixNano()})
	return nil
}
func (s *SettingService) inputContextLimitSnapshot(ctx context.Context) *cachedInputContextLimit {
	if s == nil || s.settingRepo == nil {
		return &cachedInputContextLimit{}
	}
	if v, ok := s.inputContextLimitCache.Load().(*cachedInputContextLimit); ok && time.Now().Before(v.expires) {
		return v
	}
	result, _, _ := s.inputContextLimitSF.Do(SettingKeyInputContextLimit, func() (any, error) {
		s.inputContextLimitMu.Lock()
		defer s.inputContextLimitMu.Unlock()
		prior, _ := s.inputContextLimitCache.Load().(*cachedInputContextLimit)
		if prior != nil && time.Now().Before(prior.expires) {
			return prior, nil
		}
		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		settings, err := s.GetInputContextLimitSettings(dbCtx)
		ttl := 60 * time.Second
		version := time.Now().UnixNano()
		if err != nil {
			slog.Warn("input context limit refresh failed; retaining last known policy", "error", err)
			settings = &InputContextLimitSettings{}
			ttl = 5 * time.Second
			if prior != nil {
				settings = &prior.settings
				version = prior.version
			}
		}
		// Orphan configuration remains visible to administrators but is not applied.
		effective := cloneInputContextLimit(*settings)
		if err == nil && s.defaultSubGroupReader != nil {
			for id := range effective.GroupLimits {
				n, _ := strconv.ParseInt(id, 10, 64)
				g, e := s.defaultSubGroupReader.GetByID(dbCtx, n)
				if e == nil && g == nil || errors.Is(e, ErrGroupNotFound) {
					delete(effective.GroupLimits, id)
				}
			}
		}
		v := &cachedInputContextLimit{effective, time.Now().Add(ttl), version}
		s.inputContextLimitCache.Store(v)
		return v, nil
	})
	return result.(*cachedInputContextLimit)
}
