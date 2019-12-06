package asset

import (
	"encoding/json"

	"github.com/bytom/vapor/blockchain/query"
	chainjson "github.com/bytom/vapor/encoding/json"
)

//Annotated annotate the asset
func Annotated(a *Asset) (*query.AnnotatedAsset, error) {
	jsonDefinition := json.RawMessage(`{}`)

	// a.RawDefinitionByte is the asset definition as it appears on the
	// blockchain, so it's untrusted and may not be valid json.
	if chainjson.IsValidJSON(a.RawDefinitionByte) {
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
