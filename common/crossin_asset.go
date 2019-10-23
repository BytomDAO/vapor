package common

import (
	"encoding/json"
	"errors"
)

func IsCrossChainAssetOfNoBytom(rawDefinitionByte []byte) (bool, error) {
	var defMap map[string]interface{}
	if err := json.Unmarshal(rawDefinitionByte, &defMap); err != nil {
		return false, err
	}

	description, ok := defMap["description"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	issueAssetAction, ok := description["issue_asset_action"].(string)
	if !ok {
		return false, nil
	}

	if issueAssetAction != "cross_chain" {
		return false, errors.New("issueAssetAction is error")
	}
	return true, nil
}
