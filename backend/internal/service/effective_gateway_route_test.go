//go:build !unit

package service

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type effectiveRouteGroupRepo struct {
	GroupRepository
	groups map[int64]*Group
	err    error
}

func (r effectiveRouteGroupRepo) GetByIDLite(_ context.Context, id int64) (*Group, error) {
	if r.err != nil {
		return nil, r.err
	}
	group := r.groups[id]
	if group == nil {
		return nil, ErrGroupNotFound
	}
	return group, nil
}

type effectiveRouteCompositeRepo struct {
	CompositeModelRouteRepository
	err error
}

func (r effectiveRouteCompositeRepo) ListByGroup(context.Context, int64, bool) ([]CompositeModelRoute, error) {
	return nil, r.err
}

func requireEffectiveGatewayApplicationError(t *testing.T, err, target error, status int, reason, message string) {
	t.Helper()
	require.ErrorIs(t, err, target)
	gotStatus, body := infraerrors.ToHTTP(err)
	require.Equal(t, status, gotStatus)
	require.Equal(t, int32(status), body.Code)
	require.Equal(t, reason, body.Reason)
	require.Equal(t, message, body.Message)
}

type effectiveRouteSubscriptionRepo struct {
	UserSubscriptionRepository
	subscriptions map[[2]int64]*UserSubscription
}

type effectiveRouteNilSubscriptionRepo struct {
	UserSubscriptionRepository
}

func (effectiveRouteNilSubscriptionRepo) GetActiveByUserIDAndGroupID(context.Context, int64, int64) (*UserSubscription, error) {
	return nil, nil
}

func (r effectiveRouteSubscriptionRepo) GetActiveByUserIDAndGroupID(_ context.Context, userID, groupID int64) (*UserSubscription, error) {
	sub := r.subscriptions[[2]int64{userID, groupID}]
	if sub == nil {
		return nil, ErrSubscriptionNotFound
	}
	copy := *sub
	return &copy, nil
}

