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
	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/tidwall/gjson"
)

type WalletManager struct {
	*openwallet.AssetsAdapterBase
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
	//wm.TxDecoder = NewTransactionDecoder(&wm)
	//wm.ContractDecoder = NewContractDecoder(&wm)
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
func (wm *WalletManager) GetBlockHeader() (*openwallet.BlockHeader, *openwallet.Error) {

	result, err := wm.WalletClient.Call("getblockheader", nil)
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