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
	"fmt"
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/tidwall/gjson"
)

type WalletManager struct {
	openwallet.AssetsAdapterBase
	WalletClient    *Client                         // 节点客户端
	Config          *WalletConfig                   //钱包管理配置
	Decoder         openwallet.AddressDecoder       //地址编码器
	TxDecoder       openwallet.TransactionDecoder   //交易单编码器
	Log             *log.OWLogger                   //日志工具
	Blockscanner    openwallet.BlockScanner         //区块扫描器
	ContractDecoder openwallet.SmartContractDecoder //智能合约解析器
}

func NewWalletManager() *WalletManager {
	wm := WalletManager{}
	wm.Config = NewConfig(Symbol)
	wm.Decoder = NewAddressDecoder(&wm)
	wm.Log = log.NewOWLogger(wm.Symbol())
	wm.Blockscanner = NewETPBlockScanner(&wm)
	wm.TxDecoder = NewTransactionDecoder(&wm)
	wm.ContractDecoder = NewContractDecoder(&wm)
	return &wm
}

func (wm *WalletManager) GetInfo() (*gjson.Result, error) {

	result, err := wm.WalletClient.Call("getinfo", nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}

//GetBlockHeight 获取区块链高度
func (wm *WalletManager) GetBlockHeader(height ...uint64) (*openwallet.BlockHeader, *openwallet.Error) {

	var request []interface{}

	if len(height) > 0 {
		request = append(request, map[string]interface{}{"height": height[0]})
	}

	result, err := wm.WalletClient.Call("getblockheader", request)
	if err != nil {
		return nil, err
	}

	header := &openwallet.BlockHeader{
		Hash:              result.Get("hash").String(),
		Merkleroot:        result.Get("merkle_tree_hash").String(),
		Previousblockhash: result.Get("previous_block_hash").String(),
		Height:            result.Get("number").Uint(),
		Version:           result.Get("version").Uint(),
		Time:              result.Get("timestamp").Uint(),
		Fork:              false,
		Symbol:            wm.Symbol(),
	}

	return header, nil
}

//GetBlockByHeight 获取区块数据
func (wm *WalletManager) GetBlockByHeight(height uint64) (*Block, *openwallet.Error) {

	request := []interface{}{
		height,
	}

	result, err := wm.WalletClient.Call("getblock", request)
	if err != nil {
		return nil, err
	}

	return wm.NewBlock(result), nil
}

//GetTransaction 获取交易单
func (wm *WalletManager) GetTransaction(txid string) (*Transaction, *openwallet.Error) {

	request := []interface{}{
		txid,
	}

	result, err := wm.WalletClient.Call("gettx", request)
	if err != nil {
		return nil, err
	}

	return wm.NewTransaction(result), nil
}

// GetAddressETP
func (wm *WalletManager) GetAddressETP(address string) (*ETPBalance, *openwallet.Error) {
	request := []interface{}{
		address,
	}

	result, err := wm.WalletClient.Call("getaddressetp", request)
	if err != nil {
		return nil, err
	}

	return NewETPBalance(result), nil
}

// GetAddressAsset
func (wm *WalletManager) GetAddressAsset(address, symbol string) (*TokenBalance, *openwallet.Error) {
	request := []interface{}{
		address,
		map[string]string{"symbol": symbol},
	}

	result, err := wm.WalletClient.Call("getaddressasset", request)

	tokenBalance := &TokenBalance{
		Address:        address,
		Decimals:       0,
		Symbol:         symbol,
		Quantity:       "0",
		Status:         "",
		LockedQuantity: "0",
	}

	if err != nil {
		return tokenBalance, nil
	}

	if result.IsArray() {
		for _, obj := range result.Array() {
			tokenBalance = NewTokenBalance(&obj)
			break
		}
	}

	return tokenBalance, nil
}

// CreateRawTx
func (wm *WalletManager) CreateRawTx(sender []string, receivers map[string]string, change, fees, symbol string, isToken bool) (string, *openwallet.Error) {

	request := map[string]interface{}{
		"senders":   sender,
		"fee":       fees,
	}

	comb := make([]string, 0)
	//["MMRbpJdtxXeNmdwRZa4JjNgraL2XKUeg4e:1460"]
	for addr, amount := range receivers {
		rec := fmt.Sprintf("%s:%s", addr, amount)
		comb = append(comb, rec)
	}

	if len(comb) == 0 {
		return "", openwallet.Errorf(openwallet.ErrCreateRawTransactionFailed, "receivers is empty")
	}

	request["receivers"] = comb

	if len(change) > 0 {
		request["mychange"] = change
	}

	if isToken {
		request["symbol"] = symbol
		request["type"] = 3
	} else {
		request["type"] = 0
	}

	result, err := wm.WalletClient.Call("createrawtx", []interface{}{request})
	if err != nil {
		return "", err
	}

	rawHex := result.String()

	return rawHex, nil
}


// DecodeRawTx
func (wm *WalletManager) DecodeRawTx(rawHex string) (*Transaction, *openwallet.Error) {

	request := []interface{}{
		rawHex,
	}

	result, err := wm.WalletClient.Call("decoderawtx", request)
	if err != nil {
		return nil, err
	}

	tx := wm.NewTransaction(result)
	tx.RawHex = rawHex

	err = wm.FillInputFields(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

// SendRawTx
func (wm *WalletManager) SendRawTx(rawHex string) (string, *openwallet.Error) {

	request := []interface{}{
		rawHex,
	}

	result, err := wm.WalletClient.Call("sendrawtx", request)
	if err != nil {
		return "", err
	}

	txid := result.String()

	return txid, nil
}

// FillInputFields
func (wm *WalletManager) FillInputFields(tx *Transaction) *openwallet.Error {
	for _, input := range tx.Vins {

		if input.isCoinbase {
			break
		}

		intxid := input.TxID
		vout := input.Vout
		preTx, txErr := wm.GetTransaction(intxid)
		if txErr != nil {
			return txErr
		} else {
			preVouts := preTx.Vouts
			if len(preVouts) > int(vout) {
				preOut := preVouts[vout]
				input.Addr = preOut.Addr
				input.Value = preOut.Value
				input.AssetAttachment = preOut.AssetAttachment
				input.IsToken = preOut.IsToken
				input.LockScript = preOut.LockScript
			}
		}
	}

	return nil
}