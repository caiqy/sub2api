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
	reservedPassthroughHeaders = map[string]struct{}{
		"authorization":     {},
		"cookie":            {},
		"x-goog-api-key":    {},
		"x-api-key":         {},
		"api-key":           {},
		"host":              {},
		"content-length":    {},
		"transfer-encoding": {},
		"connection":        {},
	}
	reservedPassthroughBodyPaths = map[string]struct{}{
		"model":  {},
		"stream": {},
	}
)

type PassthroughFieldRule struct {
	Target string `json:"target"`
	Mode   string `json:"mode"`
	Key    string `json:"key"`
	Value  string `json:"value,omitempty"`
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
		seenRulesByTarget := map[string]map[string]struct{}{}
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
		seenRulesByTarget := map[string]map[string]struct{}{}
		for _, item := range typed {
			entry, ok := item.(map[string]any)
			if !ok {
				return nil, false, fmt.Errorf("invalid passthrough field rule")
			}
			rule := PassthroughFieldRule{}
			if v, ok := entry["target"].(string); ok {
				rule.Target = v
			}
			if v, ok := entry["mode"].(string); ok {
				rule.Mode = v
			}
			if v, ok := entry["key"].(string); ok {
				rule.Key = v
			}
			if value, exists := entry["value"]; exists {
				stringValue, ok := value.(string)
				if !ok {
					return nil, false, fmt.Errorf("passthrough inject value must be a string: %s", strings.TrimSpace(rule.Key))
				}
				rule.Value = stringValue
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

	switch rule.Target {
	case "header":
		if rule.Key == "" {
			return PassthroughFieldRule{}, fmt.Errorf("passthrough header key is required")
		}
		lowerKey := strings.ToLower(rule.Key)
		if _, reserved := reservedPassthroughHeaders[lowerKey]; reserved {
			return PassthroughFieldRule{}, fmt.Errorf("reserved passthrough header key: %s", lowerKey)
		}
	case "body":
		if !passthroughBodyPathPattern.MatchString(rule.Key) {
			return PassthroughFieldRule{}, fmt.Errorf("invalid passthrough body path: %s", rule.Key)
		}
		if _, reserved := reservedPassthroughBodyPaths[rule.Key]; reserved {
			return PassthroughFieldRule{}, fmt.Errorf("reserved passthrough body path: %s", rule.Key)
		}
	default:
		return PassthroughFieldRule{}, fmt.Errorf("invalid passthrough field target: %s", rule.Target)
	}

	switch rule.Mode {
	case "forward":
		rule.Value = ""
	case "inject":
		if strings.TrimSpace(rule.Value) == "" {
			return PassthroughFieldRule{}, fmt.Errorf("passthrough inject value cannot be blank: %s", rule.Key)
		}
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

func validatePassthroughRuleDuplicate(seenRulesByTarget map[string]map[string]struct{}, rule PassthroughFieldRule) error {
	seenKeys, ok := seenRulesByTarget[rule.Target]
	if !ok {
		seenKeys = map[string]struct{}{}
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
	seenKeys[comparisonKey] = struct{}{}
	return nil
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
	enabled, rules, err := AccountPassthroughFieldRules(account)
	if err != nil || !enabled || len(rules) == 0 {
		return targetBody, err
	}
	if outbound == nil {
		outbound = http.Header{}
	}

	for _, mode := range []string{"inject", "forward"} {
		for _, rule := range rules {
			if rule.Mode != mode {
				continue
			}
			switch rule.Target {
			case "header":
				switch rule.Mode {
				case "inject":
					outbound.Set(rule.Key, rule.Value)
				case "forward":
					if value := strings.TrimSpace(inbound.Get(rule.Key)); value != "" {
						if _, reserved := reservedPassthroughHeaders[strings.ToLower(rule.Key)]; !reserved {
							outbound.Set(rule.Key, value)
						}
					}
				}
			case "body":
				var bodyValue any
				shouldApply := false
				switch rule.Mode {
				case "inject":
					bodyValue = rule.Value
					shouldApply = true
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

func accountIDForPassthroughLog(account *Account) int64 {
	if account == nil {
		return 0
	}
	return account.ID
}
