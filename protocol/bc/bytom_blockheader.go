package bc

import "io"

func (BytomBlockHeader) typ() string { return "blockheader" }
func (bh *BytomBlockHeader) writeForHash(w io.Writer) {
	mustWriteForHash(w, bh.Version)
	mustWriteForHash(w, bh.Height)
	mustWriteForHash(w, bh.PreviousBlockId)
	mustWriteForHash(w, bh.Timestamp)
	mustWriteForHash(w, bh.TransactionsRoot)
	mustWriteForHash(w, bh.TransactionStatusHash)
	mustWriteForHash(w, bh.Bits)
	mustWriteForHash(w, bh.Nonce)
}

// NewBytomBlockHeader creates a new BlockHeader and populates
// its body.
func NewBytomBlockHeader(version, height uint64, previousBlockID *Hash, timestamp uint64, transactionsRoot, transactionStatusHash *Hash, nonce, bits uint64) *BytomBlockHeader {
	return &BytomBlockHeader{
		Version:               version,
		Height:                height,
		PreviousBlockId:       previousBlockID,
		Timestamp:             timestamp,
		TransactionsRoot:      transactionsRoot,
		TransactionStatusHash: transactionStatusHash,
		TransactionStatus:     nil,
		Bits:                  bits,
		Nonce:                 nonce,
	}
}
