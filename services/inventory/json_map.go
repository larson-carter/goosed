package inventory

import "gorm.io/datatypes"

func mapFromJSONMap(src datatypes.JSONMap) map[string]any {
	if src == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func toJSONMap(src map[string]any) datatypes.JSONMap {
	out := datatypes.JSONMap{}
	if src == nil {
		return out
	}
	for k, v := range src {
		out[k] = v
	}
	return out
}
