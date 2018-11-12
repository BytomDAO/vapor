package types

import (
	"fmt"

	"github.com/bytom/protocol/bc/types/bytom"
	"github.com/bytom/protocol/vm"
	"github.com/bytom/protocol/vm/vmutil"
)

// MapTx converts a types TxData object into its entries-based
// representation.
func MapTx(oldTx *TxData) *bytom.Tx {
	txID, txHeader, entries := mapTx(oldTx)
	tx := &bytom.Tx{
		TxHeader: txHeader,
		ID:       txID,
		Entries:  entries,
		InputIDs: make([]bytom.Hash, len(oldTx.Inputs)),
	}

	spentOutputIDs := make(map[bytom.Hash]bool)
	for id, e := range entries {
		var ord uint64
		switch e := e.(type) {
		case *bytom.Issuance:
			ord = e.Ordinal

		case *bytom.Spend:
			ord = e.Ordinal
			spentOutputIDs[*e.SpentOutputId] = true
			if *e.WitnessDestination.Value.AssetId == *bytom.BTMAssetID {
				tx.GasInputIDs = append(tx.GasInputIDs, id)
			}

		case *bytom.Coinbase:
			ord = 0

		default:
			continue
		}

		if ord >= uint64(len(oldTx.Inputs)) {
			continue
		}
		tx.InputIDs[ord] = id
	}

	for id := range spentOutputIDs {
		tx.SpentOutputIDs = append(tx.SpentOutputIDs, id)
	}
	return tx
}

func mapTx(tx *TxData) (headerID bytom.Hash, hdr *bytom.TxHeader, entryMap map[bytom.Hash]bytom.Entry) {
	entryMap = make(map[bytom.Hash]bytom.Entry)
	addEntry := func(e bytom.Entry) bytom.Hash {
		id := bytom.EntryID(e)
		entryMap[id] = e
		return id
	}

	var (
		spends    []*bytom.Spend
		issuances []*bytom.Issuance
		coinbase  *bytom.Coinbase
	)

	muxSources := make([]*bytom.ValueSource, len(tx.Inputs))
	for i, input := range tx.Inputs {
		switch inp := input.TypedInput.(type) {
		case *IssuanceInput:
			nonceHash := inp.NonceHash()
			assetDefHash := inp.AssetDefinitionHash()
			value := input.AssetAmount()

			issuance := bytom.NewIssuance(&nonceHash, &value, uint64(i))
			issuance.WitnessAssetDefinition = &bytom.AssetDefinition{
				Data: &assetDefHash,
				IssuanceProgram: &bytom.Program{
					VmVersion: inp.VMVersion,
					Code:      inp.IssuanceProgram,
				},
			}
			issuance.WitnessArguments = inp.Arguments
			issuanceID := addEntry(issuance)

			muxSources[i] = &bytom.ValueSource{
				Ref:   &issuanceID,
				Value: &value,
			}
			issuances = append(issuances, issuance)

		case *SpendInput:
			// create entry for prevout
			prog := &bytom.Program{VmVersion: inp.VMVersion, Code: inp.ControlProgram}
			src := &bytom.ValueSource{
				Ref:      &inp.SourceID,
				Value:    &inp.AssetAmount,
				Position: inp.SourcePosition,
			}
			prevout := bytom.NewOutput(src, prog, 0) // ordinal doesn't matter for prevouts, only for result outputs
			prevoutID := addEntry(prevout)
			fmt.Println("00000000000000: ", prevoutID.String())
			// create entry for spend
			spend := bytom.NewSpend(&prevoutID, uint64(i))
			spend.WitnessArguments = inp.Arguments
			spendID := addEntry(spend)
			// setup mux
			muxSources[i] = &bytom.ValueSource{
				Ref:   &spendID,
				Value: &inp.AssetAmount,
			}
			spends = append(spends, spend)

		case *CoinbaseInput:
			coinbase = bytom.NewCoinbase(inp.Arbitrary)
			coinbaseID := addEntry(coinbase)

			out := tx.Outputs[0]
			muxSources[i] = &bytom.ValueSource{
				Ref:   &coinbaseID,
				Value: &out.AssetAmount,
			}
		}
	}

	mux := bytom.NewMux(muxSources, &bytom.Program{VmVersion: 1, Code: []byte{byte(vm.OP_TRUE)}})
	muxID := addEntry(mux)

	// connect the inputs to the mux
	for _, spend := range spends {
		spentOutput := entryMap[*spend.SpentOutputId].(*bytom.Output)
		spend.SetDestination(&muxID, spentOutput.Source.Value, spend.Ordinal)
	}
	for _, issuance := range issuances {
		issuance.SetDestination(&muxID, issuance.Value, issuance.Ordinal)
	}

	if coinbase != nil {
		coinbase.SetDestination(&muxID, mux.Sources[0].Value, 0)
	}

	// convert types.outputs to the bytom.output
	var resultIDs []*bytom.Hash
	for i, out := range tx.Outputs {
		src := &bytom.ValueSource{
			Ref:      &muxID,
			Value:    &out.AssetAmount,
			Position: uint64(i),
		}
		var resultID bytom.Hash
		if vmutil.IsUnspendable(out.ControlProgram) {
			// retirement
			r := bytom.NewRetirement(src, uint64(i))
			resultID = addEntry(r)
		} else {
			// non-retirement
			prog := &bytom.Program{out.VMVersion, out.ControlProgram}
			o := bytom.NewOutput(src, prog, uint64(i))
			resultID = addEntry(o)
		}

		dest := &bytom.ValueDestination{
			Value:    src.Value,
			Ref:      &resultID,
			Position: 0,
		}
		resultIDs = append(resultIDs, &resultID)
		mux.WitnessDestinations = append(mux.WitnessDestinations, dest)
	}

	h := bytom.NewTxHeader(tx.Version, tx.SerializedSize, tx.TimeRange, resultIDs)
	return addEntry(h), h, entryMap
}

func mapBlockHeader(old *BlockHeader) (bytom.Hash, *bytom.BlockHeader) {
	bh := bytom.NewBlockHeader(old.Version, old.Height, &old.PreviousBlockHash, old.Timestamp, &old.TransactionsMerkleRoot, &old.TransactionStatusHash, old.Nonce, old.Bits)
	return bytom.EntryID(bh), bh
}

// MapBlock converts a types block to bc block
func MapBlock(old *Block) *bytom.Block {
	if old == nil {
		return nil
	}

	b := new(bytom.Block)
	b.ID, b.BlockHeader = mapBlockHeader(&old.BlockHeader)
	for _, oldTx := range old.Transactions {
		b.Transactions = append(b.Transactions, oldTx.Tx)
	}
	return b
}
