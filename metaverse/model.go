/*
 * Copyright 2019 The openwallet Authors
 * This file is part of the openwallet library.
 *
 * The openwallet library is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Lesser General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * The openwallet library is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Lesser General Public License for more details.
 */

package metaverse

import (
	"github.com/blocktree/openwallet/openwallet"
	"github.com/tidwall/gjson"
)

type AssetAttachment struct {
	Quantity string `json:"quantity"`
	Symbol   string `json:"symbol"`
}

type Block struct {

	/*

			{
		        "bits" : "1",
		        "hash" : "b81848ef9ae86e84c3da26564bc6ab3a79efc628239d11471ab5cd25c0684c2d",
		        "merkle_tree_hash" : "2a845dfa63a7c20d40dbc4b15c3e970ef36332b367500fd89307053cb4c1a2c1",
		        "mixhash" : "0",
		        "nonce" : "0",
		        "number" : 0,
		        "previous_block_hash" : "0000000000000000000000000000000000000000000000000000000000000000",
		        "timestamp" : 1486796400,
		        "transaction_count" : 1,
		        "version" : 1
		        "transactions" : []
		    }

	*/

	Hash              string
	Merkleroot        string
	Previousblockhash string
	Height            uint64
	Version           uint64
	Time              uint64
	Fork              bool
	transactions      []*Transaction
}

func (wm *WalletManager) NewBlock(json *gjson.Result) *Block {
	obj := &Block{}
	//解析json
	obj.Height = gjson.Get(json.Raw, "number").Uint()
	obj.Hash = gjson.Get(json.Raw, "hash").String()
	obj.Merkleroot = gjson.Get(json.Raw, "merkle_tree_hash").String()
	obj.Previousblockhash = gjson.Get(json.Raw, "previous_block_hash").String()
	obj.Version = gjson.Get(json.Raw, "version").Uint()
	obj.Time = gjson.Get(json.Raw, "timestamp").Uint()

	transactions := make([]*Transaction, 0)
	for _, tx := range gjson.Get(json.Raw, "transactions").Array() {
		txObj := wm.NewTransaction(&tx)
		txObj.BlockHeight = obj.Height
		txObj.BlockHash = obj.Hash
		txObj.Blocktime = int64(obj.Time)
		transactions = append(transactions, txObj)
	}

	obj.transactions = transactions

	return obj
}

//BlockHeader 区块链头
func (b *Block) BlockHeader(symbol string) *openwallet.BlockHeader {

	obj := openwallet.BlockHeader{}
	//解析json
	obj.Hash = b.Hash
	obj.Merkleroot = b.Merkleroot
	obj.Previousblockhash = b.Previousblockhash
	obj.Height = b.Height
	obj.Version = b.Version
	obj.Time = b.Time
	obj.Symbol = symbol

	return &obj
}

type Transaction struct {
	TxID        string
	Version     uint64
	LockTime    int64
	BlockHash   string
	BlockHeight uint64
	Blocktime   int64
	IsCoinBase  bool
	Decimals    int32
	RawHex      string

	Vins  []*Vin
	Vouts []*Vout
}

type Vin struct {
	isCoinbase      bool
	TxID            string
	Vout            uint64
	N               uint64
	Addr            string
	Value           string
	AssetAttachment *AssetAttachment
	IsToken         bool
	LockScript      string
}

type Vout struct {
	N                 uint64
	Addr              string
	Value             string
	Type              string
	AssetAttachment   *AssetAttachment
	IsToken           bool
	LockedHeightRange int64
	LockScript        string
}

func (wm *WalletManager) NewTransaction(json *gjson.Result) *Transaction {

	/*
			{
			"hash": "20a1627a5cdf6cb6d3656161af949d2e54f18114f51888421733a2de8763a2b5",
			"inputs": [
				{
					"address": "MG65zQHtch4zxj9ghZKyTcjrRDiCdPAf8M",
					"previous_output": {
						"hash": "9b38a59167492f4f56dfd2c40cc57eb005b9cf7a25cec5c0e3362d321a7447bd",
						"index": 1
					},
					"script": "[ 3045022100e36fe4e2c254a0dbb59176e8056411983c82fcfc999a99e242e2f720b00f4daf02200f03526b5105b1cf5678b40b98f84f50f3520aa293322a57a6af1919522a174401 ] [ 033a67f19bad4eab86ffade1bd050885e205562e07f8ebb50a114eb15b233a3b86 ]",
					"sequence": 4294967295
				},
			],
			"lock_time": "0",
			"outputs": [
				{
					"address": "MJYG1e7rjQDob7kMFqRKdHYWcwoErrthGT",
					"attachment": {
						"type": "etp"
					},
					"index": 0,
					"locked_height_range": 0,
					"script": "dup hash160 [ 74b57910184277f877886301eaa3358af56c0a47 ] equalverify checksig",
					"value": 187712588
				}
			],
			"version": "4"
		}
	*/

	obj := Transaction{}
	//解析json
	obj.TxID = gjson.Get(json.Raw, "hash").String()
	obj.Version = gjson.Get(json.Raw, "version").Uint()
	obj.LockTime = gjson.Get(json.Raw, "lock_time").Int()
	obj.BlockHeight = gjson.Get(json.Raw, "height").Uint()
	obj.Vins = make([]*Vin, 0)
	if vins := gjson.Get(json.Raw, "inputs"); vins.IsArray() {
		for i, vin := range vins.Array() {
			input := NewTxInput(&vin)
			input.N = uint64(i)
			obj.Vins = append(obj.Vins, input)
		}
	}

	obj.Vouts = make([]*Vout, 0)
	if vouts := gjson.Get(json.Raw, "outputs"); vouts.IsArray() {
		for _, vout := range vouts.Array() {
			output := NewTxOut(&vout)
			obj.Vouts = append(obj.Vouts, output)
		}
	}

	return &obj
}

