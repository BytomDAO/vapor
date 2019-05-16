package asset

import (
	"encoding/json"

	"github.com/vapor/blockchain/query"
)

func isValidJSON(b []byte) bool {
	var v interface{}
	err := json.Unmarshal(b, &v)
	return err == nil
}

//Annotated annotate the asset
func Annotated(a *Asset) (*query.AnnotatedAsset, error) {
	jsonDefinition := json.RawMessage(`{}`)

	// a.RawDefinitionByte is the asset definition as it appears on the
	// blockchain, so it's untrusted and may not be valid json.
	if isValidJSON(a.RawDefinitionByte) {
		jsonDefinition = json.RawMessage(a.RawDefinitionByte)
	}

	annotatedAsset := &query.AnnotatedAsset{
		ID:                a.AssetID,
		Alias:             *a.Alias,
		VMVersion:         a.VMVersion,
		RawDefinitionByte: a.RawDefinitionByte,
		Definition:        &jsonDefinition,
	}

	return annotatedAsset, nil
}
