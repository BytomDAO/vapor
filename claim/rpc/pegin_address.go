package rpc

import (
	"encoding/hex"

	"github.com/vapor/common"
	"github.com/vapor/consensus"
	"github.com/vapor/crypto"
	chainjson "github.com/vapor/encoding/json"
	"github.com/vapor/equity/pegin_contract"
	"github.com/vapor/protocol/vm/vmutil"
	"github.com/vapor/wallet"
)

type fundingResp struct {
	MainchainAddress string             `json:"mainchain_address"`
	ControlProgram   chainjson.HexBytes `json:"control_program,omitempty"`
	ClaimScript      chainjson.HexBytes `json:"claim_script"`
}

type BytomPeginRpc struct {
	ClaimArgs
	Wallet *wallet.Wallet
}

type ClaimArgs struct {
	AccountID    string `json:"account_id"`
	AccountAlias string `json:"account_alias"`
}

func (b *BytomPeginRpc) GetPeginAddress() (interface{}, error) {

	accountID := b.AccountID
	if b.AccountAlias != "" {
		account, err := b.Wallet.AccountMgr.FindByAlias(b.AccountAlias)
		if err != nil {
			return nil, err
		}

		accountID = account.ID
	}

	mainchainAddress, claimScript, err := b.CreatePeginAddress(accountID, false)
	if err != nil {
		return nil, err
	}

	return &fundingResp{
		MainchainAddress: mainchainAddress,
		ClaimScript:      claimScript,
	}, nil
}

func (b *BytomPeginRpc) GetPeginContractAddress() (interface{}, error) {
	accountID := b.AccountID
	if b.AccountAlias != "" {
		account, err := b.Wallet.AccountMgr.FindByAlias(b.AccountAlias)
		if err != nil {
			return nil, err
		}

		accountID = account.ID
	}

	mainchainAddress, controlProgram, claimScript, err := b.CreatePeginContractAddress(accountID, false)
	if err != nil {
		return nil, err
	}

	return &fundingResp{
		MainchainAddress: mainchainAddress,
		ControlProgram:   controlProgram,
		ClaimScript:      claimScript,
	}, nil
}

func (b *BytomPeginRpc) CreatePeginAddress(accountID string, change bool) (string, []byte, error) {
	// 通过配置获取
	claimCtrlProg, err := b.Wallet.AccountMgr.CreateAddress(b.AccountID, change)
	if err != nil {
		return "", nil, err
	}
	claimScript := claimCtrlProg.ControlProgram

	federationRedeemScript := vmutil.CalculateContract(consensus.ActiveNetParams.FedpegXPubs, claimScript)

	scriptHash := crypto.Sha256(federationRedeemScript)

	address, err := common.NewPeginAddressWitnessScriptHash(scriptHash, &consensus.ActiveNetParams)
	if err != nil {
		return "", nil, err
	}

	return address.EncodeAddress(), claimScript, nil

}

func (b *BytomPeginRpc) GetPeginControlPrograms(claimScript []byte) (string, []byte) {
	federationRedeemScript := vmutil.CalculateContract(consensus.ActiveNetParams.FedpegXPubs, claimScript)
	scriptHash := crypto.Sha256(federationRedeemScript)

	address, err := common.NewPeginAddressWitnessScriptHash(scriptHash, &consensus.ActiveNetParams)
	if err != nil {
		return "", nil
	}

	redeemContract := address.ScriptAddress()

	program := []byte{}
	program, err = vmutil.P2WSHProgram(redeemContract)
	if err != nil {
		return "", nil
	}

	return address.EncodeAddress(), program
}

func (b *BytomPeginRpc) CreatePeginContractPrograms(accountID string, change bool) (string, []byte, error) {
	// 通过配置获取
	claimCtrlProg, err := b.Wallet.AccountMgr.CreateAddress(accountID, change)
	if err != nil {
		return "", nil, err
	}
	claimScript := claimCtrlProg.ControlProgram

	peginContractPrograms, err := pegin_contract.GetPeginContractPrograms(claimScript)
	if err != nil {
		return "", nil, err
	}
	return hex.EncodeToString(peginContractPrograms), claimScript, nil

}

func (b *BytomPeginRpc) CreatePeginContractAddress(accountID string, change bool) (string, []byte, []byte, error) {
	// 通过配置获取
	claimCtrlProg, err := b.Wallet.AccountMgr.CreateAddress(accountID, change)
	if err != nil {
		return "", nil, nil, err
	}
	claimScript := claimCtrlProg.ControlProgram

	peginContractPrograms, err := pegin_contract.GetPeginContractPrograms(claimScript)
	if err != nil {
		return "", nil, nil, err
	}

	scriptHash := crypto.Sha256(peginContractPrograms)

	address, err := common.NewPeginAddressWitnessScriptHash(scriptHash, &consensus.ActiveNetParams)
	if err != nil {
		return "", nil, nil, err
	}

	redeemContract := address.ScriptAddress()

	program := []byte{}
	program, err = vmutil.P2WSHProgram(redeemContract)
	if err != nil {
		return "", nil, nil, err
	}

	return address.EncodeAddress(), program, claimScript, nil

}

func (b *BytomPeginRpc) GetPeginContractControlPrograms(claimScript []byte) (string, []byte) {

	peginContractPrograms, err := pegin_contract.GetPeginContractPrograms(claimScript)
	if err != nil {
		return "", nil
	}
	scriptHash := crypto.Sha256(peginContractPrograms)

	address, err := common.NewPeginAddressWitnessScriptHash(scriptHash, &consensus.ActiveNetParams)
	if err != nil {
		return "", nil
	}

	redeemContract := address.ScriptAddress()

	program := []byte{}
	program, err = vmutil.P2WSHProgram(redeemContract)
	if err != nil {
		return "", nil
	}

	return address.EncodeAddress(), program
}
