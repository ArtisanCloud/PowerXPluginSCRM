package marketplace

import "gorm.io/datatypes"

func toJSONMap(data map[string]any) datatypes.JSONMap {
	if len(data) == 0 {
		return datatypes.JSONMap{}
	}
	out := make(datatypes.JSONMap, len(data))
	for k, v := range data {
		out[k] = v
	}
	return out
}

func jsonFromBytes(payload []byte) datatypes.JSONMap {
	if len(payload) == 0 {
		return datatypes.JSONMap{}
	}
	return datatypes.JSONMap{"raw": string(payload)}
}
