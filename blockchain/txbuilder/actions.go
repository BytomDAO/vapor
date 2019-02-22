package txbuilder

import (
	"context"
	stdjson "encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	ipfs "github.com/ipfs/go-ipfs-api"
	"github.com/vapor/common"
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

	out := types.NewTxOutput(*a.AssetId, a.Amount, program)
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

	out := types.NewTxOutput(*a.AssetId, a.Amount, a.Program)
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
	out := types.NewTxOutput(*a.AssetId, a.Amount, program)
	return b.AddOutput(out)
}

func (a *retireAction) ActionType() string {
	return "retire"
}

const (
	file uint32 = iota
	data
)

type dataAction struct {
	Type uint32 `json:"type"`
	Data string `json:"data"`
}

func (a *dataAction) Build(ctx context.Context, b *TemplateBuilder) error {

	var r io.Reader

	switch a.Type {
	case file:
		// 检查文件是否存在
		fi, err := os.Stat(a.Data)
		if os.IsNotExist(err) {
			return err
		}
		if fi.IsDir() {
			return fmt.Errorf("data [%s] is directory", a.Data)
		}
		r, err = os.Open(a.Data)
		if err != nil {
			return err
		}

	case data:
		if a.Data == "" {
			return errors.New("data is empty")
		}
		// 生成文件对象
		r = strings.NewReader(a.Data)
	default:
	}

	// 连接ipfs节点
	sh := ipfs.NewShell("localhost:5001")
	cid, err := sh.Add(r)
	if err != nil {
		return err
	}

	fmt.Println(cid)

	return nil
}
