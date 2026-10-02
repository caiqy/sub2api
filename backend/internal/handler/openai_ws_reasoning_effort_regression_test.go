package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	coderws "github.com/coder/websocket"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
)

func newOpenAIWSReasoningRegressionCache() *concurrencyCacheMock {
	return &concurrencyCacheMock{
		acquireUserSlotFn: func(context.Context, int64, int, string) (bool, error) {
			return true, nil
		},
		acquireUserGroupSlotFn: func(context.Context, int64, int64, int, string) (bool, error) {
			return true, nil
		},
		acquireAccountSlotFn: func(context.Context, int64, int, string) (bool, error) {
			return true, nil
		},
	}
}

func readOpenAIWSPassthroughCompletedTurn(t *testing.T, env *openAIWSRegressionEnv, conn *coderws.Conn) {
	t.Helper()
	created := env.readMessage(t, conn)
	require.Equal(t, "response.created", gjson.GetBytes(created, "type").String())
	completed := env.readMessage(t, conn)
	require.Equal(t, "response.completed", gjson.GetBytes(completed, "type").String())
}

func TestOpenAIResponsesWebSocketPassthroughBareFollowupUsesSessionModelForReasoningPolicy(t *testing.T) {
	for _, tc := range []struct {
		name              string
		maxEffort         string
		overLimit         string
		mappingFrom       string
		mappingTo         string
		firstEffort       string
		followupEffort    string
		wantFirstEffort   string
		wantFollowupError bool
	}{
		{
			name:            "model mapping",
			mappingFrom:     "max",
			mappingTo:       "high",
			firstEffort:     "max",
			followupEffort:  "max",
			wantFirstEffort: "high",
		},
		{
			name:              "mapped value is denied",
			maxEffort:         "high",
			overLimit:         service.ReasoningEffortOverLimitDeny,
			mappingFrom:       "medium",
			mappingTo:         "xhigh",
			firstEffort:       "low",
			followupEffort:    "medium",
			wantFirstEffort:   "low",
			wantFollowupError: true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
				Passthrough:             true,
				CaptureUpstreamMessages: true,
			})
			defer env.Close()
			env.apiKey.Group = &service.Group{
				ID:                          2,
				Platform:                    service.PlatformOpenAI,
				Status:                      service.StatusActive,
				Hydrated:                    true,
				MaxReasoningEffort:          tc.maxEffort,
				MaxReasoningEffortOverLimit: tc.overLimit,
				ReasoningEffortMappings: []service.ReasoningEffortMapping{{
					From:  tc.mappingFrom,
					To:    tc.mappingTo,
					Model: "gpt-5.4-max",
				}},
			}

			client := env.dial(t)
			defer func() { _ = client.CloseNow() }()
			env.writeMessage(t, client, fmt.Sprintf(`{"type":"response.create","model":"gpt-5.4-max","reasoning":{"effort":"%s"},"stream":false}`, tc.firstEffort))
			readOpenAIWSPassthroughCompletedTurn(t, env, client)
			firstUpstream := <-env.upstreamMessages
			require.Equal(t, "gpt-5.4-max", gjson.GetBytes(firstUpstream, "model").String())
			require.Equal(t, tc.wantFirstEffort, gjson.GetBytes(firstUpstream, "reasoning.effort").String())

			env.writeMessage(t, client, fmt.Sprintf(`{"type":"response.create","reasoning":{"effort":"%s"},"stream":false}`, tc.followupEffort))
			if tc.wantFollowupError {
				reason := env.readCloseError(t, client, coderws.StatusPolicyViolation)
				require.Contains(t, reason, "xhigh")
				env.waitRequestDone(t)
				return
			}

			readOpenAIWSPassthroughCompletedTurn(t, env, client)
			followupUpstream := <-env.upstreamMessages
			require.Equal(t, "gpt-5.4-max", gjson.GetBytes(followupUpstream, "model").String())
			require.Equal(t, "high", gjson.GetBytes(followupUpstream, "reasoning.effort").String())
			require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
			env.waitRequestDone(t)
		})
	}
}

