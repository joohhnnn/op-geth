// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package legacypool

import (
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/policy"
)

// Tests that transactions can be added to strict lists and list contents and
// nonce boundaries are correctly maintained.
func TestStrictListAdd(t *testing.T) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 1024)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}
	// Insert the transactions in a random order
	list := newList(true)
	for _, v := range rand.Perm(len(txs)) {
		list.Add(txs[v], DefaultConfig.PriceBump, nil)
	}
	// Verify internal state
	if len(list.txs.items) != len(txs) {
		t.Errorf("transaction count mismatch: have %d, want %d", len(list.txs.items), len(txs))
	}
	for i, tx := range txs {
		if list.txs.items[tx.Nonce()] != tx {
			t.Errorf("item %d: transaction mismatch: have %v, want %v", i, list.txs.items[tx.Nonce()], tx)
		}
	}
}

// TestFilterTxOptions tests filtering by invalid TxOptions.
func TestFilterTxOptions(t *testing.T) {
	// Create an in memory state db to test against.
	memDb := rawdb.NewMemoryDatabase()
	db := state.NewDatabase(memDb)
	state, _ := state.New(common.Hash{}, db, nil)

	// Create a private key to sign transactions.
	key, _ := crypto.GenerateKey()

	// Create a list.
	list := newList(true)

	// Create a transaction with no defined tx options
	// and add to the list.
	tx := transaction(0, 1000, key)
	list.Add(tx, DefaultConfig.PriceBump, nil)

	// There should be no drops at this point.
	// No state has been modified.
	drops, errs := list.FilterTxOptions(state)
	if len(errs) != 0 {
		t.Fatalf("got errors when filtering by TxOptions: %s", errs)
	}
	if count := len(drops); count != 0 {
		t.Fatalf("got %d filtered by TxOptions when there should not be any", count)
	}

	// Create another transaction with a known account storage root tx option
	// and add to the list.
	tx2 := transaction(1, 1000, key)
	tx2.SetTxOptions(&policy.TxOptions{
		KnownAccounts: map[common.Address]policy.KnownAccount{
			common.Address{19: 1}: policy.KnownAccount{
				StorageRoot: &types.EmptyRootHash,
			},
		},
	})
	list.Add(tx2, DefaultConfig.PriceBump, nil)

	// There should still be no drops as no state has been modified.
	drops, errs = list.FilterTxOptions(state)
	if len(errs) != 0 {
		t.Fatalf("got errors when filtering by TxOptions: %s", errs)
	}
	if count := len(drops); count != 0 {
		t.Fatalf("got %d filtered by TxOptions when there should not be any", count)
	}

	// Set state that conflicts with tx2's policy
	state.SetState(common.Address{19: 1}, common.Hash{}, common.Hash{31: 1})

	// tx2 should be the single transaction filtered out
	drops, errs = list.FilterTxOptions(state)
	if len(errs) != 0 {
		t.Fatalf("got errors when filtering by TxOptions: %s", errs)
	}
	if count := len(drops); count != 1 {
		t.Fatalf("got %d filtered by TxOptions when there should be a single one", count)
	}
	if drops[0] != tx2 {
		t.Fatalf("Got %x, expected %x", drops[0].Hash(), tx2.Hash())
	}
}


func BenchmarkListAdd(b *testing.B) {
	// Generate a list of transactions to insert
	key, _ := crypto.GenerateKey()

	txs := make(types.Transactions, 100000)
	for i := 0; i < len(txs); i++ {
		txs[i] = transaction(uint64(i), 0, key)
	}
	// Insert the transactions in a random order
	priceLimit := big.NewInt(int64(DefaultConfig.PriceLimit))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		list := newList(true)
		for _, v := range rand.Perm(len(txs)) {
			list.Add(txs[v], DefaultConfig.PriceBump, nil)
			list.Filter(priceLimit, DefaultConfig.PriceBump)
		}
	}
}
