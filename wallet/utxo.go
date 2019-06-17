package wallet

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/vapor/account"
	"github.com/vapor/consensus"
	"github.com/vapor/consensus/segwit"
	"github.com/vapor/crypto/sha3pool"
	dbm "github.com/vapor/database/leveldb"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol/bc/types"
)

// GetAccountUtxos return all account unspent outputs
func (w *Wallet) GetAccountUtxos(accountID string, id string, unconfirmed, isSmartContract bool, vote bool) []*account.UTXO {
	prefix := account.UTXOPreFix
	if isSmartContract {
		prefix = account.SUTXOPrefix
	}

	accountUtxos := []*account.UTXO{}
	if unconfirmed {
		accountUtxos = w.AccountMgr.ListUnconfirmedUtxo(accountID, isSmartContract)
	}

	accountUtxoIter := w.DB.IteratorPrefix([]byte(prefix + id))
	defer accountUtxoIter.Release()

	for accountUtxoIter.Next() {
		accountUtxo := &account.UTXO{}
		if err := json.Unmarshal(accountUtxoIter.Value(), accountUtxo); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Warn("GetAccountUtxos fail on unmarshal utxo")
			continue
		}

		if vote && accountUtxo.Vote == nil {
			continue
		}

		if accountID == accountUtxo.AccountID || accountID == "" {
			accountUtxos = append(accountUtxos, accountUtxo)
		}
	}
	return accountUtxos
}

func (w *Wallet) attachUtxos(batch dbm.Batch, b *types.Block, txStatus *bc.TransactionStatus) {
	for txIndex, tx := range b.Transactions {
		statusFail, err := txStatus.GetStatus(txIndex)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("attachUtxos fail on get tx status")
			continue
		}

		//hand update the transaction input utxos
		inputUtxos := txInToUtxos(tx, statusFail)
		for _, inputUtxo := range inputUtxos {
			if segwit.IsP2WScript(inputUtxo.ControlProgram) {
				batch.Delete(account.StandardUTXOKey(inputUtxo.OutputID))
			} else {
				batch.Delete(account.ContractUTXOKey(inputUtxo.OutputID))
			}
		}

		//hand update the transaction output utxos
		outputUtxos := txOutToUtxos(tx, statusFail, b.Height)
		utxos := w.filterAccountUtxo(outputUtxos)
		if err := batchSaveUtxos(utxos, batch); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("attachUtxos fail on batchSaveUtxos")
		}
	}
}

func (w *Wallet) detachUtxos(batch dbm.Batch, b *types.Block, txStatus *bc.TransactionStatus) {
	for txIndex := len(b.Transactions) - 1; txIndex >= 0; txIndex-- {
		tx := b.Transactions[txIndex]
		for j := range tx.Outputs {
			code := []byte{}
			switch resOut := tx.Entries[*tx.ResultIds[j]].(type) {
			case *bc.IntraChainOutput:
				code = resOut.ControlProgram.Code
			case *bc.VoteOutput:
				code = resOut.ControlProgram.Code
			default:
				continue
			}

			if segwit.IsP2WScript(code) {
				batch.Delete(account.StandardUTXOKey(*tx.ResultIds[j]))
			} else {
				batch.Delete(account.ContractUTXOKey(*tx.ResultIds[j]))
			}
		}

		statusFail, err := txStatus.GetStatus(txIndex)
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("detachUtxos fail on get tx status")
			continue
		}

		inputUtxos := txInToUtxos(tx, statusFail)
		utxos := w.filterAccountUtxo(inputUtxos)
		if err := batchSaveUtxos(utxos, batch); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("detachUtxos fail on batchSaveUtxos")
			return
		}
	}
}

func (w *Wallet) filterAccountUtxo(utxos []*account.UTXO) []*account.UTXO {
	outsByScript := make(map[string][]*account.UTXO, len(utxos))
	for _, utxo := range utxos {
		scriptStr := string(utxo.ControlProgram)
		outsByScript[scriptStr] = append(outsByScript[scriptStr], utxo)
	}

	result := make([]*account.UTXO, 0, len(utxos))
	for s := range outsByScript {
		if !segwit.IsP2WScript([]byte(s)) {
			for _, utxo := range outsByScript[s] {
				result = append(result, utxo)
			}
			continue
		}

		var hash [32]byte
		sha3pool.Sum256(hash[:], []byte(s))
		data := w.DB.Get(account.ContractKey(hash))
		if data == nil {
			continue
		}

		cp := &account.CtrlProgram{}
		if err := json.Unmarshal(data, cp); err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("filterAccountUtxo fail on unmarshal control program")
			continue
		}

		for _, utxo := range outsByScript[s] {
			utxo.AccountID = cp.AccountID
			utxo.Address = cp.Address
			utxo.ControlProgramIndex = cp.KeyIndex
			utxo.Change = cp.Change
			result = append(result, utxo)
		}
	}
	return result
}