func TestOpenAIResponsesWebSocketCompositeRewritePreservesRequestedReasoningEffortInUsage(t *testing.T) {
	for _, tc := range []struct {
		name          string
		upstreamError bool
		closeCode     coderws.StatusCode
	}{
		{name: "success"},
		{name: "failure", upstreamError: true, closeCode: coderws.StatusInternalError},
	} {
		t.Run(tc.name, func(t *testing.T) {
			usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
			resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{{
				GroupID:        2,
				PublicModel:    "gpt-5.4-max",
				MatchType:      service.CompositeRouteMatchExact,
				TargetPlatform: service.PlatformOpenAI,
				UpstreamModel:  "gpt-5.4",
				Endpoint:       service.CompositeRouteEndpointResponses,
				Enabled:        true,
			}}})
			env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
				Passthrough:                 true,
				UpstreamError:               tc.upstreamError,
				UpstreamErrorCode:           "server_error",
				CaptureFirstUpstreamMessage: true,
				CompositeResolver:           resolver,
				UsageLogRepo:                usageRepo,
			})
			defer env.Close()
			env.apiKey.Group = &service.Group{
				ID:       2,
				Platform: service.PlatformComposite,
				Status:   service.StatusActive,
				Hydrated: true,
			}

			client := env.dial(t)
			defer func() { _ = client.CloseNow() }()
			env.writeMessage(t, client, `{"type":"response.create","model":"gpt-5.4-max","stream":false}`)
			if tc.upstreamError {
				env.readCloseError(t, client, tc.closeCode)
			} else {
				readOpenAIWSPassthroughCompletedTurn(t, env, client)
				require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
			}
			env.waitRequestDone(t)

			var log *service.UsageLog
			select {
			case log = <-usageRepo.created:
			case <-time.After(5 * time.Second):
				t.Fatal("composite websocket usage was not recorded")
			}
			require.NotNil(t, log)
			require.NotNil(t, log.RequestedReasoningEffort)
			require.Equal(t, "max", *log.RequestedReasoningEffort)
			select {
			case payload := <-env.firstUpstreamMessage:
				require.Equal(t, "gpt-5.4", gjson.GetBytes(payload, "model").String())
			case <-time.After(5 * time.Second):
				t.Fatal("composite websocket did not reach upstream")
			}
		})
	}
}

func TestOpenAIResponsesWebSocketCompositeRoutesSessionUpdateModel(t *testing.T) {
	usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 2)}
	resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{
		{
			GroupID: 2, PublicModel: "public-first", MatchType: service.CompositeRouteMatchExact,
			TargetPlatform: service.PlatformOpenAI, UpstreamModel: "upstream-first",
			Endpoint: service.CompositeRouteEndpointResponses, Enabled: true,
		},
		{
			GroupID: 2, PublicModel: "public-next", MatchType: service.CompositeRouteMatchExact,
			TargetPlatform: service.PlatformOpenAI, UpstreamModel: "upstream-next",
			Endpoint: service.CompositeRouteEndpointResponses, Enabled: true,
		},
	}})
	env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
		Passthrough: true, CaptureUpstreamMessages: true, CompositeResolver: resolver, UsageLogRepo: usageRepo,
		AccountModelMapping: map[string]string{"upstream-first": "account-first", "upstream-next": "account-next"},
	})
	defer env.Close()
	env.apiKey.Group = &service.Group{ID: 2, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true}

	client := env.dial(t)
	defer func() { _ = client.CloseNow() }()
	env.writeMessage(t, client, `{"type":"response.create","model":"public-first","stream":false}`)
	require.Equal(t, "account-first", gjson.GetBytes(<-env.upstreamMessages, "model").String())
	readOpenAIWSPassthroughCompletedTurn(t, env, client)

	env.writeMessage(t, client, `{"type":"session.update","session":{"model":"public-next"}}`)
	require.Equal(t, "account-next", gjson.GetBytes(<-env.upstreamMessages, "session.model").String())
	require.Equal(t, "public-next", gjson.GetBytes(env.readMessage(t, client), "session.model").String())

	env.writeMessage(t, client, `{"type":"response.create","stream":false}`)
	require.False(t, gjson.GetBytes(<-env.upstreamMessages, "model").Exists())
	readOpenAIWSPassthroughCompletedTurn(t, env, client)
	require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
	env.waitRequestDone(t)
	firstLog := <-usageRepo.created
	secondLog := <-usageRepo.created
	require.NotNil(t, firstLog.UpstreamModel)
	require.Equal(t, "account-first", *firstLog.UpstreamModel)
	require.NotNil(t, secondLog.UpstreamModel)
	require.Equal(t, "account-next", *secondLog.UpstreamModel)
}