func NewTxInput(json *gjson.Result) *Vin {

	/*
		{
			"address": "MG65zQHtch4zxj9ghZKyTcjrRDiCdPAf8M",
			"previous_output": {
				"hash": "9b38a59167492f4f56dfd2c40cc57eb005b9cf7a25cec5c0e3362d321a7447bd",
				"index": 1
			},
			"script": "[ 3045022100e36fe4e2c254a0dbb59176e8056411983c82fcfc999a99e242e2f720b00f4daf02200f03526b5105b1cf5678b40b98f84f50f3520aa293322a57a6af1919522a174401 ] [ 033a67f19bad4eab86ffade1bd050885e205562e07f8ebb50a114eb15b233a3b86 ]",
			"sequence": 4294967295
		}
	*/
	obj := Vin{}
	//解析json
	obj.TxID = gjson.Get(json.Raw, "previous_output.hash").String()
	obj.Vout = gjson.Get(json.Raw, "previous_output.index").Uint()
	obj.Addr = gjson.Get(json.Raw, "address").String()

	if obj.TxID == "0000000000000000000000000000000000000000000000000000000000000000" {
		obj.isCoinbase = true
	} else {
		obj.isCoinbase = false
	}

	return &obj
}

func NewTxOut(json *gjson.Result) *Vout {

	/*
		{
			"address": "38nhbDZ7QD9mEiQN3Hx9YjmTusVFWZoRBm",
			"attachment": {
				"quantity": 268220000,
				"symbol": "DNA",
				"type": "asset-transfer"
			},
			"index": 0,
			"locked_height_range": 0,
			"script": "hash160 [ 4ddc128cb6cf0d51baa74b8ea5f05ed6adcdf4ba ] equal",
			"value": 0
		},
		{
			"address": "MJHnq4qNYEptPC4Fc4pZMtrp45htA65Ygr",
			"attachment": {
				"type": "etp"
			},
			"index": 1,
			"locked_height_range": 0,
			"script": "dup hash160 [ 71f8f5732d9af26486b2afe4c8bc13a60b2c0db6 ] equalverify checksig",
			"value": 48950000
		}
	*/
	obj := Vout{}
	//解析json
	obj.Value = gjson.Get(json.Raw, "value").String()
	obj.N = gjson.Get(json.Raw, "index").Uint()
	obj.Addr = gjson.Get(json.Raw, "address").String()
	obj.Type = gjson.Get(json.Raw, "attachment.type").String()
	obj.LockedHeightRange = gjson.Get(json.Raw, "locked_height_range").Int()
	obj.LockScript = gjson.Get(json.Raw, "script").String()

	if obj.Type == "etp" {
		obj.IsToken = false
	} else if obj.Type == "asset-transfer" {
		obj.IsToken = true
	}

	obj.AssetAttachment = &AssetAttachment{
		Quantity: gjson.Get(json.Raw, "attachment.quantity").String(),
		Symbol:   gjson.Get(json.Raw, "attachment.symbol").String(),
	}

	return &obj
}

type ETPBalance struct {
	/*
		"address" : "MTDcfh43xT93odL1Y2uULhRLeWED2fDvBX",
		"available" : 8799980000,
		"confirmed" : 8799980000,
		"frozen" : 0,
		"received" : 129898990000,
		"unspent" : 8799980000
	*/

	Address   string
	Available string
	Confirmed string
	Frozen    string
	Received  string
	Unspent   string
}

func NewETPBalance(json *gjson.Result) *ETPBalance {
	obj := &ETPBalance{}
	//解析json
	obj.Address = gjson.Get(json.Raw, "address").String()
	obj.Available = gjson.Get(json.Raw, "available").String()
	obj.Confirmed = gjson.Get(json.Raw, "confirmed").String()
	obj.Frozen = gjson.Get(json.Raw, "frozen").String()
	obj.Received = gjson.Get(json.Raw, "received").String()
	obj.Unspent = gjson.Get(json.Raw, "unspent").String()

	return obj
}

type TokenBalance struct {
	/*

		"address" : "MTDcfh43xT93odL1Y2uULhRLeWED2fDvBX",
		"decimal_number" : 4,
		"description" : "Metaverse Dual Chain Official Token",
		"issuer" : "DNA",
		"locked_quantity" : 0,
		"quantity" : 51864340000,
		"secondaryissue_threshold" : 0,
		"status" : "unspent",
		"symbol" : "DNA"

	*/

	Address        string
	Decimals       int32
	Symbol         string
	Quantity       string
	Status         string
	LockedQuantity string
}

func NewTokenBalance(json *gjson.Result) *TokenBalance {
	obj := &TokenBalance{}
	//解析json
	obj.Address = gjson.Get(json.Raw, "address").String()
	obj.Decimals = int32(gjson.Get(json.Raw, "decimal_number").Int())
	obj.Symbol = gjson.Get(json.Raw, "symbol").String()
	obj.Quantity = gjson.Get(json.Raw, "quantity").String()
	obj.Status = gjson.Get(json.Raw, "status").String()
	obj.LockedQuantity = gjson.Get(json.Raw, "locked_quantity").String()

	return obj
}
