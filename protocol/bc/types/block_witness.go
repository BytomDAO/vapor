package types

import (
	"io"

	"github.com/bytom/vapor/encoding/blockchain"
)

// BlockWitness save the consensus node sign
type BlockWitness struct {
	// Witness is a vector of arguments  for validating this block.
	Witness [][]byte
}

func (bw *BlockWitness) readFrom(r *blockchain.Reader) (err error) {
	bw.Witness, err = blockchain.ReadVarstrList(r)
	return err
}

func (bw *BlockWitness) writeTo(w io.Writer) error {
	_, err := blockchain.WriteVarstrList(w, bw.Witness)
	return err
}

// Set save data to index position
func (bw *BlockWitness) Set(index uint64, data []byte) {
	if uint64(len(bw.Witness)) <= index {
		newWitness := make([][]byte, index+1, index+1)
		copy(newWitness, bw.Witness)
		bw.Witness = newWitness
	}
	bw.Witness[index] = data
}

// Delete remove data from index position
func (bw *BlockWitness) Delete(index uint64) {
	if uint64(len(bw.Witness)) > index {
		bw.Witness[index] = nil
	}
}

// Get return data from index position
func (bw *BlockWitness) Get(index uint64) []byte {
	if uint64(len(bw.Witness)) > index {
		return bw.Witness[index]
	}
	return nil
}