func TestOpenAIResponsesWebSocketCompositePublicAllowlistBeforeRoute(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, rejectedFrame := range []struct{ name, body string }{
			{"plain", `{"type":"response.create","model":"blocked-public","stream":false}`},
			{"middle_duplicate", `{"type":"response.create","model":"public-first","model":"blocked-public","model":"public-first","stream":false,"input":[{"type":"reasoning","content":[{"type":"reasoning_text","text":"drop"}],"summary":[{"type":"summary_text","text":"keep"}]}]}`},
			{"case_variant", `{"type":"response.create","model":"public-first","Model":"blocked-public","stream":false}`},
		} {
			t.Run(fmt.Sprintf("passthrough=%t/%s", passthrough, rejectedFrame.name), func(t *testing.T) {
				resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{
					{GroupID: 2, PublicModel: "public-first", MatchType: service.CompositeRouteMatchExact, TargetPlatform: service.PlatformOpenAI, UpstreamModel: "gpt-5.4", Endpoint: service.CompositeRouteEndpointResponses, Enabled: true},
					{GroupID: 2, PublicModel: "blocked-public", MatchType: service.CompositeRouteMatchExact, TargetPlatform: service.PlatformOpenAI, UpstreamModel: "gpt-5.4", Endpoint: service.CompositeRouteEndpointResponses, Enabled: true},
				}})
				env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
					Passthrough: passthrough, CaptureUpstreamMessages: true, CompositeResolver: resolver,
				})
				defer env.Close()
				env.apiKey.Group = &service.Group{
					ID: 2, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true,
					ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{"public-first", "gpt-5.4"}},
				}
				client := env.dial(t)
				defer func() { _ = client.CloseNow() }()
				env.writeMessage(t, client, `{"type":"response.create","model":"public-first","stream":false}`)
				select {
				case payload := <-env.upstreamMessages:
					require.Equal(t, "gpt-5.4", gjson.GetBytes(payload, "model").String())
				case <-time.After(5 * time.Second):
					t.Fatal("allowed first turn did not reach local upstream")
				}
				if passthrough {
					readOpenAIWSPassthroughCompletedTurn(t, env, client)
				} else {
					require.Equal(t, "response.completed", gjson.GetBytes(env.readMessage(t, client), "type").String())
				}
				env.writeMessage(t, client, rejectedFrame.body)
				require.Contains(t, env.readCloseError(t, client, coderws.StatusPolicyViolation), "blocked-public")
				env.waitRequestDone(t)
				select {
				case payload := <-env.upstreamMessages:
					t.Fatalf("forbidden public model reached local upstream: %s", payload)
				default:
				}
			})
		}
	}
}

