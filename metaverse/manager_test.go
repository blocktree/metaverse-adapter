/*
 * Copyright 2018 The openwallet Authors
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
	"github.com/astaxie/beego/config"
	"github.com/blocktree/openwallet/v2/log"
	"path/filepath"
	"testing"
)

var (
	tw *WalletManager
)

func init() {

	tw = testNewWalletManager()
}

func testNewWalletManager() *WalletManager {
	wm := NewWalletManager()

	//读取配置
	absFile := filepath.Join("conf", "ETP.ini")
	c, err := config.NewConfig("ini", absFile)
	if err != nil {
		return nil
	}
	wm.LoadAssetsConfig(c)
	wm.WalletClient.Debug = true
	return wm
}

func TestWalletManager_GetInfo(t *testing.T) {
	tw.GetInfo()
}

func TestWalletManager_GetBlockHeader(t *testing.T) {
	//height := GetLocalBlockHeight()
	header, err := tw.GetBlockHeader()
	if err != nil {
		t.Errorf("GetBlockHeader failed unexpected error: %v\n", err)
		return
	}
	log.Infof("GetBlockHeader = %+v", header)
}

func TestWalletManager_GetBlockByHeight(t *testing.T) {
	block, err := tw.GetBlockByHeight(2958165)

	if err != nil {
		t.Errorf("GetBlockByHeight failed unexpected error: %v\n", err)
		return
	}

	t.Logf("BlockHash = %v \n", block.Hash)
	t.Logf("BlockHeight = %v \n", block.Height)
	t.Logf("Blocktime = %v \n", block.Time)

	for _, tx := range block.transactions {

		t.Logf("TxID = %v \n", tx.TxID)
		t.Logf("IsCoinBase = %v \n", tx.IsCoinBase)
		t.Logf("LockTime = %v \n", tx.LockTime)

		t.Logf("========= vins ========= \n")

		for i, vin := range tx.Vins {
			t.Logf("TxID[%d] = %v \n", i, vin.TxID)
			t.Logf("Vout[%d] = %v \n", i, vin.Vout)
			t.Logf("Addr[%d] = %v \n", i, vin.Addr)
			t.Logf("Value[%d] = %v \n", i, vin.Value)
		}

		t.Logf("========= vouts ========= \n")

		for i, out := range tx.Vouts {
			t.Logf("Addr[%d] = %v \n", i, out.Addr)
			t.Logf("Value[%d] = %v \n", i, out.Value)
			t.Logf("Value[%d] = %v \n", i, out.Type)
			t.Logf("Value[%d] = %v \n", i, out.IsToken)
			t.Logf("Value[%d] = %v \n", i, out.AssetAttachment)
		}

	}

}

func TestWalletManager_GetTransaction(t *testing.T) {

	tx, err := tw.GetTransaction("485b64ce2dc3fac43c3a9cc35f9158e0b11ccf11930ff1f2993d74be40d96605")

	if err != nil {
		t.Errorf("GetTransaction failed unexpected error: %v\n", err)
		return
	}

	t.Logf("TxID = %v \n", tx.TxID)
	t.Logf("IsCoinBase = %v \n", tx.IsCoinBase)
	t.Logf("LockTime = %v \n", tx.LockTime)

	t.Logf("========= vins ========= \n")

	for i, vin := range tx.Vins {
		t.Logf("TxID[%d] = %v \n", i, vin.TxID)
		t.Logf("Vout[%d] = %v \n", i, vin.Vout)
		t.Logf("Addr[%d] = %v \n", i, vin.Addr)
		t.Logf("Value[%d] = %v \n", i, vin.Value)
	}

	t.Logf("========= vouts ========= \n")

	for i, out := range tx.Vouts {
		t.Logf("Addr[%d] = %v \n", i, out.Addr)
		t.Logf("Value[%d] = %v \n", i, out.Value)
		t.Logf("Value[%d] = %v \n", i, out.Type)
		t.Logf("Value[%d] = %v \n", i, out.IsToken)
		t.Logf("Value[%d] = %v \n", i, out.AssetAttachment)
	}
}

func TestWalletManager_GetAddressETP(t *testing.T) {
	balance, err := tw.GetAddressETP("234234")

	if err != nil {
		t.Errorf("GetAddressETP failed unexpected error: %v\n", err)
		return
	}
	log.Infof("balance = %+v", balance)
}

func TestWalletManager_GetAddressAsset(t *testing.T) {
	balance, err := tw.GetAddressAsset("33434", "DNA")

	if err != nil {
		t.Errorf("GetAddressAsset failed unexpected error: %v\n", err)
		return
	}
	log.Infof("balance = %+v", balance)
}

func TestWalletManager_CreateRawTx(t *testing.T) {
	sender := "MUsTC2PCF52yNvAeGNXJUKy9CfLVHV9yYj"
	receiver := "B6UbwNZrz82QqUPBsSCPUkEJ1CjxSzSwnewziCseCt4c"
	amount := "100000"
	fees := tw.Config.MinFees.Shift(tw.Decimal()).String()
	rawHex, err := tw.CreateRawTx([]string{sender}, map[string]string{receiver: amount}, "", fees, "", false)
	if err != nil {
		t.Errorf("CreateRawTx failed unexpected error: %v\n", err)
		return
	}
	log.Infof("rawHex = %+v", rawHex)
}

func TestWalletManager_DecodeRawTx(t *testing.T) {
	rawHex := "0400000001d58650140f9957acec4cabc9a05a0e332fa7123d394ce89791f6058466c722910000000000ffffffff02a0860100000000001976a914d3e7f1c96a7be7903867a17f18e16cae8fad8d4d88ac0100000000000000501c993b000000001976a914e607f73ea755a41b4b649114a9bed5dba1ca8da088ac010000000000000000000000"
	tx, err := tw.DecodeRawTx(rawHex)
	if err != nil {
		t.Errorf("DecodeRawTx failed unexpected error: %v\n", err)
		return
	}

	t.Logf("TxID = %v \n", tx.TxID)
	t.Logf("IsCoinBase = %v \n", tx.IsCoinBase)
	t.Logf("LockTime = %v \n", tx.LockTime)
	t.Logf("RawHex = %v \n", tx.RawHex)

	t.Logf("========= vins ========= \n")

	for i, vin := range tx.Vins {
		t.Logf("TxID[%d] = %v \n", i, vin.TxID)
		t.Logf("Vout[%d] = %v \n", i, vin.Vout)
		t.Logf("Addr[%d] = %v \n", i, vin.Addr)
		t.Logf("Value[%d] = %v \n", i, vin.Value)
	}

	t.Logf("========= vouts ========= \n")

	for i, out := range tx.Vouts {
		t.Logf("Addr[%d] = %v \n", i, out.Addr)
		t.Logf("Value[%d] = %v \n", i, out.Value)
		t.Logf("Value[%d] = %v \n", i, out.Type)
		t.Logf("Value[%d] = %v \n", i, out.IsToken)
		t.Logf("Value[%d] = %v \n", i, out.AssetAttachment)
	}
}
