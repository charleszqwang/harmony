package types

import (
	"io"

	"github.com/ethereum/go-ethereum/rlp"

	"github.com/harmony-one/harmony/block"
)

// BodyV1 is the V1 block body
type BodyV1 struct {
	f bodyFieldsV1
}

type bodyFieldsV1 struct {
	Transactions     []*Transaction
	Uncles           []*block.Header
	IncomingReceipts CXReceiptsProofs
}

// Transactions returns the list of transactions.
//
// The returned list is a deep copy; the caller may do anything with it without
// affecting the original.
func (b *BodyV1) Transactions() (txs []*Transaction) {
	for _, tx := range b.f.Transactions {
		txs = append(txs, tx.Copy())
	}
	return txs
}

// TransactionAt returns the transaction at the given index in this block.
// It returns nil if index is out of bounds.
func (b *BodyV1) TransactionAt(index int) *Transaction {
	if index < 0 || index >= len(b.f.Transactions) {
		return nil
	}
	return b.f.Transactions[index].Copy()
}

// SetTransactions sets the list of transactions with a deep copy of the given
// list.
func (b *BodyV1) SetTransactions(newTransactions []*Transaction) {
	var txs []*Transaction
	for _, tx := range newTransactions {
		txs = append(txs, tx.Copy())
	}
	b.f.Transactions = txs
}

// Uncles returns a deep copy of the list of uncle headers of this block.
func (b *BodyV1) Uncles() (uncles []*block.Header) {
	for _, uncle := range b.f.Uncles {
		uncles = append(uncles, CopyHeader(uncle))
	}
	return uncles
}

// SetUncles sets the list of uncle headers with a deep copy of the given list.
func (b *BodyV1) SetUncles(newUncle []*block.Header) {
	var uncles []*block.Header
	for _, uncle := range newUncle {
		uncles = append(uncles, CopyHeader(uncle))
	}
	b.f.Uncles = uncles
}

// IncomingReceipts returns a deep copy of the list of incoming cross-shard
// transaction receipts of this block.
func (b *BodyV1) IncomingReceipts() (incomingReceipts CXReceiptsProofs) {
	return b.f.IncomingReceipts.Copy()
}

// SetIncomingReceipts sets the list of incoming cross-shard transaction
// receipts of this block with a dep copy of the given list.
func (b *BodyV1) SetIncomingReceipts(newIncomingReceipts CXReceiptsProofs) {
	b.f.IncomingReceipts = newIncomingReceipts.Copy()
}

// EncodeRLP RLP-encodes the block body into the given writer.
func (b *BodyV1) EncodeRLP(w io.Writer) error {
	return rlp.Encode(w, &b.f)
}

// DecodeRLP RLP-decodes a block body from the given RLP stream into the
// receiver.
func (b *BodyV1) DecodeRLP(s *rlp.Stream) error {
	return s.Decode(&b.f)
}