func TestOpenAIResponsesWebSocketCompositePublicIdentityAcrossMappedTurns(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, channelMapped := range []bool{false, true} {
			t.Run(fmt.Sprintf("passthrough=%t/channelMapped=%t", passthrough, channelMapped), func(t *testing.T) {
				usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 4)}
				var channelMapping map[string]string
				if channelMapped {
					channelMapping = map[string]string{"gpt-5.4": "gpt-5.4-channel"}
				}
				publicPrice, routeInputPrice, routeOutputPrice := 1e-6, 2e-6, 3e-6
				channelInputPrice, channelOutputPrice := 3e-6, 4e-6
				resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{
					{GroupID: 2, PublicModel: "public-first", MatchType: service.CompositeRouteMatchExact, TargetPlatform: service.PlatformOpenAI, UpstreamModel: "gpt-5.4", Endpoint: service.CompositeRouteEndpointResponses, Enabled: true},
					{GroupID: 2, PublicModel: "public-next", MatchType: service.CompositeRouteMatchExact, TargetPlatform: service.PlatformOpenAI, UpstreamModel: "gpt-5.4", Endpoint: service.CompositeRouteEndpointResponses, Enabled: true},
				}})
				env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
					Passthrough: passthrough, CaptureUpstreamMessages: true, CompositeResolver: resolver, UsageLogRepo: usageRepo,
					ChannelModelMapping: channelMapping,
					ChannelModelPricing: []service.ChannelModelPricing{
						{Platform: service.PlatformOpenAI, Models: []string{"public-first"}, BillingMode: service.BillingModeToken, InputPrice: &publicPrice, OutputPrice: &publicPrice},
						{Platform: service.PlatformOpenAI, Models: []string{"public-next"}, BillingMode: service.BillingModeToken, InputPrice: &publicPrice, OutputPrice: &publicPrice},
						{Platform: service.PlatformOpenAI, Models: []string{"gpt-5.4"}, BillingMode: service.BillingModeToken, InputPrice: &routeInputPrice, OutputPrice: &routeOutputPrice},
						{Platform: service.PlatformOpenAI, Models: []string{"gpt-5.4-channel"}, BillingMode: service.BillingModeToken, InputPrice: &channelInputPrice, OutputPrice: &channelOutputPrice},
					},
					AccountModelMapping: map[string]string{"gpt-5.4": "gpt-5.4-account", "gpt-5.4-channel": "gpt-5.4-account", "public-next": "gpt-5.4-account"},
				})
				defer env.Close()
				env.apiKey.Group = &service.Group{
					ID: 2, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true,
					ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{"public-first", "public-next"}},
				}
				client := env.dial(t)
				defer func() { _ = client.CloseNow() }()
				for _, publicModel := range []string{"public-first", "", "public-next", ""} {
					payload := `{"type":"response.create","stream":false}`
					if publicModel != "" {
						payload = fmt.Sprintf(`{"type":"response.create","model":%q,"stream":false}`, publicModel)
					}
					env.writeMessage(t, client, payload)
					select {
					case payload := <-env.upstreamMessages:
						require.Equal(t, "gpt-5.4-account", gjson.GetBytes(payload, "model").String())
					case <-time.After(5 * time.Second):
						t.Fatal("allowed public turn did not reach local upstream")
					}
					if passthrough {
						readOpenAIWSPassthroughCompletedTurn(t, env, client)
					} else {
						require.Equal(t, "response.completed", gjson.GetBytes(env.readMessage(t, client), "type").String())
					}
				}
				require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
				env.waitRequestDone(t)
				expectedModels := []string{"public-first", "public-first", "public-next", "public-next"}
				if passthrough {
					// An explicit response.create overrides one frame; session.update
					// changes the passthrough session model used by a bare frame.
					expectedModels[3] = "public-first"
				}
				for _, publicModel := range expectedModels {
					select {
					case entry := <-usageRepo.created:
						require.Equal(t, publicModel, entry.Model)
						require.Equal(t, publicModel, entry.RequestedModel)
						require.NotNil(t, entry.UpstreamModel)
						require.Equal(t, "gpt-5.4-account", *entry.UpstreamModel)
						require.NotNil(t, entry.ModelMappingChain)
						chain := publicModel + "→gpt-5.4"
						cost := routeInputPrice + routeOutputPrice
						if channelMapped {
							chain += "→gpt-5.4-channel"
							cost = channelInputPrice + channelOutputPrice
						}
						require.Equal(t, chain+"→gpt-5.4-account", *entry.ModelMappingChain)
						require.InDelta(t, cost, entry.TotalCost, 1e-12)
					case <-time.After(5 * time.Second):
						t.Fatal("usage identity snapshot was not recorded")
					}
				}
			})
		}
	}
}