func TestEffectiveGatewayRouteResolverUsesFinalSubscriptionGroup(t *testing.T) {
	startID, finalID := int64(1), int64(2)
	override := 7
	start := &Group{ID: startID, Platform: PlatformAnthropic, Status: StatusActive, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
	final := &Group{ID: finalID, Platform: PlatformComposite, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	input := &APIKey{UserID: 42, GroupID: &startID, Group: start, User: &User{ID: 42, UserGroupRPMOverride: &override}}
	subscription := &UserSubscription{ID: 9, UserID: 42, GroupID: finalID, Status: SubscriptionStatusActive, ExpiresAt: time.Now().Add(time.Hour)}
	resolver := NewEffectiveGatewayRouteResolver(
		&APIKeyService{
			groupRepo:   effectiveRouteGroupRepo{groups: map[int64]*Group{finalID: final}},
			userSubRepo: effectiveRouteSubscriptionRepo{subscriptions: map[[2]int64]*UserSubscription{{42, finalID}: subscription}},
		},
		NewCompositeRouteResolver(compositeRouteRepoStub{routes: []CompositeModelRoute{{
			GroupID: finalID, PublicModel: "public-model", MatchType: CompositeRouteMatchExact,
			TargetPlatform: PlatformAnthropic, UpstreamModel: "claude-sonnet-4-6", Endpoint: CompositeRouteEndpointMessages, Enabled: true,
		}}}),
		&config.Config{},
	)

	route, err := resolver.Resolve(context.Background(), input, nil, start, "public-model", CompositeRouteEndpointMessages)

	require.NoError(t, err)
	require.NotSame(t, input, route.APIKey)
	require.NotSame(t, input.User, route.APIKey.User)
	require.Same(t, start, input.Group)
	require.Equal(t, startID, *input.GroupID)
	require.Same(t, &override, input.User.UserGroupRPMOverride)
	require.Nil(t, route.APIKey.User.UserGroupRPMOverride)
	require.Equal(t, finalID, *route.GroupID)
	require.Equal(t, finalID, *route.APIKey.GroupID)
	require.Same(t, final, route.Group)
	require.Same(t, final, route.APIKey.Group)
	require.NotNil(t, route.Subscription)
	require.Equal(t, subscription.ID, route.Subscription.ID)
	require.Equal(t, subscription.UserID, route.Subscription.UserID)
	require.Equal(t, subscription.GroupID, route.Subscription.GroupID)
	require.Equal(t, EffectiveGatewayBillingSubscription, route.BillingSource)
	require.Equal(t, PlatformAnthropic, route.Platform)
	require.NotNil(t, route.Decision)
	require.Equal(t, finalID, route.Decision.GroupID)
	require.Equal(t, "public-model", route.ClientModel)
	require.Equal(t, "claude-sonnet-4-6", route.RoutingModel)
	require.Equal(t, "claude-sonnet-4-6", route.UpstreamModel)
}

func TestEffectiveGatewayRouteResolverRejectsUnavailableFinalSubscription(t *testing.T) {
	startID, finalID := int64(1), int64(2)
	start := &Group{ID: startID, Status: StatusActive, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
	final := &Group{ID: finalID, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	resolver := NewEffectiveGatewayRouteResolver(
		&APIKeyService{
			groupRepo:   effectiveRouteGroupRepo{groups: map[int64]*Group{finalID: final}},
			userSubRepo: effectiveRouteSubscriptionRepo{},
		},
		nil,
		&config.Config{},
	)

	_, err := resolver.Resolve(context.Background(), &APIKey{UserID: 42, GroupID: &startID, Group: start, User: &User{ID: 42}}, nil, start, "claude-sonnet-4-6", CompositeRouteEndpointMessages)

	require.ErrorIs(t, err, ErrEffectiveGatewayRouteUnavailable)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)
}

func TestEffectiveGatewayRouteResolverRejectsNilFinalSubscription(t *testing.T) {
	groupID := int64(2)
	group := &Group{ID: groupID, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	resolver := NewEffectiveGatewayRouteResolver(
		&APIKeyService{userSubRepo: effectiveRouteNilSubscriptionRepo{}},
		nil,
		&config.Config{},
	)

	_, err := resolver.Resolve(context.Background(), &APIKey{UserID: 42, GroupID: &groupID, Group: group, User: &User{ID: 42}}, nil, group, "claude-sonnet-4-6", CompositeRouteEndpointMessages)

	require.ErrorIs(t, err, ErrEffectiveGatewayRouteUnavailable)
	require.ErrorIs(t, err, ErrSubscriptionNotFound)
}

func TestEffectiveGatewayRouteResolverRejectsDisallowedFinalStandardGroup(t *testing.T) {
	startID, finalID := int64(1), int64(2)
	start := &Group{ID: startID, Status: StatusActive, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
	final := &Group{ID: finalID, Status: StatusActive, IsExclusive: true}
	resolver := NewEffectiveGatewayRouteResolver(
		&APIKeyService{groupRepo: effectiveRouteGroupRepo{groups: map[int64]*Group{finalID: final}}},
		nil,
		&config.Config{},
	)

	_, err := resolver.Resolve(context.Background(), &APIKey{GroupID: &startID, Group: start, User: &User{ID: 42}}, nil, start, "claude-sonnet-4-6", CompositeRouteEndpointMessages)

	require.ErrorIs(t, err, ErrGroupNotAllowed)
}

func TestEffectiveGatewayRouteResolverSimpleModeSkipsSubscriptionLookup(t *testing.T) {
	groupID := int64(2)
	group := &Group{ID: groupID, Status: StatusActive, SubscriptionType: SubscriptionTypeSubscription}
	resolver := NewEffectiveGatewayRouteResolver(
		&APIKeyService{userSubRepo: effectiveRouteSubscriptionRepo{}},
		nil,
		&config.Config{RunMode: config.RunModeSimple},
	)

	route, err := resolver.Resolve(context.Background(), &APIKey{GroupID: &groupID, Group: group, User: &User{ID: 42}}, nil, group, "claude-sonnet-4-6", CompositeRouteEndpointMessages)

	require.NoError(t, err)
	require.Equal(t, EffectiveGatewayBillingSimpleSkip, route.BillingSource)
	require.Nil(t, route.Subscription)
}

func TestEffectiveGatewayRouteResolverPreservesNonCompositeIdentity(t *testing.T) {
	groupID := int64(2)
	group := &Group{ID: groupID, Platform: PlatformAnthropic, Status: StatusActive}
	input := &APIKey{GroupID: &groupID, Group: group, User: &User{ID: 42}}
	resolver := NewEffectiveGatewayRouteResolver(&APIKeyService{}, nil, &config.Config{})

	route, err := resolver.Resolve(context.Background(), input, nil, group, "claude-sonnet-4-6", CompositeRouteEndpointMessages)

	require.NoError(t, err)
	require.Same(t, input, route.APIKey)
	require.Same(t, input.User, route.APIKey.User)
	require.Equal(t, EffectiveGatewayBillingBalance, route.BillingSource)
	require.Nil(t, route.Decision)
	require.Equal(t, PlatformAnthropic, route.Platform)
	require.Equal(t, "claude-sonnet-4-6", route.RoutingModel)
	require.Equal(t, "claude-sonnet-4-6", route.UpstreamModel)
}

func TestEffectiveGatewayRouteResolverMapsGroupPolicyErrors(t *testing.T) {
	startID, finalID := int64(1), int64(2)
	sentinel := errors.New("group repository unavailable")
	tests := []struct {
		name       string
		groupRepo  GroupRepository
		want       error
		wantStatus int
		wantReason string
		wantMsg    string
		wantCause  error
	}{
		{
			name:       "missing final group",
			groupRepo:  effectiveRouteGroupRepo{},
			want:       ErrEffectiveGatewayGroupDeleted,
			wantStatus: http.StatusForbidden,
			wantReason: "GROUP_DELETED",
			wantMsg:    "API Key 所属分组已删除",
			wantCause:  ErrGroupNotFound,
		},
		{
			name:       "missing group repository",
			want:       ErrEffectiveGatewayGroupDeleted,
			wantStatus: http.StatusForbidden,
			wantReason: "GROUP_DELETED",
			wantMsg:    "API Key 所属分组已删除",
			wantCause:  ErrGroupNotFound,
		},
		{
			name:       "disabled final group",
			groupRepo:  effectiveRouteGroupRepo{groups: map[int64]*Group{finalID: {ID: finalID, Status: StatusDisabled}}},
			want:       ErrEffectiveGatewayGroupDisabled,
			wantStatus: http.StatusForbidden,
			wantReason: "GROUP_DISABLED",
			wantMsg:    "API Key 所属分组已停用",
		},
		{
			name:       "repository cause",
			groupRepo:  effectiveRouteGroupRepo{err: sentinel},
			want:       ErrEffectiveGatewayRouteUnavailable,
			wantStatus: http.StatusServiceUnavailable,
			wantReason: "NO_AVAILABLE_ACCOUNTS",
			wantMsg:    "No available accounts",
			wantCause:  sentinel,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			start := &Group{ID: startID, Status: StatusActive, ClaudeCodeOnly: true, FallbackGroupID: &finalID}
			resolver := NewEffectiveGatewayRouteResolver(&APIKeyService{groupRepo: tt.groupRepo}, nil, &config.Config{})

			_, err := resolver.Resolve(context.Background(), &APIKey{GroupID: &startID, Group: start, User: &User{ID: 42}}, nil, start, "claude-sonnet-4-6", CompositeRouteEndpointMessages)

			requireEffectiveGatewayApplicationError(t, err, tt.want, tt.wantStatus, tt.wantReason, tt.wantMsg)
			if tt.wantCause != nil {
				require.ErrorIs(t, err, tt.wantCause)
			}
		})
	}
}

func TestEffectiveGatewayRouteResolverMapsCompositePolicyErrors(t *testing.T) {
	groupID := int64(2)
	group := &Group{ID: groupID, Platform: PlatformComposite, Status: StatusActive}
	input := &APIKey{GroupID: &groupID, Group: group, User: &User{ID: 42}}
	sentinel := errors.New("composite repository unavailable")
	tests := []struct {
		name      string
		repo      CompositeModelRouteRepository
		model     string
		wantCause error
	}{
		{name: "resolver cause", repo: effectiveRouteCompositeRepo{err: sentinel}, model: "unknown-model", wantCause: sentinel},
		{name: "unmatched decision", repo: effectiveRouteCompositeRepo{}, model: "unknown-model"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewEffectiveGatewayRouteResolver(&APIKeyService{}, NewCompositeRouteResolver(tt.repo), &config.Config{})

			_, err := resolver.Resolve(context.Background(), input, nil, group, tt.model, CompositeRouteEndpointMessages)

			requireEffectiveGatewayApplicationError(t, err, ErrEffectiveGatewayRouteUnavailable, http.StatusServiceUnavailable, "NO_AVAILABLE_ACCOUNTS", "No available accounts")
			if tt.wantCause != nil {
				require.ErrorIs(t, err, tt.wantCause)
			}
		})
	}
}

func TestEffectiveGatewayRouteWithChannelMappingPreservesOrUpdatesRoutingIdentity(t *testing.T) {
	route := EffectiveGatewayRoute{RoutingModel: "claude-sonnet-4-6", UpstreamModel: "stale-model"}

	unmapped := route.WithChannelMapping(ChannelMappingResult{Mapped: false, MappedModel: "ignored-model"})
	mapped := route.WithChannelMapping(ChannelMappingResult{Mapped: true, MappedModel: "mapped-model"})
	fallback := EffectiveGatewayRoute{}.WithChannelMapping(ChannelMappingResult{Mapped: false, MappedModel: "fallback-model"})

	require.Equal(t, "claude-sonnet-4-6", unmapped.RoutingModel)
	require.Equal(t, unmapped.RoutingModel, unmapped.UpstreamModel)
	require.Equal(t, "mapped-model", mapped.RoutingModel)
	require.Equal(t, mapped.RoutingModel, mapped.UpstreamModel)
	require.Equal(t, "fallback-model", fallback.RoutingModel)
	require.Equal(t, fallback.RoutingModel, fallback.UpstreamModel)
}

func TestEffectiveGatewayRouteWithUpstreamModelIgnoresEmptyValue(t *testing.T) {
	route := EffectiveGatewayRoute{UpstreamModel: "claude-sonnet-4-6"}

	require.Equal(t, "claude-sonnet-4-6", route.WithUpstreamModel("").UpstreamModel)
	require.Equal(t, "actual-upstream-model", route.WithUpstreamModel("actual-upstream-model").UpstreamModel)
}

func TestEffectiveGatewayRouteContextRoundTrip(t *testing.T) {
	route := EffectiveGatewayRoute{
		Endpoint:      CompositeRouteEndpointMessages,
		ClientModel:   "public-model",
		RoutingModel:  "mapped-model",
		UpstreamModel: "actual-upstream-model",
		Channel:       ChannelMappingResult{Mapped: true, MappedModel: "mapped-model", ChannelID: 7},
	}

	got, ok := EffectiveGatewayRouteFromContext(WithEffectiveGatewayRoute(context.Background(), route))
	_, emptyOK := EffectiveGatewayRouteFromContext(context.Background())

	require.True(t, ok)
	require.Equal(t, route, got)
	require.False(t, emptyOK)
}
