package wecomauth

import (
	"fmt"
	"strings"
)

type Kind string

const (
	KindSelfBuilt         Kind = "self_built"
	KindDelegatedOpenWork Kind = "delegated_openwork"
)

func ResolveKind(channelCode, appType string) (Kind, error) {
	channel := strings.ToLower(strings.TrimSpace(channelCode))
	app := strings.ToLower(strings.TrimSpace(appType))
	if channel != "wechat" {
		return "", fmt.Errorf("unsupported channel_code: %s", channelCode)
	}
	switch app {
	case "wecom":
		return KindSelfBuilt, nil
	case "openwork":
		return KindDelegatedOpenWork, nil
	default:
		return "", fmt.Errorf("unsupported app_type: %s", appType)
	}
}

func IsDelegated(kind Kind) bool {
	return kind == KindDelegatedOpenWork
}