func TestOpenAIResponsesWebSocketCompositeRejectsUnroutedSessionUpdateModel(t *testing.T) {
	resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{{
		GroupID: 2, PublicModel: "public-first", MatchType: service.CompositeRouteMatchExact,
		TargetPlatform: service.PlatformOpenAI, UpstreamModel: "upstream-first",
		Endpoint: service.CompositeRouteEndpointResponses, Enabled: true,
	}}})
	env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
		Passthrough: true, CaptureUpstreamMessages: true, CompositeResolver: resolver,
	})
	defer env.Close()
	env.apiKey.Group = &service.Group{ID: 2, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true}

	client := env.dial(t)
	defer func() { _ = client.CloseNow() }()
	env.writeMessage(t, client, `{"type":"response.create","model":"public-first","stream":false}`)
	<-env.upstreamMessages
	readOpenAIWSPassthroughCompletedTurn(t, env, client)

	env.writeMessage(t, client, `{"type":"session.update","session":{"model":"not-routed"}}`)
	env.readCloseError(t, client, coderws.StatusPolicyViolation)
	select {
	case payload := <-env.upstreamMessages:
		t.Fatalf("unrouted session.update reached upstream: %s", payload)
	case <-time.After(100 * time.Millisecond):
	}
	env.waitRequestDone(t)
}

func TestOpenAIResponsesWebSocketCompositeRequestedPricingKeepsPrivateBaseline(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		for _, explicitPublicPrice := range []bool{false, true} {
			t.Run(fmt.Sprintf("passthrough=%t/explicitPublicPrice=%t", passthrough, explicitPublicPrice), func(t *testing.T) {
				const publicModel = "gpt-5.2"
				usageRepo := &openAIChatCompletionsUsageLogRepoStub{created: make(chan *service.UsageLog, 1)}
				inputPrice, outputPrice, publicPrice := 2e-6, 3e-6, 1e-6
				prices := []service.ChannelModelPricing{{Platform: service.PlatformOpenAI, Models: []string{"gpt-5.4-account"}, BillingMode: service.BillingModeToken, InputPrice: &inputPrice, OutputPrice: &outputPrice}}
				if explicitPublicPrice {
					prices = append(prices, service.ChannelModelPricing{Platform: service.PlatformOpenAI, Models: []string{publicModel}, BillingMode: service.BillingModeToken, InputPrice: &publicPrice, OutputPrice: &publicPrice})
				}
				resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{{GroupID: 2, PublicModel: publicModel, MatchType: service.CompositeRouteMatchExact, TargetPlatform: service.PlatformOpenAI, UpstreamModel: "gpt-5.4", Endpoint: service.CompositeRouteEndpointResponses, Enabled: true}}})
				env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
					Passthrough: passthrough, CaptureUpstreamMessages: true, CompositeResolver: resolver, UsageLogRepo: usageRepo,
					AccountModelMapping: map[string]string{"gpt-5.4": "gpt-5.4-account"}, ChannelModelPricing: prices,
					ChannelBillingModelSource: service.BillingModelSourceRequested,
				})
				defer env.Close()
				env.apiKey.Group = &service.Group{ID: 2, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true, ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{publicModel}}}
				client := env.dial(t)
				defer func() { _ = client.CloseNow() }()
				env.writeMessage(t, client, `{"type":"response.create","model":"gpt-5.2","stream":false}`)
				require.Equal(t, "gpt-5.4-account", gjson.GetBytes(<-env.upstreamMessages, "model").String())
				if passthrough {
					readOpenAIWSPassthroughCompletedTurn(t, env, client)
				} else {
					require.Equal(t, "response.completed", gjson.GetBytes(env.readMessage(t, client), "type").String())
				}
				require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
				env.waitRequestDone(t)
				entry := <-usageRepo.created
				require.Equal(t, publicModel, entry.Model)
				require.Equal(t, publicModel, entry.RequestedModel)
				expected := inputPrice + outputPrice
				if explicitPublicPrice {
					expected = 2 * publicPrice
				}
				require.InDelta(t, expected, entry.TotalCost, 1e-12)
			})
		}
	}
}

