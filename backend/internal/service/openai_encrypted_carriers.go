package service

import "strings"

// encryptedCarrierDigests validates before rewriting: unknown encrypted payloads
// may be necessary context, so a partial cleanup must never enable a retry.
func encryptedCarrierDigests(item any) ([]string, bool) {
	m, ok := item.(map[string]any)
	if !ok {
		return nil, true
	}
	typ, _ := m["type"].(string)
	typ = strings.TrimSpace(typ)
	if (typ == "compaction" || typ == "compaction_summary") && containsEncryptedCarrier(m["content"]) {
		return nil, false
	}
	var digests []string
	if encrypted, exists := m["encrypted_content"]; exists {
		value, valid := encrypted.(string)
		if !openAIEncryptedLineageItemType(typ) || (!valid && encrypted != nil) {
			return nil, false
		}
		if value != "" {
			digests = append(digests, openAIEncryptedContentDigest(value))
		}
	}
	for key, value := range m {
		if key == "content" {
			if parts, ok := value.([]any); ok {
				for _, part := range parts {
					d, safe := encryptedCarrierDigests(part)
					if !safe {
						return nil, false
					}
					digests = append(digests, d...)
				}
				continue
			}
		}
		if containsEncryptedCarrier(value) {
			return nil, false
		}
	}
	// A message must retain meaningful content after optional reasoning parts go.
	if typ == "message" || (typ == "" && m["role"] != nil) {
		if parts, ok := m["content"].([]any); ok {
			retained := false
			hasOptionalParts := false
			for _, p := range parts {
				if pm, ok := p.(map[string]any); ok {
					pt, _ := pm["type"].(string)
					hasOptionalParts = hasOptionalParts || openAIEncryptedLineageItemType(strings.TrimSpace(pt))
					text, _ := pm["text"].(string)
					if (pt == "input_text" || pt == "output_text") && strings.TrimSpace(text) != "" {
						retained = true
					}
				}
			}
			if (len(digests) > 0 || hasOptionalParts) && !retained {
				return nil, false
			}
		}
	} else if typ != "reasoning" && len(digests) > 0 && !openAIEncryptedLineageItemType(typ) {
		return nil, false
	}
	return digests, true
}

func containsEncryptedCarrier(value any) bool {
	switch v := value.(type) {
	case map[string]any:
		if _, ok := v["encrypted_content"]; ok {
			return true
		}
		for _, child := range v {
			if containsEncryptedCarrier(child) {
				return true
			}
		}
	case []any:
		for _, child := range v {
			if containsEncryptedCarrier(child) {
				return true
			}
		}
	}
	return false
}

func sanitizeOpenAIEncryptedCarrier(item any, match func(string) bool) (any, bool, bool) {
	m, ok := item.(map[string]any)
	if !ok {
		return item, false, true
	}
	typ, _ := m["type"].(string)
	typ = strings.TrimSpace(typ)
	if !openAIEncryptedLineageItemType(typ) && typ != "message" && !(typ == "" && m["role"] != nil) {
		return item, false, true
	}
	changed := false
	if parts, ok := m["content"].([]any); ok {
		var next []any
		for i, p := range parts {
			n, c, k := sanitizeOpenAIEncryptedCarrier(p, match)
			if c {
				changed = true
				if next == nil {
					next = make([]any, 0, len(parts))
					next = append(next, parts[:i]...)
				}
			}
			if next != nil && k {
				next = append(next, n)
			}
		}
		if changed {
			m["content"] = next
		}
	}
	encrypted, exists := m["encrypted_content"]
	value, _ := encrypted.(string)
	if openAIEncryptedLineageItemType(typ) && exists && ((value != "" && match(openAIEncryptedContentDigest(value))) || (value == "" && match(""))) {
		if typ != "reasoning" {
			return nil, true, false
		}
		delete(m, "encrypted_content")
		changed = true
	}
	if typ == "reasoning" && (changed || match("")) {
		if v, has := m["content"]; has && v == nil {
			delete(m, "content")
			changed = true
		}
		if changed && len(m) == 1 {
			return nil, true, false
		}
	}
	return item, changed, true
}

func openAIEncryptedRecoverySummary(body map[string]any) (int, string) {
	count := 0
	items, ok := body["input"].([]any)
	if typed, typedOK := body["input"].([]map[string]any); typedOK {
		items = make([]any, len(typed))
		for i, item := range typed {
			items[i] = item
		}
		ok = true
	}
	if !ok {
		items = []any{body["input"]}
	}
	for _, item := range items {
		d, safe := encryptedCarrierDigests(item)
		if !safe {
			return count, "unsafe_carrier"
		}
		count += len(d)
	}
	if count == 0 {
		return 0, "no_safe_carrier"
	}
	return count, "safe_carriers"
}
