package elasticsearch

import (
	"encoding/json"
	"strings"
)

func parseBool(path string, interfaceMap map[string]interface{}) bool {
	value := walkInterfaceMap(path, interfaceMap)

	if parsedBool, ok := value.(bool); ok {
		return parsedBool
	}
	return false
}

func parseString(path string, interfaceMap map[string]interface{}) string {
	value := walkInterfaceMap(path, interfaceMap)

	if parsedString, ok := value.(string); ok {
		return parsedString
	}
	return ""
}

func parseInt32(path string, interfaceMap map[string]interface{}) int32 {
	return int32(parseFloat64(path, interfaceMap))
}

func parseFloat64(path string, interfaceMap map[string]interface{}) float64 {
	value := walkInterfaceMap(path, interfaceMap)

	if parsedFloat, ok := value.(float64); ok {
		return parsedFloat
	}
	return float64(-1)
}

func walkInterfaceMap(path string, interfaceMap map[string]interface{}) interface{} {
	current := interfaceMap
	keys := strings.Split(path, ".")
	keyCount := len(keys)

	for index, key := range keys {
		if current[key] != nil {
			if index+1 < keyCount {
				current = current[key].(map[string]interface{})
			} else {
				return current[key]
			}
		} else {
			return nil
		}
	}

	return nil
}

func getMapFromBody(rawBody string) (map[string]interface{}, error) {
	if rawBody == "" {
		return make(map[string]interface{}), nil
	}
	var results map[string]interface{}
	err := json.Unmarshal([]byte(rawBody), &results)
	if err != nil {
		results = make(map[string]interface{})
		results["results"] = rawBody
	}

	return results, nil
}