func TestOpenAIResponsesWebSocketCompositeSessionUpdateRawModelCandidates(t *testing.T) {
	for _, frame := range []string{
		`{"type":"session.update","session":{"model":"public-first","Model":"blocked-public"}}`,
		`{"type":"session.update","model":"public-first","session":{"Model":"blocked-public"}}`,
		`{"type":"session.update","Session":{"Model":"blocked-public"}}`,
	} {
		t.Run(frame, func(t *testing.T) {
			resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{{GroupID: 2, PublicModel: "public-first", MatchType: service.CompositeRouteMatchExact, TargetPlatform: service.PlatformOpenAI, UpstreamModel: "gpt-5.4", Endpoint: service.CompositeRouteEndpointResponses, Enabled: true}}})
			env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{Passthrough: true, CaptureUpstreamMessages: true, CompositeResolver: resolver})
			defer env.Close()
			env.apiKey.Group = &service.Group{ID: 2, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true, ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{"public-first"}}}
			client := env.dial(t)
			defer func() { _ = client.CloseNow() }()
			env.writeMessage(t, client, `{"type":"response.create","model":"public-first","stream":false}`)
			<-env.upstreamMessages
			readOpenAIWSPassthroughCompletedTurn(t, env, client)
			env.writeMessage(t, client, frame)
			require.Contains(t, env.readCloseError(t, client, coderws.StatusPolicyViolation), "blocked-public")
			env.waitRequestDone(t)
			select {
			case payload := <-env.upstreamMessages:
				t.Fatalf("forbidden session model reached local upstream: %s", payload)
			default:
			}
		})
	}
}

func TestOpenAIResponsesWebSocketCompositeSessionUpdateRejectsAllowedAmbiguity(t *testing.T) {
	for _, frame := range []string{
		`{"type":"session.update","session":{"model":"public-first","Model":"public-first"}}`,
		`{"type":"session.update","session":{"model":"public-first","Model":"public-other"}}`,
		`{"type":"session.update","session":{"model":"public-first","Model":""}}`,
		`{"type":"session.update","session":{"model":"public-first","Model":null}}`,
		`{"type":"session.update","session":{"model":"public-first","Model":42}}`,
		`{"type":"session.update","session":{"Model":"public-first"}}`,
		`{"type":"session.update","model":"public-first","session":{"model":"public-first"}}`,
	} {
		t.Run(frame, func(t *testing.T) {
			resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{
				{GroupID: 2, PublicModel: "public-first", MatchType: service.CompositeRouteMatchExact, TargetPlatform: service.PlatformOpenAI, UpstreamModel: "gpt-5.4", Endpoint: service.CompositeRouteEndpointResponses, Enabled: true},
				{GroupID: 2, PublicModel: "public-other", MatchType: service.CompositeRouteMatchExact, TargetPlatform: service.PlatformGrok, UpstreamModel: "grok-4", Endpoint: service.CompositeRouteEndpointResponses, Enabled: true},
			}})
			env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{Passthrough: true, CaptureUpstreamMessages: true, CompositeResolver: resolver})
			defer env.Close()
			env.apiKey.Group = &service.Group{ID: 2, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true, ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{"public-first", "public-other"}}}
			client := env.dial(t)
			defer func() { _ = client.CloseNow() }()
			env.writeMessage(t, client, `{"type":"response.create","model":"public-first","stream":false}`)
			<-env.upstreamMessages
			readOpenAIWSPassthroughCompletedTurn(t, env, client)
			env.writeMessage(t, client, frame)
			require.Contains(t, env.readCloseError(t, client, coderws.StatusPolicyViolation), "ambiguous session model")
			env.waitRequestDone(t)
			select {
			case payload := <-env.upstreamMessages:
				t.Fatalf("ambiguous session model reached local upstream: %s", payload)
			default:
			}
		})
	}
}

