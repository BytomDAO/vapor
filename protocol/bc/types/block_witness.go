package types

import (
	"io"

	"github.com/vapor/encoding/blockchain"
)

type BlockWitness struct {
	// Witness is a vector of arguments to the previous block's
	// ConsensusProgram for validating this block.
	Witness [][]byte
}

func (bw *BlockWitness) writeTo(w io.Writer) error {
	_, err := blockchain.WriteVarstrList(w, bw.Witness)
	return err
}

func (bw *BlockWitness) readFrom(r *blockchain.Reader) (err error) {
	bw.Witness, err = blockchain.ReadVarstrList(r)
	return err
}
