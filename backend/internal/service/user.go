package service

import (
	"encoding/json"
	"hash/fnv"
	"sort"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID                          int64
	Email                       string
	Username                    string
	Notes                       string
	AvatarURL                   string
	AvatarSource                string
	AvatarMIME                  string
	AvatarByteSize              int
	AvatarSHA256                string
	PasswordHash                string
	Role                        string
	Balance                     float64
	Concurrency                 int
	Status                      string
	AllowedGroups               []int64
	BlockedGroups               []int64
	HiddenPurchasePage          bool
	HiddenCustomMenuResourceIDs []int64
	HiddenCustomMenuIDs         []string
	TokenVersion                int64 // Incremented on password change to invalidate existing tokens
	// TokenVersionResolved indicates TokenVersion already contains the fingerprint-derived
	// value expected in JWT claims and refresh-token state.
	TokenVersionResolved bool
	SignupSource         string
	LastLoginAt          *time.Time
	LastActiveAt         *time.Time
	LastUsedAt           *time.Time
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            *time.Time // 非 nil 表示用户已软删除

	// GroupRates 用户专属分组倍率配置
	// map[groupID]rateMultiplier
	GroupRates map[int64]float64

	// TOTP 双因素认证字段
	TotpSecretEncrypted *string    // AES-256-GCM 加密的 TOTP 密钥
	TotpEnabled         bool       // 是否启用 TOTP
	TotpEnabledAt       *time.Time // TOTP 启用时间

	// 余额不足通知
	BalanceNotifyEnabled       bool
	BalanceNotifyThresholdType string // "fixed" (default) | "percentage"
	BalanceNotifyThreshold     *float64
	BalanceNotifyExtraEmails   []NotifyEmailEntry
	TotalRecharged             float64

	// RPMLimit 用户级每分钟请求数上限（0 = 不限制）。仅在所用分组未设置 rpm_limit
	// 且该 (用户, 分组) 无 rpm_override 时作为全局兜底生效，计数键 rpm:u:{userID}:{min}。
	RPMLimit int

	// UserGroupRPMOverride 来自 auth cache snapshot 的 (user, group) RPM 覆盖值。
	// nil = 该 API Key 对应的 (user, group) 无 override；非 nil 时 checkRPM 直接使用，
	// 避免每请求查 DB。字段不持久化到数据库。
	UserGroupRPMOverride *int

	APIKeys       []APIKey
	Subscriptions []UserSubscription
}

func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

func (u *User) IsActive() bool {
	return u.Status == StatusActive
}

// CanBindGroup checks whether a user can bind to a given group.
// For standard groups:
// - Public groups (non-exclusive): all users can bind unless explicitly blocked
// - Exclusive groups: only users with the group in AllowedGroups can bind
func (u *User) CanBindGroup(groupID int64, isExclusive bool) bool {
	// 公开分组（非专属）：默认可绑定，除非按用户禁用
	if !isExclusive {
		for _, id := range u.BlockedGroups {
			if id == groupID {
				return false
			}
		}
		return true
	}
	// 专属分组：需要在 AllowedGroups 中
	for _, id := range u.AllowedGroups {
		if id == groupID {
			return true
		}
	}
	return false
}

func (u *User) IsPurchasePageHidden() bool {
	return u != nil && u.HiddenPurchasePage
}

func (u *User) IsCustomMenuHidden(id string) bool {
	if u == nil || id == "" {
		return false
	}
	resourceID := CustomMenuResourceID(id)
	for _, hiddenID := range u.HiddenCustomMenuResourceIDs {
		if hiddenID == resourceID {
			return true
		}
	}
	return false
}

func CustomMenuResourceID(id string) int64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(id))
	out := int64(h.Sum64() & ^uint64(1<<63))
	if out <= 1 {
		return 2
	}
	return out
}

func ResolveHiddenCustomMenuIDs(customMenuItemsRaw string, resourceIDs []int64) []string {
	if len(resourceIDs) == 0 {
		return nil
	}
	hidden := make(map[int64]struct{}, len(resourceIDs))
	for _, id := range resourceIDs {
		hidden[id] = struct{}{}
	}
	var items []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(customMenuItemsRaw)), &items); err != nil {
		return nil
	}
	out := make([]string, 0, len(hidden))
	seen := map[string]struct{}{}
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, ok := hidden[CustomMenuResourceID(id)]; !ok {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

func (u *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hash)
	return nil
}

func (u *User) CheckPassword(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)) == nil
}
