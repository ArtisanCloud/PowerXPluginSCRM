package social_channel_governance

import "context"

// WeComFoundationAdapter exposes minimal foundation capability matrix for wechat/wecom.
// 当前仅开放 tags 域，避免未实现域被误判为可用。
type WeComFoundationAdapter struct{}

func NewWeComFoundationAdapter() *WeComFoundationAdapter {
	return &WeComFoundationAdapter{}
}

func (a *WeComFoundationAdapter) CapabilityMatrix(_ context.Context, _ string) map[string]string {
	return map[string]string{
		"auth":              "supported",
		"tags":              "supported",
		"org":               "not_supported",
		"external_contacts": "not_supported",
		"leads":             "not_supported",
	}
}

