package mock

import (
	"math/rand"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bytom/vapor/blockchain/txbuilder"
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/crypto/ed25519/chainkd"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
)

func NewCrosschainTx(privateKey string) *types.Tx {
	var xprv chainkd.XPrv
	if err := xprv.UnmarshalText([]byte(privateKey)); err != nil {
		log.WithField("err", err).Panic("fail on xprv UnmarshalText")
	}

	//generate the tx
	randomHash := [32]byte{}
	rand.Seed(time.Now().UnixNano())
	rand.Read(randomHash[:])
	builder := txbuilder.NewBuilder(time.Now())
	input := types.NewCrossChainInput(nil, bc.NewHash(randomHash), *consensus.BTMAssetID, 10000000, 1, 1, []byte{}, []byte{})
	if err := builder.AddInput(input, &txbuilder.SigningInstruction{}); err != nil {
		log.WithField("err", err).Panic("fail on add builder input")
	}

	output := types.NewIntraChainOutput(*consensus.BTMAssetID, 10000000, []byte{0x00, 0x14, 0xd4, 0x88, 0xda, 0x3b, 0x78, 0x1a, 0xa4, 0xe1, 0xdb, 0x9b, 0x7b, 0x27, 0x4d, 0xf1, 0x04, 0x33, 0x54, 0x83, 0x0b, 0x07})
	if err := builder.AddOutput(output); err != nil {
		log.WithField("err", err).Panic("fail on add builder output")
	}

	tpl, _, err := builder.Build()
	if err != nil {
		log.WithField("err", err).Panic("fail on add builder tx")
	}

	//sign tx
	signHash := tpl.Hash(0).Byte32()
	sign := xprv.Sign(signHash[:])
	tpl.Transaction.SetInputArguments(0, [][]byte{sign})

	data, err := tpl.Transaction.TxData.MarshalText()
	if err != nil {
		log.WithField("err", err).Panic("fail on unmarshal tx")
	}

	tpl.Transaction.TxData.SerializedSize = uint64(len(data) / 2)
	tpl.Transaction.Tx.SerializedSize = uint64(len(data) / 2)
	return tpl.Transaction
}
