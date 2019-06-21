package federation

import (
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"time"

	btmTypes "github.com/bytom/protocol/bc/types"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"

	"github.com/vapor/crypto/ed25519/chainkd"
	"github.com/vapor/errors"
	"github.com/vapor/federation/common"
	"github.com/vapor/federation/config"
	"github.com/vapor/federation/database"
	"github.com/vapor/federation/database/orm"
	"github.com/vapor/federation/service"
	"github.com/vapor/federation/util"
	vaporBc "github.com/vapor/protocol/bc"
	vaporTypes "github.com/vapor/protocol/bc/types"
)

var collectInterval = 5 * time.Second

type warder struct {
	db            *gorm.DB
	assetStore    *database.AssetStore
	txCh          chan *orm.CrossTransaction
	fedProg       []byte
	position      uint8
	xpub          chainkd.XPub
	xprv          chainkd.XPrv
	mainchainNode *service.Node
	sidechainNode *service.Node
	remotes       []*service.Warder
}

func NewWarder(db *gorm.DB, assetStore *database.AssetStore, cfg *config.Config) *warder {
	local, remotes := parseWarders(cfg)
	return &warder{
		db:            db,
		assetStore:    assetStore,
		txCh:          make(chan *orm.CrossTransaction),
		fedProg:       util.ParseFedProg(cfg.Warders, cfg.Quorum),
		position:      local.Position,
		xpub:          local.XPub,
		xprv:          string2xprv(xprvStr),
		mainchainNode: service.NewNode(cfg.Mainchain.Upstream),
		sidechainNode: service.NewNode(cfg.Sidechain.Upstream),
		remotes:       remotes,
	}
}

func parseWarders(cfg *config.Config) (*service.Warder, []*service.Warder) {
	var local *service.Warder
	var remotes []*service.Warder
	for _, warderCfg := range cfg.Warders {
		if warderCfg.IsLocal {
			local = service.NewWarder(&warderCfg)
		} else {
			remote := service.NewWarder(&warderCfg)
			remotes = append(remotes, remote)
		}
	}

	if local == nil {
		log.Fatal("none local warder set")
	}

	return local, remotes
}

func (w *warder) Run() {
	ticker := time.NewTicker(collectInterval)
	for ; true; <-ticker.C {
		txs := []*orm.CrossTransaction{}
		if err := w.db.Preload("Chain").Preload("Reqs").
			// do not use "Where(&orm.CrossTransaction{Status: common.CrossTxInitiatedStatus})" directly,
			// otherwise the field "status" will be ignored
			Model(&orm.CrossTransaction{}).Where("status = ?", common.CrossTxInitiatedStatus).
			Find(&txs).Error; err == gorm.ErrRecordNotFound {
			continue
		} else if err != nil {
			log.Warnln("collectPendingTx", err)
		}

		for _, tx := range txs {
			go w.tryProcessCrossTx(tx)
		}
	}
}

func (w *warder) tryProcessCrossTx(ormTx *orm.CrossTransaction) error {
	dbTx := w.db.Begin()
	if err := w.processCrossTx(ormTx); err != nil {
		dbTx.Rollback()
		return err
	}

	return dbTx.Commit().Error
}

func (w *warder) processCrossTx(ormTx *orm.CrossTransaction) error {
	if err := w.validateCrossTx(ormTx); err != nil {
		log.Warnln("invalid cross-chain tx", ormTx)
		return err
	}

	destTx, destTxID, err := w.proposeDestTx(ormTx)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx}).Warnln("proposeDestTx")
		return err
	}

	if err := w.initDestTxSigns(destTx, ormTx); err != nil {
		log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx}).Warnln("initDestTxSigns")
		return err
	}

	signs, err := w.getSigns(destTx, ormTx)
	if err != nil {
		log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx}).Warnln("getSigns")
		return err
	}

	w.attachSignsForTx(destTx, ormTx, w.position, signs)

	for _, remote := range w.remotes {
		signs, err := remote.RequestSigns(destTx, ormTx)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "remote": remote, "cross-chain tx": ormTx}).Warnln("RequestSign")
			return err
		}

		w.attachSignsForTx(destTx, ormTx, remote.Position, signs)
	}

	if w.isTxSignsReachQuorum(destTx) && w.isLeader() {
		submittedTxID, err := w.submitTx(destTx)
		if err != nil {
			log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx, "dest tx": destTx}).Warnln("submitTx")
			return err
		}

		if submittedTxID != destTxID {
			log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx, "builtTx ID": destTxID, "submittedTx ID": submittedTxID}).Warnln("submitTx ID mismatch")
			return err
		}

		if err := w.updateSubmission(ormTx); err != nil {
			log.WithFields(log.Fields{"err": err, "cross-chain tx": ormTx}).Warnln("updateSubmission")
			return err
		}
	}

	return nil
}

