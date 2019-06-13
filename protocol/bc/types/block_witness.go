package types

import (
	"io"

	"github.com/vapor/encoding/blockchain"
	"github.com/vapor/errors"
)

var (
	errInvalidBlockWitnessIndex = errors.New("block witness index exceed length.")
)

type BlockWitness struct {
	// Witness is a vector of arguments  for validating this block.
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

func (bw *BlockWitness) Update(index uint64, data []byte) error {
	if index >= uint64(len(bw.Witness)) {
		return errInvalidBlockWitnessIndex
	}
	bw.Witness[index] = data
	return nil
}
