package mergeutxo

import (
	"github.com/bytom/vapor/consensus"
	"github.com/bytom/vapor/protocol/bc"
	"github.com/bytom/vapor/protocol/bc/types"
	"github.com/bytom/vapor/toolbar/apinode"
)

func MergeUTXO(hostPort, accountID, password, address string, amount uint64) ([]string, error) {
	actions := []interface{}{}

	actions = append(actions, &apinode.ControlAddressAction{
		Address:     address,
		AssetAmount: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: amount},
	})

	actions = append(actions, &apinode.SpendAccountAction{
		AccountID:   accountID,
		AssetAmount: &bc.AssetAmount{AssetId: consensus.BTMAssetID, Amount: amount},
	})

	node := apinode.NewNode(hostPort)

	tpls, err := node.BuildChainTxs(actions)
	if err != nil {
		return []string{}, err
	}

	tpls, err = node.SignTxs(tpls, password)
	if err != nil {
		return []string{}, err
	}

	txs := []*types.Tx{}
	for _, tpl := range tpls {
		txs = append(txs, tpl.Transaction)
	}

	return node.SubmitTxs(txs)
}
