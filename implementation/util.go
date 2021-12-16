package implementation

import "fmt"

func convertInterfaceMapToStringMap(m map[string]interface{}) map[string]string {
	mapString := make(map[string]string)
	for key, value := range m {
		var strValue string
		strKey := fmt.Sprintf("%v", key)
		if value == nil {
			strValue = ""
		} else {
			strValue = fmt.Sprintf("%v", value)
		}
		mapString[strKey] = strValue
	}
	return mapString
}