func batchSaveUtxos(utxos []*account.UTXO, batch dbm.Batch) error {
	for _, utxo := range utxos {
		data, err := json.Marshal(utxo)
		if err != nil {
			return errors.Wrap(err, "failed marshal accountutxo")
		}

		if segwit.IsP2WScript(utxo.ControlProgram) {
			batch.Set(account.StandardUTXOKey(utxo.OutputID), data)
		} else {
			batch.Set(account.ContractUTXOKey(utxo.OutputID), data)
		}
	}
	return nil
}

func txInToUtxos(tx *types.Tx, statusFail bool) []*account.UTXO {
	utxos := []*account.UTXO{}
	for _, inpID := range tx.Tx.InputIDs {

		e, err := tx.Entry(inpID)
		if err != nil {
			continue
		}
		utxo := &account.UTXO{}
		switch inp := e.(type) {
		case *bc.Spend:
			resOut, err := tx.IntraChainOutput(*inp.SpentOutputId)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("txInToUtxos fail on get resOut for spedn")
				continue
			}
			if statusFail && *resOut.Source.Value.AssetId != *consensus.BTMAssetID {
				continue
			}
			utxo = &account.UTXO{
				OutputID:       *inp.SpentOutputId,
				AssetID:        *resOut.Source.Value.AssetId,
				Amount:         resOut.Source.Value.Amount,
				ControlProgram: resOut.ControlProgram.Code,
				SourceID:       *resOut.Source.Ref,
				SourcePos:      resOut.Source.Position,
			}
		case *bc.VetoInput:
			resOut, err := tx.VoteOutput(*inp.SpentOutputId)
			if err != nil {
				log.WithFields(log.Fields{"module": logModule, "err": err}).Error("txInToUtxos fail on get resOut for vetoInput")
				continue
			}
			if statusFail && *resOut.Source.Value.AssetId != *consensus.BTMAssetID {
				continue
			}
			utxo = &account.UTXO{
				OutputID:       *inp.SpentOutputId,
				AssetID:        *resOut.Source.Value.AssetId,
				Amount:         resOut.Source.Value.Amount,
				ControlProgram: resOut.ControlProgram.Code,
				SourceID:       *resOut.Source.Ref,
				SourcePos:      resOut.Source.Position,
				Vote:           resOut.Vote,
			}
		default:
			continue
		}
		utxos = append(utxos, utxo)
	}
	return utxos
}

func txOutToUtxos(tx *types.Tx, statusFail bool, blockHeight uint64) []*account.UTXO {
	validHeight := uint64(0)
	if tx.Inputs[0].InputType() == types.CoinbaseInputType {
		validHeight = blockHeight + consensus.CoinbasePendingBlockNumber
	}

	utxos := []*account.UTXO{}
	for i, out := range tx.Outputs {
		entryOutput, err := tx.Entry(*tx.ResultIds[i])
		if err != nil {
			log.WithFields(log.Fields{"module": logModule, "err": err}).Error("txOutToUtxos fail on get entryOutput")
			continue
		}

		utxo := &account.UTXO{}
		switch bcOut := entryOutput.(type) {
		case *bc.IntraChainOutput:
			if statusFail && *out.AssetAmount().AssetId != *consensus.BTMAssetID {
				continue
			}
			utxo = &account.UTXO{
				OutputID:       *tx.OutputID(i),
				AssetID:        *out.AssetAmount().AssetId,
				Amount:         out.AssetAmount().Amount,
				ControlProgram: out.ControlProgram(),
				SourceID:       *bcOut.Source.Ref,
				SourcePos:      bcOut.Source.Position,
				ValidHeight:    validHeight,
			}

		case *bc.VoteOutput:
			if statusFail && *out.AssetAmount().AssetId != *consensus.BTMAssetID {
				continue
			}

			voteValidHeight := blockHeight + consensus.VotePendingBlockNumber
			if validHeight < voteValidHeight {
				validHeight = voteValidHeight
			}

			utxo = &account.UTXO{
				OutputID:       *tx.OutputID(i),
				AssetID:        *out.AssetAmount().AssetId,
				Amount:         out.AssetAmount().Amount,
				ControlProgram: out.ControlProgram(),
				SourceID:       *bcOut.Source.Ref,
				SourcePos:      bcOut.Source.Position,
				ValidHeight:    validHeight,
				Vote:           bcOut.Vote,
			}

		default:
			log.WithFields(log.Fields{"module": logModule}).Warn("txOutToUtxos fail on get bcOut")
			continue
		}

		utxos = append(utxos, utxo)
	}
	return utxos
}
