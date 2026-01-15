package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// FirstNonNil 在 map 中按顺序返回首个非 nil 的键值
func FirstNonNil(m map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			return v
		}
	}
	return nil
}

func ToString(v any) string {
	if v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case json.Number:
		return string(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case int:
		return strconv.Itoa(t)
	case int64:
		return strconv.FormatInt(t, 10)
	case map[string]any:
		if lv, ok := t["low"].(float64); ok {
			return strconv.FormatInt(int64(lv), 10)
		}
		b, _ := json.Marshal(t)
		return string(b)
	default:
		s := fmt.Sprint(t)
		return strings.TrimSpace(s)
	}
}

func ToInt64(v any) int64 {
	if v == nil {
		return 0
	}
	switch t := v.(type) {
	case float64:
		return int64(t)
	case int:
		return int64(t)
	case int64:
		return t
	case json.Number:
		i, _ := t.Int64()
		return i
	case string:
		if t == "" {
			return 0
		}
		if iv, err := strconv.ParseInt(t, 10, 64); err == nil {
			return iv
		}
		return 0
	case map[string]any:
		if lv, ok := t["low"].(float64); ok {
			return int64(lv)
		}
		return 0
	default:
		return 0
	}
}

// FlattenMemberMap 规范化上游 member 字段为插件内部键
func FlattenMemberMap(src map[string]any) map[string]any {
	out := map[string]any{}
	// id：优先 id/Id；其次 ref.id
	if idv := FirstNonNil(src, "id", "Id"); idv != nil {
		out["id"] = ToInt64(idv)
	} else if ref, ok := src["ref"].(map[string]any); ok {
		out["id"] = ToInt64(FirstNonNil(ref, "id", "Id"))
	}

	out["username"] = ToString(FirstNonNil(src, "username", "user_name", "userName"))
	out["display_name"] = ToString(FirstNonNil(src, "display_name", "displayName"))
	out["email"] = ToString(FirstNonNil(src, "email", "mail"))
	out["phone"] = ToString(FirstNonNil(src, "phone", "mobile", "phoneNumber", "phone_number"))

	// status -> string
	if sv := FirstNonNil(src, "status", "member_status", "memberStatus"); sv != nil {
		out["status"] = ToString(sv)
	} else {
		out["status"] = ""
	}

	out["created_at"] = ToString(FirstNonNil(src, "created_at", "createdAt"))
	out["updated_at"] = ToString(FirstNonNil(src, "updated_at", "updatedAt"))

	if arrv := FirstNonNil(src, "team_ids", "teamIds"); arrv != nil {
		if arr, ok := arrv.([]any); ok {
			var ids []int64
			for _, v := range arr {
				ids = append(ids, ToInt64(v))
			}
			out["team_ids"] = ids
		}
	}
	return out
}