func (w *warder) validateCrossTx(tx *orm.CrossTransaction) error {
	switch tx.Status {
	case common.CrossTxRejectedStatus:
		return errors.New("cross-chain tx rejected")
	case common.CrossTxSubmittedStatus:
		return errors.New("cross-chain tx submitted")
	case common.CrossTxCompletedStatus:
		return errors.New("cross-chain tx completed")
	default:
		return nil
	}
}

func (w *warder) proposeDestTx(tx *orm.CrossTransaction) (interface{}, string, error) {
	switch tx.Chain.Name {
	case "bytom":
		return w.buildSidechainTx(tx)
	case "vapor":
		return w.buildMainchainTx(tx)
	default:
		return nil, "", errors.New("unknown source chain")
	}
}

func (w *warder) buildSidechainTx(ormTx *orm.CrossTransaction) (*vaporTypes.Tx, string, error) {
	destTxData := &vaporTypes.TxData{Version: 1, TimeRange: 0}
	muxID := &vaporBc.Hash{}
	if err := muxID.UnmarshalText([]byte(ormTx.SourceMuxID)); err != nil {
		return nil, "", errors.Wrap(err, "Unmarshal muxID")
	}

	for _, req := range ormTx.Reqs {
		// getAsset from assetStore instead of preload asset, in order to save db query overload
		asset, err := w.assetStore.GetByOrmID(req.AssetID)
		if err != nil {
			return nil, "", errors.Wrap(err, "get asset by ormAsset ID")
		}

		assetID := &vaporBc.AssetID{}
		if err := assetID.UnmarshalText([]byte(asset.AssetID)); err != nil {
			return nil, "", errors.Wrap(err, "Unmarshal muxID")
		}

		rawDefinitionByte, err := hex.DecodeString(asset.RawDefinitionByte)
		if err != nil {
			return nil, "", errors.Wrap(err, "decode rawDefinitionByte")
		}

		issuanceProgramByte, err := hex.DecodeString(asset.IssuanceProgram)
		if err != nil {
			return nil, "", errors.Wrap(err, "decode issuanceProgramByte")
		}

		input := vaporTypes.NewCrossChainInput(nil, *muxID, *assetID, req.AssetAmount, req.SourcePos, 1, rawDefinitionByte, issuanceProgramByte)
		destTxData.Inputs = append(destTxData.Inputs, input)

		controlProgram, err := hex.DecodeString(req.Script)
		if err != nil {
			return nil, "", errors.Wrap(err, "decode req.Script")
		}

		output := vaporTypes.NewIntraChainOutput(*assetID, req.AssetAmount, controlProgram)
		destTxData.Outputs = append(destTxData.Outputs, output)
	}

	destTx := vaporTypes.NewTx(*destTxData)
	w.addInputWitness(destTx)

	if err := w.db.Model(&orm.CrossTransaction{}).
		Where(&orm.CrossTransaction{ID: ormTx.ID}).
		UpdateColumn(&orm.CrossTransaction{
			DestTxHash: sql.NullString{destTx.ID.String(), true},
		}).Error; err != nil {
		return nil, "", err
	}

	return destTx, destTx.ID.String(), nil
}

func (w *warder) buildMainchainTx(ormTx *orm.CrossTransaction) (*btmTypes.Tx, string, error) {
	return nil, "", errors.New("buildMainchainTx not implemented yet")
}

