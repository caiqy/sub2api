package securityaudit

import (
	"context"
	"errors"
	"net/http"
	"sync"
)

type LegacyEngine interface {
	Check(ctx context.Context, req Request) (*LegacyDecision, error)
}

// lazyLegacyEngine may call body only before CheckLazy returns and must not retain it.
type lazyLegacyEngine interface {
	CheckLazy(ctx context.Context, req Request, body func() []byte) (*LegacyDecision, error)
}

type PromptEngine interface {
	EffectiveMode() Mode
	Enqueue(ctx context.Context, req Request) error
	Evaluate(ctx context.Context, req Request) (*PromptDecision, error)
}

type Coordinator struct {
	legacy LegacyEngine
	prompt PromptEngine
}

func NewCoordinator(legacy LegacyEngine, prompt PromptEngine) *Coordinator {
	return &Coordinator{legacy: legacy, prompt: prompt}
}

func (c *Coordinator) Check(ctx context.Context, req Request) Decision {
	body := req.Body
	return c.CheckLazy(ctx, req, func() []byte { return body })
}

// CheckLazy evaluates body only during this call. Engines must not retain the
// provider; async requests are frozen before they are cloned for enqueue.
func (c *Coordinator) CheckLazy(ctx context.Context, req Request, body func() []byte) Decision {
	if c == nil {
		return allowDecision(nil, nil)
	}
	initialBody := req.Body
	var once sync.Once
	var frozenBody []byte
	bodyOnce := func() []byte {
		once.Do(func() {
			if body != nil {
				frozenBody = body()
			} else {
				frozenBody = initialBody
			}
		})
		return frozenBody
	}
	mode := ModeOff
	if c.prompt != nil {
		mode = c.prompt.EffectiveMode()
	}
	switch mode {
	case ModeAsync:
		// Enqueue is deliberately best-effort. The implementation owns a bounded
		// context and copies request memory before it can outlive the Handler.
		req.Body = bodyOnce()
		_ = c.prompt.Enqueue(ctx, req.Clone())
		legacy, _ := c.checkLegacyLazy(ctx, req, bodyOnce)
		return prioritize(legacy, nil)
	case ModeBlocking:
		return c.checkBlockingLazy(ctx, req, bodyOnce)
	default:
		legacy, _ := c.checkLegacyLazy(ctx, req, bodyOnce)
		return prioritize(legacy, nil)
	}
}

func (c *Coordinator) checkBlockingLazy(ctx context.Context, req Request, body func() []byte) Decision {
	var wg sync.WaitGroup
	wg.Add(2)
	var legacy *LegacyDecision
	var prompt *PromptDecision
	go func(req Request) {
		defer wg.Done()
		legacy, _ = c.checkLegacyLazy(ctx, req, body)
	}(req)
	go func(req Request) {
		defer wg.Done()
		if c.prompt == nil {
			prompt = unavailablePromptDecision(ErrorCodeUnavailable)
			return
		}
		req.Body = body()
		result, err := c.prompt.Evaluate(ctx, req.Clone())
		if err != nil {
			var guardErr *GuardError
			if errors.As(err, &guardErr) && guardErr.Code == ErrorCodeInvalidResponse {
				prompt = unavailablePromptDecision(ErrorCodeInvalidResponse)
				return
			}
			prompt = unavailablePromptDecision(ErrorCodeUnavailable)
			return
		}
		if result == nil {
			prompt = unavailablePromptDecision(ErrorCodeUnavailable)
			return
		}
		prompt = result
	}(req)
	wg.Wait()
	return prioritize(legacy, prompt)
}

func (c *Coordinator) checkLegacyLazy(ctx context.Context, req Request, body func() []byte) (*LegacyDecision, error) {
	if c.legacy == nil {
		return nil, nil
	}
	if lazy, ok := c.legacy.(lazyLegacyEngine); ok {
		return lazy.CheckLazy(ctx, req, body)
	}
	req.Body = body()
	return c.legacy.Check(ctx, req)
}

func prioritize(legacy *LegacyDecision, prompt *PromptDecision) Decision {
	if legacy != nil && legacy.Blocked {
		status := legacy.StatusCode
		if status < 400 || status > 599 {
			status = http.StatusForbidden
		}
		code := legacy.ErrorCode
		if code == "" {
			code = "content_policy_violation"
		}
		return Decision{
			Kind: DecisionBlock, HTTPStatus: status, ErrorCode: code, ClientMessage: legacy.Message,
			Legacy: legacy, Prompt: prompt, AllowNextStage: false,
		}
	}
	if prompt == nil {
		return allowDecision(legacy, nil)
	}
	switch prompt.Kind {
	case DecisionBlock:
		return Decision{Kind: DecisionBlock, HTTPStatus: http.StatusForbidden, ErrorCode: ErrorCodeBlocked,
			ClientMessage: "提示词安全审计拒绝了该请求，请调整输入后重试", Legacy: legacy, Prompt: prompt}
	case DecisionInvalid:
		return Decision{Kind: DecisionInvalid, HTTPStatus: http.StatusServiceUnavailable, ErrorCode: ErrorCodeInvalidResponse,
			ClientMessage: "提示词安全审计暂时不可用，请稍后重试", Legacy: legacy, Prompt: prompt}
	case DecisionUnavailable:
		return Decision{Kind: DecisionUnavailable, HTTPStatus: http.StatusServiceUnavailable, ErrorCode: ErrorCodeUnavailable,
			ClientMessage: "提示词安全审计暂时不可用，请稍后重试", Legacy: legacy, Prompt: prompt}
	case DecisionFlag:
		return Decision{Kind: DecisionFlag, HTTPStatus: http.StatusOK, Legacy: legacy, Prompt: prompt, AllowNextStage: true}
	default:
		return allowDecision(legacy, prompt)
	}
}

func allowDecision(legacy *LegacyDecision, prompt *PromptDecision) Decision {
	return Decision{Kind: DecisionAllow, HTTPStatus: http.StatusOK, Legacy: legacy, Prompt: prompt, AllowNextStage: true}
}

func unavailablePromptDecision(code string) *PromptDecision {
	kind := DecisionUnavailable
	if code == ErrorCodeInvalidResponse {
		kind = DecisionInvalid
	}
	return &PromptDecision{Kind: kind, ErrorCode: code, AllowNextStage: false}
}
