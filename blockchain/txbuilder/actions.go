package txbuilder

import (
	"context"
	stdjson "encoding/json"
	"errors"

	"golang.org/x/crypto/sha3"

	"github.com/vapor/common"
	"github.com/vapor/config"
	"github.com/vapor/consensus"
	"github.com/vapor/encoding/json"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
	"github.com/vapor/protocol/vm/vmutil"
)

// DecodeControlAddressAction convert input data to action struct
func DecodeControlAddressAction(data []byte) (Action, error) {
	a := new(controlAddressAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type controlAddressAction struct {
	bc.AssetAmount
	Address string `json:"address"`
}

func (a *controlAddressAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.Address == "" {
		missing = append(missing, "address")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	address, err := common.DecodeAddress(a.Address, &consensus.ActiveNetParams)
	if err != nil {
		return err
	}

	redeemContract := address.ScriptAddress()
	program := []byte{}
	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return errors.New("unsupport address type")
	}
	if err != nil {
		return err
	}

	out := types.NewIntraChainOutput(*a.AssetId, a.Amount, program)
	return b.AddOutput(out)
}

func (a *controlAddressAction) ActionType() string {
	return "control_address"
}

// DecodeControlProgramAction convert input data to action struct
func DecodeControlProgramAction(data []byte) (Action, error) {
	a := new(controlProgramAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type controlProgramAction struct {
	bc.AssetAmount
	Program json.HexBytes `json:"control_program"`
}

func (a *controlProgramAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if len(a.Program) == 0 {
		missing = append(missing, "control_program")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	out := types.NewIntraChainOutput(*a.AssetId, a.Amount, a.Program)
	return b.AddOutput(out)
}

func (a *controlProgramAction) ActionType() string {
	return "control_program"
}

// DecodeRetireAction convert input data to action struct
func DecodeRetireAction(data []byte) (Action, error) {
	a := new(retireAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type retireAction struct {
	bc.AssetAmount
	Arbitrary json.HexBytes `json:"arbitrary"`
}

func (a *retireAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	program, err := vmutil.RetireProgram(a.Arbitrary)
	if err != nil {
		return err
	}
	out := types.NewIntraChainOutput(*a.AssetId, a.Amount, program)
	return b.AddOutput(out)
}

func (a *retireAction) ActionType() string {
	return "retire"
}

// DecodeCrossOutAction convert input data to action struct
func DecodeCrossOutAction(data []byte) (Action, error) {
	a := new(crossOutAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type crossOutAction struct {
	bc.AssetAmount
	Address string `json:"address"`
}

func (a *crossOutAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.Address == "" {
		missing = append(missing, "address")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	address, err := common.DecodeAddress(a.Address, &consensus.MainNetParams)
	if err != nil {
		return err
	}

	redeemContract := address.ScriptAddress()
	program := []byte{}
	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return errors.New("unsupport address type")
	}
	if err != nil {
		return err
	}

	out := types.NewCrossChainOutput(*a.AssetId, a.Amount, program)
	return b.AddOutput(out)
}

func (a *crossOutAction) ActionType() string {
	return "cross_chain_out"
}

// DecodeVoteOutputAction convert input data to action struct
func DecodeVoteOutputAction(data []byte) (Action, error) {
	a := new(voteOutputAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type voteOutputAction struct {
	bc.AssetAmount
	Address string        `json:"address"`
	Vote    json.HexBytes `json:"vote"`
}

func (a *voteOutputAction) Build(ctx context.Context, b *TemplateBuilder) error {
	var missing []string
	if a.Address == "" {
		missing = append(missing, "address")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}
	if len(a.Vote) == 0 {
		missing = append(missing, "vote")
	}
	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	address, err := common.DecodeAddress(a.Address, &consensus.ActiveNetParams)
	if err != nil {
		return err
	}

	redeemContract := address.ScriptAddress()
	program := []byte{}
	switch address.(type) {
	case *common.AddressWitnessPubKeyHash:
		program, err = vmutil.P2WPKHProgram(redeemContract)
	case *common.AddressWitnessScriptHash:
		program, err = vmutil.P2WSHProgram(redeemContract)
	default:
		return errors.New("unsupport address type")
	}
	if err != nil {
		return err
	}

	out := types.NewVoteOutput(*a.AssetId, a.Amount, program, a.Vote)
	return b.AddOutput(out)
}

func (a *voteOutputAction) ActionType() string {
	return "vote_output"
}

// DecodeCrossInAction convert input data to action struct
func DecodeCrossInAction(data []byte) (Action, error) {
	a := new(crossInAction)
	err := stdjson.Unmarshal(data, a)
	return a, err
}

type crossInAction struct {
	bc.AssetAmount
	SourceID          bc.Hash       `json:"source_id"`
	SourcePos         uint64        `json:"source_pos"`
	VMVersion         uint64        `json:"vm_version"`
	RawDefinitionByte json.HexBytes `json:"raw_definition_byte"`
	IssuanceProgram   json.HexBytes `json:"issuance_program"`
}

func (a *crossInAction) Build(ctx context.Context, builder *TemplateBuilder) error {
	var missing []string
	if a.SourceID.IsZero() {
		missing = append(missing, "source_id")
	}
	if a.AssetId.IsZero() {
		missing = append(missing, "asset_id")
	}
	if a.Amount == 0 {
		missing = append(missing, "amount")
	}

	if len(missing) > 0 {
		return MissingFieldsError(missing...)
	}

	if err := a.checkAssetID(); err != nil {
		return err
	}

	// arguments will be set when materializeWitnesses
	txin := types.NewCrossChainInput(nil, a.SourceID, *a.AssetId, a.Amount, a.SourcePos, a.VMVersion, a.RawDefinitionByte, a.IssuanceProgram)
	tplIn := &SigningInstruction{}
	fed := config.CommonConfig.Federation
	tplIn.AddRawWitnessKeys(fed.Xpubs, nil, fed.Quorum)
	return builder.AddInput(txin, tplIn)
}

func (a *crossInAction) ActionType() string {
	return "cross_chain_in"
}

func (c *crossInAction) checkAssetID() error {
	defHash := bc.NewHash(sha3.Sum256(c.RawDefinitionByte))
	assetID := bc.ComputeAssetID(c.IssuanceProgram, c.VMVersion, &defHash)

	if *c.AssetId != *consensus.BTMAssetID && assetID != *c.AssetAmount.AssetId {
		return errors.New("incorrect asset_idincorrect asset_id")
	}

	return nil
}
