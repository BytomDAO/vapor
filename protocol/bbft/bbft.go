package bbft

import (
	"time"

	"github.com/vapor/crypto/ed25519"
	"github.com/vapor/errors"
	"github.com/vapor/protocol/bc"
	"github.com/vapor/protocol"
	"github.com/vapor/database"
)

type bbft struct {
	consensusNodeManager *consensusNodeManager
}

func newBbft(store *database.Store, chain *protocol.Chain) *bbft {
	return &bbft{
		consensusNodeManager: newConsensusNodeManager(store, chain),
	}
}

// IsConsensusPubkey determine whether a public key is a consensus node at a specified height
func (b *bbft) IsConsensusPubkey(height uint64, pubkey []byte) (bool, error) {
	return b.consensusNodeManager.isConsensusPubkey(height, pubkey)
}

// NextLeaderTime returns the start time of the specified public key as the next leader node
func (b *bbft) NextLeaderTime(pubkey []byte) (*time.Time, error) {
	return b.consensusNodeManager.nextLeaderTime(pubkey)
}

// ValidateSign verify the signature of block id
func (b *bbft) ValidateSign(blockID bc.Hash, pubkey, sign []byte) error {
	if ok := ed25519.Verify(ed25519.PublicKey(pubkey), blockID.Bytes(), sign); !ok {
		return errors.New("validate block signature fail")
	}
	return nil
}