// tx is a pointer to types.Tx, so the InputArguments can be set and be valid afterward
func (w *warder) addInputWitness(tx interface{}) {
	switch tx := tx.(type) {
	case *vaporTypes.Tx:
		args := [][]byte{w.fedProg}
		for i := range tx.Inputs {
			tx.SetInputArguments(uint32(i), args)
		}

	case *btmTypes.Tx:
		args := [][]byte{util.SegWitWrap(w.fedProg)}
		for i := range tx.Inputs {
			tx.SetInputArguments(uint32(i), args)
		}
	}
}

func (w *warder) initDestTxSigns(destTx interface{}, ormTx *orm.CrossTransaction) error {
	for i := 1; i <= len(w.remotes)+1; i++ {
		if err := w.db.Create(&orm.CrossTransactionSign{
			CrossTransactionID: ormTx.ID,
			WarderID:           uint8(i),
			Status:             common.CrossTxSignPendingStatus,
		}).Error; err != nil {
			return err
		}
	}

	return w.db.Model(&orm.CrossTransaction{}).
		Where(&orm.CrossTransaction{ID: ormTx.ID}).
		UpdateColumn(&orm.CrossTransaction{
			Status: common.CrossTxPendingStatus,
		}).Error
}

// TODO:
func (w *warder) getSigns(destTx interface{}, ormTx *orm.CrossTransaction) ([]string, error) {
	if ormTx.Status != common.CrossTxPendingStatus || !ormTx.DestTxHash.Valid {
		return nil, errors.New("cross-chain tx status error")
	}

	signData, err := w.getSignData(destTx)
	if err != nil {
		return nil, errors.New("getSignData")
	}

	var signs []string
	for _, data := range signData {
		// TODO: sign it
		msg := []byte{}
		sign := w.xprv.Sign(msg)
		signs = append(signs, hex.EncodeToString(sign))
	}

	return signs, nil
}

func (w *warder) getSignData(destTx interface{}) ([]string, error) {
	var signData []string

	switch destTx := destTx.(type) {
	case *vaporTypes.Tx:
		signData = make([]string, len(destTx.Inputs))
		for i := range destTx.Inputs {
			signHash := destTx.SigHash(uint32(i))
			signData[i] = signHash.String()
		}

	case *btmTypes.Tx:
		signData = make([]string, len(destTx.Inputs))
		for i := range destTx.Inputs {
			signHash := destTx.SigHash(uint32(i))
			signData[i] = signHash.String()
		}

	default:
		return []string{}, errors.New("unknown tx type")
	}

	return signData, nil
}

// TODO:
func (w *warder) attachSignsForTx(destTx interface{}, ormTx *orm.CrossTransaction, position uint8, signs []string) error {

	b, err := json.Marshal(signs)
	if err != nil {
		return errors.Wrap(err, "marshal signs")
	}

	return w.db.Model(&orm.CrossTransactionSign{}).
		Where(&orm.CrossTransactionSign{
			CrossTransactionID: ormTx.ID,
			WarderID:           w.position,
		}).
		UpdateColumn(&orm.CrossTransactionSign{
			Signatures: string(b),
			Status:     common.CrossTxSignCompletedStatus,
		}).Error

}

// TODO:
func (w *warder) isTxSignsReachQuorum(destTx interface{}) bool {
	return false
}

func (w *warder) isLeader() bool {
	return w.position == 1
}

func (w *warder) submitTx(destTx interface{}) (string, error) {
	switch tx := destTx.(type) {
	case *btmTypes.Tx:
		return w.mainchainNode.SubmitTx(tx)
	case *vaporTypes.Tx:
		return w.sidechainNode.SubmitTx(tx)
	default:
		return "", errors.New("unknown destTx type")
	}
}

func (w *warder) updateSubmission(ormTx *orm.CrossTransaction) error {
	if err := w.db.Model(&orm.CrossTransaction{}).
		Where(&orm.CrossTransaction{ID: ormTx.ID}).
		UpdateColumn(&orm.CrossTransaction{
			Status: common.CrossTxSubmittedStatus,
		}).Error; err != nil {
		return err
	}

	for _, remote := range w.remotes {
		remote.NotifySubmission(ormTx)
	}
	return nil
}
