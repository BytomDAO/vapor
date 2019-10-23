package common

import (
	"encoding/json"
)

func IsCrossChainAssetOfNoBytom(rawDefinitionByte []byte) bool {
	var defMap map[string]interface{}
	json.Unmarshal(rawDefinitionByte, &defMap)

	description, ok := defMap["description"].(map[string]interface{})
	if !ok {
		return false
	}

	issueAssetAction, ok := description["issue_asset_action"].(string)
	if !ok {
		return false
	}

	if issueAssetAction != "cross_chain" {
		return false
	}
	return true
}
