package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	accountPassthroughEnabledKey = "passthrough_fields_enabled"
	accountPassthroughRulesKey   = "passthrough_field_rules"
)

var (
	passthroughBodyPathPattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*(\.[A-Za-z_][A-Za-z0-9_]*)*$`)
)

type PassthroughFieldRule struct {
	Target    string `json:"target"`
	Mode      string `json:"mode"`
	Key       string `json:"key"`
	Value     string `json:"value,omitempty"`
	SourceKey string `json:"source_key,omitempty"`
}

type NormalizePassthroughFieldsInput struct {
	ExistingType              string
	RequestedType             string
	Extra                     map[string]any
	ExplicitlySubmittedConfig bool
}

func NormalizeAccountPassthroughFields(input NormalizePassthroughFieldsInput) (map[string]any, error) {
	requestedType := strings.TrimSpace(input.RequestedType)
	if requestedType == "" {
		requestedType = strings.TrimSpace(input.ExistingType)
	}

	normalized := clonePassthroughExtra(input.Extra)
	if requestedType != AccountTypeAPIKey {
		if input.ExplicitlySubmittedConfig && hasPassthroughConfigKeys(normalized) {
			return nil, fmt.Errorf("passthrough field rules are only supported for apikey accounts")
		}
		delete(normalized, accountPassthroughEnabledKey)
		delete(normalized, accountPassthroughRulesKey)
		return normalized, nil
	}

	enabled := false
	if rawEnabled, ok := normalized[accountPassthroughEnabledKey]; ok {
		parsedEnabled, ok := rawEnabled.(bool)
		if !ok {
			return nil, fmt.Errorf("invalid passthrough enabled flag")
		}
		enabled = parsedEnabled
	}

	rules, hasRules, err := parsePassthroughRules(normalized[accountPassthroughRulesKey])
	if err != nil {
		return nil, err
	}

	if !hasRules {
		delete(normalized, accountPassthroughRulesKey)
	} else {
		normalized[accountPassthroughRulesKey] = rules
	}
	if _, exists := normalized[accountPassthroughEnabledKey]; exists {
		normalized[accountPassthroughEnabledKey] = enabled
	}

	return normalized, nil
}

func AccountPassthroughFieldRules(account *Account) (enabled bool, rules []PassthroughFieldRule, err error) {
	if account == nil || account.Type != AccountTypeAPIKey {
		return false, nil, nil
	}
	normalized, err := NormalizeAccountPassthroughFields(NormalizePassthroughFieldsInput{
		RequestedType: account.Type,
		Extra:         account.Extra,
	})
	if err != nil {
		return false, nil, err
	}
	if rawEnabled, ok := normalized[accountPassthroughEnabledKey].(bool); ok {
		enabled = rawEnabled
	}
	if rawRules, ok := normalized[accountPassthroughRulesKey].([]PassthroughFieldRule); ok {
		rules = rawRules
	}
	return enabled, rules, nil
}

func parsePassthroughRules(raw any) ([]PassthroughFieldRule, bool, error) {
	if raw == nil {
		return nil, false, nil
	}

	switch typed := raw.(type) {
	case []PassthroughFieldRule:
		validated := make([]PassthroughFieldRule, 0, len(typed))
		seenRulesByTarget := map[string]map[string]PassthroughFieldRule{}
		for _, rule := range typed {
			normalizedRule, err := normalizePassthroughRule(rule)
			if err != nil {
				return nil, false, err
			}
			if err := validatePassthroughRuleDuplicate(seenRulesByTarget, normalizedRule); err != nil {
				return nil, false, err
			}
			validated = append(validated, normalizedRule)
		}
		return validated, true, nil
	case []any:
		validated := make([]PassthroughFieldRule, 0, len(typed))
		seenRulesByTarget := map[string]map[string]PassthroughFieldRule{}
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, false, fmt.Errorf("invalid passthrough field rule")
			}
			rule := PassthroughFieldRule{}
			if value, exists := entry["target"]; exists {
				stringValue, ok := value.(string)
				if !ok {
					return nil, false, fmt.Errorf("passthrough target must be a string")
				}
				rule.Target = stringValue
			}
			if value, exists := entry["key"]; exists {
				stringValue, ok := value.(string)
				if !ok {
					return nil, false, fmt.Errorf("passthrough key must be a string")
				}
				rule.Key = stringValue
			}
			if value, exists := entry["mode"]; exists {
				stringValue, ok := value.(string)
				if !ok {
					return nil, false, fmt.Errorf("passthrough mode must be a string: %s", strings.TrimSpace(rule.Key))
				}
				rule.Mode = stringValue
			}
			if value, exists := entry["value"]; exists {
				stringValue, ok := value.(string)
				if !ok {
					return nil, false, fmt.Errorf("passthrough inject value must be a string: %s", strings.TrimSpace(rule.Key))
				}
				rule.Value = stringValue
			}
			if sourceKey, exists := entry["source_key"]; exists {
				stringSourceKey, ok := sourceKey.(string)
				if !ok {
					return nil, false, fmt.Errorf("passthrough map source_key must be a string: %s", strings.TrimSpace(rule.Key))
				}
				rule.SourceKey = stringSourceKey
			}
			normalizedRule, err := normalizePassthroughRule(rule)
			if err != nil {
				return nil, false, err
			}
			if err := validatePassthroughRuleDuplicate(seenRulesByTarget, normalizedRule); err != nil {
				return nil, false, err
			}
			validated = append(validated, normalizedRule)
		}
		return validated, true, nil
	default:
		return nil, false, fmt.Errorf("invalid passthrough field rules")
	}
}

func normalizePassthroughRule(rule PassthroughFieldRule) (PassthroughFieldRule, error) {
	rule.Target = strings.TrimSpace(rule.Target)
	rule.Mode = strings.TrimSpace(rule.Mode)
	rule.Key = strings.TrimSpace(rule.Key)
	rule.SourceKey = strings.TrimSpace(rule.SourceKey)

	switch rule.Target {
	case "header":
		if rule.Key == "" {
			return PassthroughFieldRule{}, fmt.Errorf("passthrough header key is required")
		}
	case "body":
		if !passthroughBodyPathPattern.MatchString(rule.Key) {
			return PassthroughFieldRule{}, fmt.Errorf("invalid passthrough body path: %s", rule.Key)
		}
	default:
		return PassthroughFieldRule{}, fmt.Errorf("invalid passthrough field target: %s", rule.Target)
	}

	switch rule.Mode {
	case "forward":
		rule.Value = ""
		rule.SourceKey = ""
	case "inject":
		if strings.TrimSpace(rule.Value) == "" {
			return PassthroughFieldRule{}, fmt.Errorf("passthrough inject value cannot be blank: %s", rule.Key)
		}
		rule.SourceKey = ""
	case "map":
		if rule.SourceKey == "" {
			return PassthroughFieldRule{}, fmt.Errorf("passthrough map source_key is required: %s", rule.Key)
		}
		if rule.Target == "body" {
			if !passthroughBodyPathPattern.MatchString(rule.SourceKey) {
				return PassthroughFieldRule{}, fmt.Errorf("invalid passthrough body path: %s", rule.SourceKey)
			}
			if rule.SourceKey == rule.Key {
				return PassthroughFieldRule{}, fmt.Errorf("passthrough map source_key and key must be different: %s", rule.SourceKey)
			}
		} else if strings.EqualFold(rule.SourceKey, rule.Key) {
			return PassthroughFieldRule{}, fmt.Errorf("passthrough map source_key and key must be different: %s", strings.ToLower(rule.SourceKey))
		}
		rule.Value = ""
	default:
		return PassthroughFieldRule{}, fmt.Errorf("invalid passthrough field mode: %s", rule.Mode)
	}

	return rule, nil
}

func hasPassthroughConfigKeys(extra map[string]any) bool {
	if extra == nil {
		return false
	}
	_, hasEnabled := extra[accountPassthroughEnabledKey]
	_, hasRules := extra[accountPassthroughRulesKey]
	return hasEnabled || hasRules
}

func validatePassthroughRuleDuplicate(seenRulesByTarget map[string]map[string]PassthroughFieldRule, rule PassthroughFieldRule) error {
	seenKeys, ok := seenRulesByTarget[rule.Target]
	if !ok {
		seenKeys = map[string]PassthroughFieldRule{}
		seenRulesByTarget[rule.Target] = seenKeys
	}
	comparisonKey := rule.Key
	if rule.Target == "header" {
		comparisonKey = strings.ToLower(rule.Key)
	}
	if _, exists := seenKeys[comparisonKey]; exists {
		if rule.Target == "header" {
			return fmt.Errorf("duplicate passthrough header key: %s", comparisonKey)
		}
		return fmt.Errorf("duplicate passthrough body path: %s", comparisonKey)
	}
	if rule.Target == "body" {
		for existingKey := range seenKeys {
			if passthroughBodyPathHasPrefixConflict(existingKey, comparisonKey) {
				return fmt.Errorf("conflicting passthrough body path prefixes: %s, %s", existingKey, comparisonKey)
			}
		}
	}
	seenKeys[comparisonKey] = rule
	return nil
}

func passthroughBodyPathHasPrefixConflict(existing string, candidate string) bool {
	if existing == candidate {
		return false
	}
	return strings.HasPrefix(existing, candidate+".") || strings.HasPrefix(candidate, existing+".")
}

func clonePassthroughExtra(extra map[string]any) map[string]any {
	if extra == nil {
		return nil
	}
	cloned := make(map[string]any, len(extra))
	for k, v := range extra {
		cloned[k] = v
	}
	return cloned
}

func ApplyAccountPassthroughFields(
	account *Account,
	inbound http.Header,
	sourceBody []byte,
	targetBody []byte,
	outbound http.Header,
) ([]byte, error) {
	return applyAccountPassthroughFieldsWithContext(context.Background(), account, inbound, sourceBody, targetBody, outbound)
}

func applyAccountPassthroughFieldsWithContext(
	ctx context.Context,
	account *Account,
	inbound http.Header,
	sourceBody []byte,
	targetBody []byte,
	outbound http.Header,
) ([]byte, error) {
	// Fast-path: if passthrough is explicitly disabled, skip all rule parsing.
	// This ensures the toggle works as an emergency stop even when stored rules
	// are malformed.
	if account == nil || account.Type != AccountTypeAPIKey {
		return targetBody, nil
	}
	if rawEnabled, ok := account.Extra[accountPassthroughEnabledKey]; ok {
		if enabled, ok := rawEnabled.(bool); ok && !enabled {
			return targetBody, nil
		}
	}

	enabled, rules, err := AccountPassthroughFieldRules(account)
	if err != nil || !enabled || len(rules) == 0 {
		return targetBody, err
	}
	if outbound == nil {
		for _, rule := range rules {
			if rule.Target == "header" {
				return nil, fmt.Errorf("passthrough outbound headers are required for header rules")
			}
		}
		outbound = http.Header{}
	}
	baseOutbound := clonePassthroughHeader(outbound)

	for _, mode := range []string{"inject", "map", "forward"} {
		for _, rule := range rules {
			if rule.Mode != mode {
				continue
			}
			switch rule.Target {
			case "header":
				switch rule.Mode {
				case "inject":
					outbound.Set(rule.Key, rule.Value)
				case "map":
					if passthroughHeaderHasKey(inbound, rule.Key) || passthroughHeaderHasKey(baseOutbound, rule.Key) || passthroughHeaderHasKey(outbound, rule.Key) {
						continue
					}
					if values, ok := passthroughHeaderValues(inbound, rule.SourceKey); ok {
						setPassthroughHeaderValues(outbound, rule.Key, values)
					}
				case "forward":
					if values, ok := passthroughHeaderValues(inbound, rule.Key); ok {
						setPassthroughHeaderValues(outbound, rule.Key, values)
					}
				}
			case "body":
				var bodyValue any
				shouldApply := false
				switch rule.Mode {
				case "inject":
					bodyValue = rule.Value
					shouldApply = true
				case "map":
					bodyValue, shouldApply = lookupPassthroughBodyValue(sourceBody, rule.SourceKey)
					if !shouldApply {
						continue
					}
					if passthroughBodyPathExists(sourceBody, rule.Key) || passthroughBodyPathExists(targetBody, rule.Key) {
						continue
					}
					bodyValue = clonePassthroughJSONValue(bodyValue)
				case "forward":
					bodyValue, shouldApply = lookupPassthroughBodyValue(sourceBody, rule.Key)
				}
				if !shouldApply {
					continue
				}
				targetBody, err = setPassthroughBodyValue(ctx, account, targetBody, rule, bodyValue)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return targetBody, nil
}

func lookupPassthroughBodyValue(body []byte, path string) (any, bool) {
	if len(body) == 0 {
		return nil, false
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, false
	}
	value, ok := walkPassthroughBodyPath(payload, strings.Split(path, "."))
	return value, ok
}

func passthroughBodyPathExists(body []byte, path string) bool {
	_, ok := lookupPassthroughBodyValue(body, path)
	return ok
}

func walkPassthroughBodyPath(node any, parts []string) (any, bool) {
	if len(parts) == 0 {
		return node, true
	}
	obj, ok := node.(map[string]any)
	if !ok {
		return nil, false
	}
	next, exists := obj[parts[0]]
	if !exists {
		return nil, false
	}
	return walkPassthroughBodyPath(next, parts[1:])
}

func setPassthroughBodyValue(ctx context.Context, account *Account, targetBody []byte, rule PassthroughFieldRule, value any) ([]byte, error) {
	var payload any
	if len(targetBody) == 0 {
		payload = map[string]any{}
	} else if err := json.Unmarshal(targetBody, &payload); err != nil {
		return nil, err
	}
	obj, ok := payload.(map[string]any)
	if !ok {
		obj = map[string]any{}
	}
	parts := strings.Split(rule.Key, ".")
	current := obj
	for i := 0; i < len(parts)-1; i++ {
		part := parts[i]
		next, exists := current[part]
		if !exists || next == nil {
			child := map[string]any{}
			current[part] = child
			current = child
			continue
		}
		child, ok := next.(map[string]any)
		if !ok {
			conflictNode := strings.Join(parts[:i+1], ".")
			logger.FromContext(ctx).With(
				zap.String("component", "service.account_passthrough_fields"),
				zap.Int64("account_id", accountIDForPassthroughLog(account)),
				zap.String("target", rule.Target),
				zap.String("key", rule.Key),
				zap.String("conflict_node", conflictNode),
			).Warn("passthrough body path conflicts with non-object node")
			return nil, fmt.Errorf("invalid_request_error: passthrough body path conflicts with non-object node: %s", conflictNode)
		}
		current = child
	}
	current[parts[len(parts)-1]] = value
	return json.Marshal(obj)
}

func passthroughHeaderValues(header http.Header, key string) ([]string, bool) {
	if header == nil {
		return nil, false
	}
	for existingKey, values := range header {
		if strings.EqualFold(existingKey, key) {
			return append([]string(nil), values...), true
		}
	}
	return nil, false
}

func passthroughHeaderHasKey(header http.Header, key string) bool {
	_, ok := passthroughHeaderValues(header, key)
	return ok
}

func clonePassthroughHeader(header http.Header) http.Header {
	if header == nil {
		return nil
	}
	cloned := make(http.Header, len(header))
	for key, values := range header {
		cloned[key] = append([]string(nil), values...)
	}
	return cloned
}

func setPassthroughHeaderValues(header http.Header, key string, values []string) {
	if header == nil {
		return
	}
	header[http.CanonicalHeaderKey(key)] = append([]string(nil), values...)
}

func clonePassthroughJSONValue(value any) any {
	switch typed := value.(type) {
	case map[string]any:
		cloned := make(map[string]any, len(typed))
		for key, child := range typed {
			cloned[key] = clonePassthroughJSONValue(child)
		}
		return cloned
	case []any:
		cloned := make([]any, len(typed))
		for i, child := range typed {
			cloned[i] = clonePassthroughJSONValue(child)
		}
		return cloned
	default:
		return typed
	}
}

func accountIDForPassthroughLog(account *Account) int64 {
	if account == nil {
		return 0
	}
	return account.ID
}
