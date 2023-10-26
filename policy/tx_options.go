// Copyright 2021 The go-ethereum Authors
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

package policy

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

var (
	ErrLargeTxOptions   = errors.New("tx options too large")
	ErrInvalidTxOptions = errors.New("invalid tx options")
)

//go:generate go run github.com/fjl/gencodec -type TxOptions -field-override txOptionsMarshaling -out gen_tx_options_json.go

// TxOptions represent policy level user preferences. An honest block producer
// should only include a transaction if all preferences are met. There is
// nothing in consensus that forces these preferences to be followed.
type TxOptions struct {
	// KnownAccounts represents a user's preference of a known prestate before
	// their transaction is included.
	KnownAccounts KnownAccounts `json:"knownAccounts"`
	// BlockNumberMin
	BlockNumberMin *big.Int `json:"blockNumberMin,omitempty"`
	// BlockNumberMax
	BlockNumberMax *big.Int `json:"blockNumberMax,omitempty"`
	// TimestampMin
	TimestampMin *uint64 `json:"timestampMin,omitempty"`
	// TimestampMax
	TimestampMax *uint64 `json:"timestampMax,omitempty"`
}

// field type overrides for gencodec
type txOptionsMarshaling struct {
	BlockNumberMax *hexutil.Big
	BlockNumberMin *hexutil.Big
	TimestampMin   *hexutil.Uint64
	TimestampMax   *hexutil.Uint64
}

// Cost computes the cost of validating the TxOptions. It will return
// the number of storage lookups required by KnownAccounts.
func (opts *TxOptions) Cost() int {
	cost := 0
	for _, account := range opts.KnownAccounts {
		if _, isRoot := account.Root(); isRoot {
			cost += 1
		}
		if slots, isSlots := account.Slots(); isSlots {
			cost += len(slots)
		}
	}
	if opts.BlockNumberMin != nil || opts.BlockNumberMax != nil {
		cost += 1
	}
	if opts.TimestampMin != nil || opts.TimestampMax != nil {
		cost += 1
	}
	return cost
}

// Copy will copy the TxOptions
func (opts *TxOptions) Copy() TxOptions {
	cpy := TxOptions{
		KnownAccounts: make(map[common.Address]KnownAccount),
	}
	for key, val := range opts.KnownAccounts {
		cpy.KnownAccounts[key] = val.Copy()
	}
	if opts.BlockNumberMin != nil {
		*cpy.BlockNumberMin = *opts.BlockNumberMin
	}
	if opts.BlockNumberMax != nil {
		*cpy.BlockNumberMax = *opts.BlockNumberMax
	}
	if opts.TimestampMin != nil {
		*cpy.TimestampMin = *opts.TimestampMin
	}
	if opts.TimestampMax != nil {
		*cpy.TimestampMax = *opts.TimestampMax
	}
	return cpy
}

// KnownAccounts represents a set of knownAccounts
type KnownAccounts map[common.Address]KnownAccount

// KnownAccount allows for a user to express their preference of a known
// prestate at a particular account. Only one of the storage root or
// storage slots is allowed to be set. If the storage root is set, then
// the user prefers their transaction to only be included in a block if
// the account's storage root matches. If the storage slots are set,
// then the user prefers their transaction to only be included if the
// particular storage slot values from state match.
type KnownAccount struct {
	StorageRoot  *common.Hash
	StorageSlots map[common.Hash]common.Hash
}

// UnmarshalJSON will parse the JSON bytes into a KnownAccount struct.
func (ka *KnownAccount) UnmarshalJSON(data []byte) error {
	var hash common.Hash
	if err := json.Unmarshal(data, &hash); err == nil {
		ka.StorageRoot = &hash
		ka.StorageSlots = make(map[common.Hash]common.Hash)
		return nil
	}

	var mapping map[common.Hash]common.Hash
	if err := json.Unmarshal(data, &mapping); err != nil {
		return err
	}
	ka.StorageSlots = mapping

	return nil
}

// MarshalJSON will serialize the KnownAccount into JSON bytes.
func (ka *KnownAccount) MarshalJSON() ([]byte, error) {
	if ka.StorageRoot != nil {
		return json.Marshal(ka.StorageRoot)
	}
	return json.Marshal(ka.StorageSlots)
}

// Copy will copy the KnownAccount
func (ka *KnownAccount) Copy() KnownAccount {
	cpy := KnownAccount{
		StorageRoot:  nil,
		StorageSlots: make(map[common.Hash]common.Hash),
	}

	if ka.StorageRoot != nil {
		*cpy.StorageRoot = *ka.StorageRoot
	}
	for key, val := range ka.StorageSlots {
		cpy.StorageSlots[key] = val
	}
	return cpy
}

// Root will return the storage root and true when the user prefers
// execution against an account's storage root, otherwise it will
// return false.
func (ka *KnownAccount) Root() (common.Hash, bool) {
	if ka.StorageRoot == nil {
		return common.Hash{}, false
	}
	return *ka.StorageRoot, true
}

// Slots will return the storage slots and true when the user prefers
// execution against an account's particular storage slots, otherwise
// it will return false.
func (ka *KnownAccount) Slots() (map[common.Hash]common.Hash, bool) {
	if ka.StorageRoot != nil {
		return ka.StorageSlots, false
	}
	return ka.StorageSlots, true
}