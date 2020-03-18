package common

import (
	"encoding/json"
)

func IsOpenFederationIssueAsset(rawDefinitionByte []byte) bool {
	var defMap map[string]interface{}
	if err := json.Unmarshal(rawDefinitionByte, &defMap); err != nil {
		return false
	}

	description, ok := defMap["description"].(map[string]interface{})
	if !ok {
		return false
	}

	issueAssetAction, ok := description["issue_asset_action"].(string)
	if !ok {
		return false
	}

	return issueAssetAction == "open_federation_cross_chain"
}