func TestOpenAIResponsesWebSocketPublicAllowlistAcrossAccountMapping(t *testing.T) {
	for _, passthrough := range []bool{false, true} {
		t.Run(fmt.Sprintf("passthrough=%t", passthrough), func(t *testing.T) {
			env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
				Passthrough: passthrough, CaptureUpstreamMessages: true,
				AccountModelMapping: map[string]string{"public-first": "gpt-5.4-account", "public-next": "gpt-5.4-account"},
			})
			defer env.Close()
			env.apiKey.Group = &service.Group{
				ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true,
				ModelAllowlist: service.GroupModelAllowlist{Enabled: true, Models: []string{"public-first", "public-next"}},
			}
			client := env.dial(t)
			defer func() { _ = client.CloseNow() }()
			for _, model := range []string{"public-first", "public-next"} {
				env.writeMessage(t, client, fmt.Sprintf(`{"type":"response.create","model":%q,"stream":false}`, model))
				select {
				case payload := <-env.upstreamMessages:
					require.Equal(t, "gpt-5.4-account", gjson.GetBytes(payload, "model").String())
				case <-time.After(5 * time.Second):
					t.Fatal("allowed public model did not reach local upstream")
				}
				if passthrough {
					readOpenAIWSPassthroughCompletedTurn(t, env, client)
				} else {
					require.Equal(t, "response.completed", gjson.GetBytes(env.readMessage(t, client), "type").String())
				}
			}
			require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
			env.waitRequestDone(t)
		})
	}
}

func TestOpenAIResponsesWebSocketCompositeRejectsDuplicateSessionUpdateModel(t *testing.T) {
	resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{
		{
			GroupID: 2, PublicModel: "public-first", MatchType: service.CompositeRouteMatchExact,
			TargetPlatform: service.PlatformOpenAI, UpstreamModel: "upstream-first",
			Endpoint: service.CompositeRouteEndpointResponses, Enabled: true,
		},
		{
			GroupID: 2, PublicModel: "public-next", MatchType: service.CompositeRouteMatchExact,
			TargetPlatform: service.PlatformOpenAI, UpstreamModel: "upstream-next",
			Endpoint: service.CompositeRouteEndpointResponses, Enabled: true,
		},
	}})
	env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
		Passthrough: true, CaptureUpstreamMessages: true, CompositeResolver: resolver,
	})
	defer env.Close()
	env.apiKey.Group = &service.Group{ID: 2, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true}

	client := env.dial(t)
	defer func() { _ = client.CloseNow() }()
	env.writeMessage(t, client, `{"type":"response.create","model":"public-first","stream":false}`)
	<-env.upstreamMessages
	readOpenAIWSPassthroughCompletedTurn(t, env, client)

	env.writeMessage(t, client, `{"type":"session.update","session":{"model":"","model":"not-routed"}}`)
	reason := env.readCloseError(t, client, coderws.StatusPolicyViolation)
	require.Contains(t, reason, "duplicate JSON")
	select {
	case payload := <-env.upstreamMessages:
		t.Fatalf("duplicate session.update reached upstream: %s", payload)
	case <-time.After(100 * time.Millisecond):
	}
	env.waitRequestDone(t)
}

