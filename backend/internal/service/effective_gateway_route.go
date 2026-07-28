package service

import (
	"context"
	"errors"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type EffectiveGatewayBillingSource string

const (
	EffectiveGatewayBillingSimpleSkip   EffectiveGatewayBillingSource = "simple-skip"
	EffectiveGatewayBillingBalance      EffectiveGatewayBillingSource = "balance"
	EffectiveGatewayBillingSubscription EffectiveGatewayBillingSource = "subscription"
)

var (
	ErrEffectiveGatewayGroupDeleted     = infraerrors.Forbidden("GROUP_DELETED", "API Key 所属分组已删除")
	ErrEffectiveGatewayGroupDisabled    = infraerrors.Forbidden("GROUP_DISABLED", "API Key 所属分组已停用")
	ErrEffectiveGatewayRouteUnavailable = infraerrors.ServiceUnavailable("NO_AVAILABLE_ACCOUNTS", "No available accounts")
)

type EffectiveGatewayRoute struct {
	APIKey        *APIKey
	Group         *Group
	GroupID       *int64
	Subscription  *UserSubscription
	BillingSource EffectiveGatewayBillingSource
	Endpoint      string
	ClientModel   string
	RoutingModel  string
	UpstreamModel string
	Platform      string
	Decision      *CompositeRouteDecision
	Channel       ChannelMappingResult
}

type EffectiveGatewayRouteResolver struct {
	apiKeyService     *APIKeyService
	compositeResolver *CompositeRouteResolver
	cfg               *config.Config
}

func NewEffectiveGatewayRouteResolver(apiKeys *APIKeyService, composite *CompositeRouteResolver, cfg *config.Config) *EffectiveGatewayRouteResolver {
	return &EffectiveGatewayRouteResolver{apiKeyService: apiKeys, compositeResolver: composite, cfg: cfg}
}

func (r *EffectiveGatewayRouteResolver) Resolve(
	ctx context.Context,
	apiKey *APIKey,
	currentSubscription *UserSubscription,
	startGroup *Group,
	clientModel string,
	endpoint string,
) (EffectiveGatewayRoute, error) {
	group := startGroup
	if group == nil && apiKey != nil {
		group = apiKey.Group
	}
	if group != nil {
		resolved, err := r.apiKeyService.ResolveEffectiveGatewayGroup(ctx, group)
		if err != nil {
			if errors.Is(err, ErrGroupNotFound) {
				return EffectiveGatewayRoute{}, ErrEffectiveGatewayGroupDeleted.WithCause(err)
			}
			return EffectiveGatewayRoute{}, ErrEffectiveGatewayRouteUnavailable.WithCause(err)
		}
		group = resolved
		if !group.IsActive() {
			return EffectiveGatewayRoute{}, ErrEffectiveGatewayGroupDisabled
		}
		if apiKey == nil || apiKey.User == nil || (!group.IsSubscriptionType() && !apiKey.User.CanBindGroup(group.ID, group.IsExclusive)) {
			return EffectiveGatewayRoute{}, ErrGroupNotAllowed
		}
	}

	route := EffectiveGatewayRoute{
		APIKey:        apiKey,
		Group:         group,
		Endpoint:      endpoint,
		ClientModel:   clientModel,
		RoutingModel:  clientModel,
		UpstreamModel: clientModel,
	}
	if group != nil {
		groupID := group.ID
		route.GroupID = &groupID
		route.Platform = group.Platform
		if apiKey != nil && (apiKey.GroupID == nil || *apiKey.GroupID != group.ID) {
			apiKeyCopy := *apiKey
			apiKeyCopy.GroupID = &groupID
			apiKeyCopy.Group = group
			userCopy := *apiKey.User
			userCopy.UserGroupRPMOverride = nil
			apiKeyCopy.User = &userCopy
			route.APIKey = &apiKeyCopy
		}
	} else {
		route.Platform = PlatformAnthropic
	}

	if r.cfg != nil && r.cfg.RunMode == config.RunModeSimple {
		route.BillingSource = EffectiveGatewayBillingSimpleSkip
	} else if group != nil && group.IsSubscriptionType() {
		subscription := currentSubscription
		if subscription == nil || subscription.UserID != route.APIKey.UserID || subscription.GroupID != group.ID {
			if r.apiKeyService == nil || r.apiKeyService.userSubRepo == nil {
				return EffectiveGatewayRoute{}, ErrEffectiveGatewayRouteUnavailable.WithCause(ErrSubscriptionNotFound)
			}
			var err error
			subscription, err = r.apiKeyService.userSubRepo.GetActiveByUserIDAndGroupID(ctx, route.APIKey.UserID, group.ID)
			if err != nil {
				return EffectiveGatewayRoute{}, ErrEffectiveGatewayRouteUnavailable.WithCause(err)
			}
		}
		if subscription == nil {
			return EffectiveGatewayRoute{}, ErrEffectiveGatewayRouteUnavailable.WithCause(ErrSubscriptionNotFound)
		}
		route.Subscription = subscription
		route.BillingSource = EffectiveGatewayBillingSubscription
	} else {
		route.BillingSource = EffectiveGatewayBillingBalance
	}

	if group != nil && group.Platform == PlatformComposite {
		decision, err := r.compositeResolver.Resolve(ctx, group.ID, clientModel, endpoint)
		if err != nil {
			return EffectiveGatewayRoute{}, ErrEffectiveGatewayRouteUnavailable.WithCause(err)
		}
		if !decision.Matched {
			return EffectiveGatewayRoute{}, ErrEffectiveGatewayRouteUnavailable
		}
		route.Decision = &decision
		route.Platform = decision.TargetPlatform
		route.RoutingModel = decision.UpstreamModel
		route.UpstreamModel = decision.UpstreamModel
	}

	return route, nil
}

func (r EffectiveGatewayRoute) WithChannelMapping(mapping ChannelMappingResult) EffectiveGatewayRoute {
	r.Channel = mapping
	if mapping.Mapped {
		r.RoutingModel = mapping.MappedModel
	}
	if r.RoutingModel == "" {
		r.RoutingModel = mapping.MappedModel
	}
	r.UpstreamModel = r.RoutingModel
	return r
}

func (r EffectiveGatewayRoute) WithUpstreamModel(model string) EffectiveGatewayRoute {
	if model != "" {
		r.UpstreamModel = model
	}
	return r
}

type effectiveGatewayRouteContextKey struct{}

func WithEffectiveGatewayRoute(ctx context.Context, route EffectiveGatewayRoute) context.Context {
	return context.WithValue(ctx, effectiveGatewayRouteContextKey{}, route)
}

func EffectiveGatewayRouteFromContext(ctx context.Context) (EffectiveGatewayRoute, bool) {
	route, ok := ctx.Value(effectiveGatewayRouteContextKey{}).(EffectiveGatewayRoute)
	return route, ok
}
