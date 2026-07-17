# Task 33: v0.1.159 能力回归复审

## 范围与边界

- 起始提交：`9d2d4a8e4`（`docs: record v0.1.159 merge decisions`）。
- 仅审查 v0.1.159 合并后的能力回归；未开始 Task 34 或 full gate，未合并 `upstream/main`、tag 后提交，未 push、release 或 deploy。
- `.comet/current-change.json` 保持未跟踪，未修改、未加入提交。

## 结论

未发现需要修复的能力回归，未修改业务或测试代码。

### trusted proxy/IP 一致性

- `GetSecurityClientIP` 以 `TrustForwardedIPForAPIKeyACL` 为唯一开关：关闭时使用 Gin `trusted_proxies` 解析链，开启时使用受支持的转发头。
- `SessionBindingContext(cfg)` 在全局路由链注入同一解析结果；会话哈希、session mismatch 审计和常规 audit 均优先复用该绑定。
- API-key ACL 使用同一 helper。缺少注入的单测/异常路径回退到历史 `trusted_proxies` 语义。

### alpha/search 与上游副作用

- OpenAI API-key 账号支持 `alpha_search` 调度，显式 `chat_completions` 能力同样可承接该端点；Grok 仍被排除。
- API-key 上游对 `/v1/alpha/search` 返回 404/405 时产生 failover，不写账号错误状态；OAuth 的 404 保持原样透传。
- 仅 2xx 返回 `WebSearchCalls: 1`，非 2xx 透传不计费。

### Grok、图片与前端

- Responses 入口在写入 tenant/model 隔离的 cache identity 后，对已知 Free OAuth 的合格 function tools 补全 `web_search`/`x_search`，并保持非合格工具、付费/未知/API-key 账号不变。
- 图片意图仍覆盖 endpoint、image model、tool、input 和 tool choice；映射模型、纯文本 data image、计费与 image-only model 的定向用例均通过。
- API-key `base_url` 链接仅接受经 `sanitizeUrl` 校验的 HTTP(S) URL，展示为 origin，并使用 `noopener noreferrer`。
- 三个 Stripe 使用点均懒加载无副作用的 `@stripe/stripe-js/pure`；Vite 将 Stripe 保持在 `vendor-stripe` 独立 chunk。

## 测试

brief 精确命令：

```text
go -C backend test ./internal/pkg/ip ./internal/server/middleware -count=1
PASS (ip package has no default-tag tests; middleware passed)

go -C backend test ./internal/service -run 'AlphaSearch|Scheduler|Grok|Image|Account' -count=1
PASS

pnpm --dir frontend exec vitest run src/views/admin/__tests__/AccountsView.sparkShadow.spec.ts src/views/user/__tests__/StripePaymentView.spec.ts src/views/user/__tests__/stripeLazyLoading.spec.ts
PASS (3 files, 13 tests)
```

补充定向命令均通过：

- `go -C backend test -v -tags unit ./internal/pkg/ip ./internal/server/middleware -run 'IP|SessionBinding|APIKey|Audit' -count=1`
- `go -C backend test -v -tags unit ./internal/service -run '^(TestOpenAIAlphaSearch|TestGrokFreeMessagesFunctionToolCacheRoute.*|TestResolveGrokCacheIdentity.*|TestApplyGrokCacheIdentity.*|TestAccountSupportsOpenAIEndpointCapability)$' -count=1`
- `go -C backend test -v -tags unit ./internal/service -run '^(TestForwardAlphaSearchAPIKeyEndpointNotFoundFailsOver|TestForwardAlphaSearchOAuthNotFoundPassesThrough|TestShouldApplyOpenAIAlphaSearchAccountErrorSideEffects|TestIsOpenAIAlphaSearchEndpointUnsupported|TestOpenAIGatewayService_Forward_MappedImageModelUsesImageGate|TestOpenAIGatewayService_Forward_TextDataImageDoesNotForceMapMarshal|TestOpenAIGatewayService_Forward_ImageToolBillingDoesNotForceFullDecode|TestOpenAIGatewayService_Forward_ImageToolWithImageOnlyModelIsNormalized)$' -count=1`

## 关注项

- 未运行 Task 34/full gate，按任务边界保留给后续阶段。
- 前端定向测试保留既有 Browserslist data 过期警告，无失败断言。