func TestOpenAIResponsesWebSocketCompositePassthroughMapsInitialFrame(t *testing.T) {
	resolver := service.NewCompositeRouteResolver(openAIWSCompositeRouteRepo{routes: []service.CompositeModelRoute{{
		GroupID: 2, PublicModel: "public-alias", MatchType: service.CompositeRouteMatchExact,
		TargetPlatform: service.PlatformOpenAI, UpstreamModel: "gpt-5",
		Endpoint: service.CompositeRouteEndpointResponses, Enabled: true,
	}}})
	env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
		Passthrough: true, CaptureUpstreamMessages: true, CompositeResolver: resolver,
		ChannelModelMapping: map[string]string{"gpt-5": "gpt-5-channel", "gpt-5-channel": "gpt-5-channel-twice"},
		AccountModelMapping: map[string]string{"gpt-5": "gpt-5-account", "gpt-5-channel": "gpt-5-account"},
	})
	defer env.Close()
	env.apiKey.Group = &service.Group{ID: 2, Platform: service.PlatformComposite, Status: service.StatusActive, Hydrated: true}

	client := env.dial(t)
	defer func() { _ = client.CloseNow() }()
	env.writeMessage(t, client, `{"type":"response.create","model":"public-alias","stream":false}`)
	select {
	case payload := <-env.upstreamMessages:
		require.Equal(t, "gpt-5-account", gjson.GetBytes(payload, "model").String())
	case <-time.After(5 * time.Second):
		t.Fatal("mapped initial frame did not reach upstream")
	}
	readOpenAIWSPassthroughCompletedTurn(t, env, client)
	env.writeMessage(t, client, `{"type":"response.create","model":"public-alias","previous_response_id":"resp_ingress_turn_1","stream":false}`)
	select {
	case payload := <-env.upstreamMessages:
		require.Equal(t, "gpt-5-account", gjson.GetBytes(payload, "model").String())
	case <-time.After(5 * time.Second):
		t.Fatal("mapped second frame did not reach upstream")
	}
	readOpenAIWSPassthroughCompletedTurn(t, env, client)
	require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
	env.waitRequestDone(t)
}

func TestOpenAIResponsesWebSocketPassthroughAppliesAccountOnlyModelMapping(t *testing.T) {
	env := newOpenAIWSRegressionEnv(t, newOpenAIWSReasoningRegressionCache(), openAIWSRegressionEnvOptions{
		Passthrough: true, CaptureUpstreamMessages: true,
		AccountModelMapping: map[string]string{"gpt-client": "gpt-account"},
	})
	defer env.Close()
	env.apiKey.Group = &service.Group{ID: 2, Platform: service.PlatformOpenAI, Status: service.StatusActive, Hydrated: true}

	client := env.dial(t)
	defer func() { _ = client.CloseNow() }()
	env.writeMessage(t, client, `{"type":"response.create","model":"gpt-client","stream":false}`)
	select {
	case payload := <-env.upstreamMessages:
		require.Equal(t, "gpt-account", gjson.GetBytes(payload, "model").String())
	case <-time.After(5 * time.Second):
		t.Fatal("account-mapped initial frame did not reach upstream")
	}
	readOpenAIWSPassthroughCompletedTurn(t, env, client)
	env.writeMessage(t, client, `{"type":"session.update","session":{"model":"gpt-client"}}`)
	require.Equal(t, "gpt-account", gjson.GetBytes(<-env.upstreamMessages, "session.model").String())
	require.Equal(t, "gpt-client", gjson.GetBytes(env.readMessage(t, client), "session.model").String())
	env.writeMessage(t, client, `{"type":"response.create","stream":false}`)
	require.False(t, gjson.GetBytes(<-env.upstreamMessages, "model").Exists())
	readOpenAIWSPassthroughCompletedTurn(t, env, client)
	require.NoError(t, client.Close(coderws.StatusNormalClosure, "done"))
	env.waitRequestDone(t)
}
