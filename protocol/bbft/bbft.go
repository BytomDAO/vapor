package bbft

import (
	"time"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
)

type bbft struct {
	consensusNodeManager *consensusNodeManager
}

func (b *bbft) IsConsensusPubkey(height uint64, pubkey []byte) (bool, error) {
	return b.consensusNodeManager.isConsensusPubkey(height, pubkey)
}

func (b *bbft) NextLeaderTime(pubkey []byte) (*time.Time, error) {
	return b.consensusNodeManager.nextLeaderTime(pubkey)
}

func (b *bbft) ValidateSign(blockID bc.Hash, pubkey, sign []byte) error {
	if ok := ed25519.Verify(ed25519.PublicKey(pubkey), blockID.Bytes(), sign); !ok {
		return errors.New("validate block signature fail")
	}
	return nil
}
