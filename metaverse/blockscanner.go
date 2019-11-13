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
	"fmt"
	"github.com/blocktree/openwallet/openwallet"
	"github.com/shopspring/decimal"
	"time"
)

const (
	maxExtractingSize = 10 //并发的扫描线程数
)

//ETPBlockScanner
type ETPBlockScanner struct {
	*openwallet.BlockScannerBase

	CurrentBlockHeight   uint64         //当前区块高度
	extractingCH         chan struct{}  //扫描工作令牌
	wm                   *WalletManager //钱包管理者
	IsScanMemPool        bool           //是否扫描交易池
	RescanLastBlockCount uint64         //重扫上N个区块数量

}

type ExtractOutput map[string][]*openwallet.TxOutPut
type ExtractInput map[string][]*openwallet.TxInput
type ExtractData map[string]*openwallet.TxExtractData

//ExtractResult 扫描完成的提取结果
type ExtractResult struct {
	extractData map[string]ExtractData
	TxID        string
	BlockHeight uint64
	Success     bool
}

//SaveResult 保存结果
type SaveResult struct {
	TxID        string
	BlockHeight uint64
	Success     bool
}

//NewETPBlockScanner 创建区块链扫描器
func NewETPBlockScanner(wm *WalletManager) *ETPBlockScanner {
	bs := ETPBlockScanner{
		BlockScannerBase: openwallet.NewBlockScannerBase(),
	}

	bs.extractingCH = make(chan struct{}, maxExtractingSize)
	bs.wm = wm
	bs.IsScanMemPool = true
	bs.RescanLastBlockCount = 0

	//设置扫描任务
	bs.SetTask(bs.ScanBlockTask)

	return &bs
}

//SetRescanBlockHeight 重置区块链扫描高度
func (bs *ETPBlockScanner) SetRescanBlockHeight(height uint64) error {
	height = height - 1
	if height < 0 {
		return fmt.Errorf("block height to rescan must greater than 0.")
	}

	block, err := bs.wm.GetBlockHeader(height)
	if err != nil {
		return err
	}

	bs.SaveLocalBlockHead(height, block.Hash)

	return nil
}

//ScanBlockTask 扫描任务
func (bs *ETPBlockScanner) ScanBlockTask() {

	//获取本地区块高度
	localHeight, localHash, err := bs.GetLocalBlockHead()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block height; unexpected error: %v", err)
		return
	}

	currentHeight := localHeight
	currentHash := localHash

	for {

		if !bs.Scanning {
			//区块扫描器已暂停，马上结束本次任务
			return
		}

		//获取最大高度
		maxHeader, err := bs.wm.GetBlockHeader()
		if err != nil {
			//下一个高度找不到会报异常
			bs.wm.Log.Std.Info("block scanner can not get rpc-server block height; unexpected error: %v", err)
			break
		}

		maxHeight := maxHeader.Height

		//是否已到最新高度
		if currentHeight >= maxHeight {
			bs.wm.Log.Std.Info("block scanner has scanned full chain data. Current height: %d", maxHeight)
			break
		}

		//继续扫描下一个区块
		currentHeight = currentHeight + 1

		bs.wm.Log.Std.Info("block scanner scanning height: %d ...", currentHeight)

		block, err := bs.wm.GetBlockByHeight(currentHeight)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

			//记录未扫区块
			unscanRecord := openwallet.NewUnscanRecord(currentHeight, "", err.Error(), bs.wm.Symbol())
			bs.SaveUnscanRecord(unscanRecord)
			bs.wm.Log.Std.Info("block height: %d extract failed.", currentHeight)
			continue
		}

		isFork := false

		//判断hash是否上一区块的hash
		if currentHash != block.Previousblockhash {

			bs.wm.Log.Std.Info("block has been fork on height: %d.", currentHeight)
			bs.wm.Log.Std.Info("block height: %d local hash = %s ", currentHeight-1, currentHash)
			bs.wm.Log.Std.Info("block height: %d mainnet hash = %s ", currentHeight-1, block.Previousblockhash)

			bs.wm.Log.Std.Info("delete recharge records on block height: %d.", currentHeight-1)

			//查询本地分叉的区块
			forkBlock, _ := bs.GetLocalBlock(currentHeight - 1)

			//删除上一区块链的所有充值记录
			//bs.DeleteRechargesByHeight(currentHeight - 1)
			//删除上一区块链的未扫记录
			bs.DeleteUnscanRecord(currentHeight - 1)
			currentHeight = currentHeight - 2 //倒退2个区块重新扫描
			if currentHeight <= 0 {
				currentHeight = 1
			}

			localBlock, err := bs.GetLocalBlock(currentHeight)
			if err != nil {
				bs.wm.Log.Std.Error("block scanner can not get local block; unexpected error: %v", err)

				//查找core钱包的RPC
				bs.wm.Log.Info("block scanner prev block height:", currentHeight)

				localBlock, err = bs.wm.GetBlockByHeight(currentHeight)
				if err != nil {
					bs.wm.Log.Std.Error("block scanner can not get prev block; unexpected error: %v", err)
					break
				}

			}

			//重置当前区块的hash
			currentHash = localBlock.Hash

			bs.wm.Log.Std.Info("rescan block on height: %d, hash: %s .", currentHeight, currentHash)

			//重新记录一个新扫描起点
			bs.SaveLocalBlockHead(localBlock.Height, localBlock.Hash)

			isFork = true

			if forkBlock != nil {

				//通知分叉区块给观测者，异步处理
				bs.newBlockNotify(forkBlock, isFork)
			}

		} else {

			batchErr := bs.BatchExtractTransaction(block.Height, block.Hash, block.transactions)
			if batchErr != nil {
				bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", batchErr)
			}

			//重置当前区块的hash
			currentHash = block.Hash

			//保存本地新高度
			bs.SaveLocalBlockHead(currentHeight, currentHash)
			bs.SaveLocalBlock(block)

			isFork = false

			//通知新区块给观测者，异步处理
			bs.newBlockNotify(block, isFork)
		}

	}

	//重扫前N个块，为保证记录找到
	for i := currentHeight - bs.RescanLastBlockCount; i < currentHeight; i++ {
		bs.scanBlock(i)
	}

	//重扫失败区块
	bs.RescanFailedRecord()

}

//ScanBlock 扫描指定高度区块
func (bs *ETPBlockScanner) ScanBlock(height uint64) error {

	block, err := bs.scanBlock(height)
	if err != nil {
		return err
	}

	//通知新区块给观测者，异步处理
	bs.newBlockNotify(block, false)

	return nil
}

func (bs *ETPBlockScanner) scanBlock(height uint64) (*Block, error) {

	block, err := bs.wm.GetBlockByHeight(height)
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)

		//记录未扫区块
		unscanRecord := openwallet.NewUnscanRecord(height, "", err.Error(), bs.wm.Symbol())
		bs.SaveUnscanRecord(unscanRecord)
		bs.wm.Log.Std.Info("block height: %d extract failed.", height)
		return nil, err
	}

	bs.wm.Log.Std.Info("block scanner scanning height: %d ...", block.Height)

	batchErr := bs.BatchExtractTransaction(block.Height, block.Hash, block.transactions)
	if batchErr != nil {
		bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", batchErr)
	}

	//保存区块
	//bs.wm.SaveLocalBlock(block)

	return block, nil
}

//rescanFailedRecord 重扫失败记录
func (bs *ETPBlockScanner) RescanFailedRecord() {

	var (
		blockMap = make(map[uint64][]string)
	)

	list, err := bs.GetUnscanRecords()
	if err != nil {
		bs.wm.Log.Std.Info("block scanner can not get rescan data; unexpected error: %v", err)
	}

	//组合成批处理
	for _, r := range list {

		if _, exist := blockMap[r.BlockHeight]; !exist {
			blockMap[r.BlockHeight] = make([]string, 0)
		}

		if len(r.TxID) > 0 {
			arr := blockMap[r.BlockHeight]
			arr = append(arr, r.TxID)

			blockMap[r.BlockHeight] = arr
		}
	}

	for height, _ := range blockMap {

		if height == 0 {
			continue
		}

		bs.wm.Log.Std.Info("block scanner rescanning height: %d ...", height)

		block, err := bs.wm.GetBlockByHeight(height)
		if err != nil {
			bs.wm.Log.Std.Info("block scanner can not get new block data; unexpected error: %v", err)
			continue
		}

		batchErr := bs.BatchExtractTransaction(block.Height, block.Hash, block.transactions)
		if batchErr != nil {
			bs.wm.Log.Std.Info("block scanner can not extractRechargeRecords; unexpected error: %v", batchErr)
			continue
		}

		//删除未扫记录
		bs.DeleteUnscanRecord(height)
	}
}

//newBlockNotify 获得新区块后，通知给观测者
func (bs *ETPBlockScanner) newBlockNotify(block *Block, isFork bool) {
	header := block.BlockHeader(bs.wm.Symbol())
	header.Fork = isFork
	bs.NewBlockNotify(header)
}

//BatchExtractTransaction 批量提取交易单
//bitcoin 1M的区块链可以容纳3000笔交易，批量多线程处理，速度更快
func (bs *ETPBlockScanner) BatchExtractTransaction(blockHeight uint64, blockHash string, txs []*Transaction) error {

	var (
		quit       = make(chan struct{})
		done       = 0 //完成标记
		failed     = 0
		shouldDone = len(txs) //需要完成的总数
	)

	if len(txs) == 0 {
		return fmt.Errorf("BatchExtractTransaction block is nil.")
	}

	//生产通道
	producer := make(chan ExtractResult)
	defer close(producer)

	//消费通道
	worker := make(chan ExtractResult)
	defer close(worker)

	//保存工作
	saveWork := func(height uint64, result chan ExtractResult) {
		//回收创建的地址
		for gets := range result {

			if gets.Success {

				notifyErr := bs.newExtractDataNotify(height, gets.extractData)
				//saveErr := bs.SaveRechargeToWalletDB(height, gets.Recharges)
				if notifyErr != nil {
					failed++ //标记保存失败数
					bs.wm.Log.Std.Info("newExtractDataNotify unexpected error: %v", notifyErr)
				}

			} else {
				//记录未扫区块
				unscanRecord := openwallet.NewUnscanRecord(height, "", "", bs.wm.Symbol())
				bs.SaveUnscanRecord(unscanRecord)
				bs.wm.Log.Std.Info("block height: %d extract failed.", height)
				failed++ //标记保存失败数
			}
			//累计完成的线程数
			done++
			if done == shouldDone {
				//bs.wm.Log.Std.Info("done = %d, shouldDone = %d ", done, len(txs))
				close(quit) //关闭通道，等于给通道传入nil
			}
		}
	}

	//提取工作
	extractWork := func(eblockHeight uint64, eBlockHash string, mTxs []*Transaction, eProducer chan ExtractResult) {
		for _, tx := range mTxs {
			bs.extractingCH <- struct{}{}
			//shouldDone++
			go func(mBlockHeight uint64, mTx *Transaction, end chan struct{}, mProducer chan<- ExtractResult) {

				//导出提出的交易
				mProducer <- bs.ExtractTransaction(mBlockHeight, eBlockHash, mTx, bs.ScanTargetFunc)
				//释放
				<-end

			}(eblockHeight, tx, bs.extractingCH, eProducer)
		}
	}

	/*	开启导出的线程	*/

	//独立线程运行消费
	go saveWork(blockHeight, worker)

	//独立线程运行生产
	go extractWork(blockHeight, blockHash, txs, producer)

	//以下使用生产消费模式
	bs.extractRuntime(producer, worker, quit)

	if failed > 0 {
		return fmt.Errorf("block scanner saveWork failed")
	} else {
		return nil
	}

	//return nil
}

//extractRuntime 提取运行时
func (bs *ETPBlockScanner) extractRuntime(producer chan ExtractResult, worker chan ExtractResult, quit chan struct{}) {

	var (
		values = make([]ExtractResult, 0)
	)

	for {

		var activeWorker chan<- ExtractResult
		var activeValue ExtractResult

		//当数据队列有数据时，释放顶部，传输给消费者
		if len(values) > 0 {
			activeWorker = worker
			activeValue = values[0]

		}

		select {

		//生成者不断生成数据，插入到数据队列尾部
		case pa := <-producer:
			values = append(values, pa)
		case <-quit:
			//退出
			//bs.wm.Log.Std.Info("block scanner have been scanned!")
			return
		case activeWorker <- activeValue:
			//wm.Log.Std.Info("Get %d", len(activeValue))
			values = values[1:]
		}
	}

}

//ExtractTransaction 提取交易单
func (bs *ETPBlockScanner) ExtractTransaction(blockHeight uint64, blockHash string, trx *Transaction, scanTargetFunc openwallet.BlockScanTargetFunc) ExtractResult {

	var (
		result = ExtractResult{
			BlockHeight: blockHeight,
			TxID:        trx.TxID,
			extractData: make(map[string]ExtractData),
		}
	)

	bs.extractTransaction(trx, &result, scanTargetFunc)

	return result

}

//ExtractTransactionData 提取交易单
func (bs *ETPBlockScanner) extractTransaction(trx *Transaction, result *ExtractResult, scanTargetFunc openwallet.BlockScanTargetFunc) {

	var (
		success = true
	)

	if trx == nil {
		//记录哪个区块哪个交易单没有完成扫描
		success = false
	} else {

		vin := trx.Vins
		blocktime := trx.Blocktime

		//检查交易单输入信息是否完整，不完整查上一笔交易单的输出填充数据
		for _, input := range vin {

			if input.isCoinbase {
				//coinbase skip
				success = true
				break
			}

			intxid := input.TxID
			vout := input.Vout
			preTx, txErr := bs.wm.GetTransaction(intxid)
			if txErr != nil {
				success = false
				break
			} else {
				preVouts := preTx.Vouts
				if len(preVouts) > int(vout) {
					preOut := preVouts[vout]
					input.Addr = preOut.Addr
					input.Value = preOut.Value
					input.AssetAttachment = preOut.AssetAttachment
					input.IsToken = preOut.IsToken

					success = true

					//bs.wm.Log.Debug("GetTxOut:", output[vout])

				}
			}

		}

		if success {

			//提取出账部分记录
			tokenExtractInput, from, totalSpent := bs.extractTxInput(trx, result, scanTargetFunc)
			//提取入账部分记录
			tokenExtractOutput, to, totalReceived := bs.extractTxOutput(trx, result, scanTargetFunc)
			//手续费
			fees := totalSpent.Sub(totalReceived)

			for token, sourceExtractInput := range tokenExtractInput {

				tokenFees := decimal.Zero
				decimals := int32(0)
				tokenFrom := from[token]
				tokenTo := to[token]
				if token == bs.wm.Symbol() {
					tokenFees = fees
					decimals = bs.wm.Decimal()
				}

				for sourceKey, extractInput := range sourceExtractInput {
					var (
						coin openwallet.Coin
					)
					for _, input := range extractInput {
						coin = input.Coin
					}

					sourceKeyExtractData := result.extractData[token]
					if sourceKeyExtractData == nil {
						sourceKeyExtractData = make(ExtractData)
					}

					extractData := sourceKeyExtractData[sourceKey]
					if extractData == nil {
						extractData = &openwallet.TxExtractData{}
					}

					extractData.TxInputs = extractInput
					if extractData.Transaction == nil {
						extractData.Transaction = &openwallet.Transaction{
							From:        tokenFrom,
							To:          tokenTo,
							Fees:        tokenFees.String(),
							Coin:        coin,
							BlockHash:   trx.BlockHash,
							BlockHeight: trx.BlockHeight,
							TxID:        trx.TxID,
							Decimal:     decimals,
							ConfirmTime: blocktime,
							Status:      openwallet.TxStatusSuccess,
							TxType:      0,
						}
						wxID := openwallet.GenTransactionWxID(extractData.Transaction)
						extractData.Transaction.WxID = wxID
					}

					sourceKeyExtractData[sourceKey] = extractData
					result.extractData[token] = sourceKeyExtractData
				}
			}

			for token, sourceExtractOutput := range tokenExtractOutput {

				tokenFees := decimal.Zero
				decimals := int32(0)
				tokenFrom := from[token]
				tokenTo := to[token]
				if token == bs.wm.Symbol() {
					tokenFees = fees
					decimals = bs.wm.Decimal()
				}

				for sourceKey, extractOutput := range sourceExtractOutput {
					var (
						coin openwallet.Coin
					)
					for _, output := range extractOutput {
						coin = output.Coin
					}

					sourceKeyExtractData := result.extractData[token]
					if sourceKeyExtractData == nil {
						sourceKeyExtractData = make(ExtractData)
					}

					extractData := sourceKeyExtractData[sourceKey]
					if extractData == nil {
						extractData = &openwallet.TxExtractData{}
					}

					extractData.TxOutputs = extractOutput
					if extractData.Transaction == nil {
						extractData.Transaction = &openwallet.Transaction{
							From:        tokenFrom,
							To:          tokenTo,
							Fees:        tokenFees.String(),
							Coin:        coin,
							BlockHash:   trx.BlockHash,
							BlockHeight: trx.BlockHeight,
							TxID:        trx.TxID,
							Decimal:     decimals,
							ConfirmTime: blocktime,
							Status:      openwallet.TxStatusSuccess,
							TxType:      0,
						}
						wxID := openwallet.GenTransactionWxID(extractData.Transaction)
						extractData.Transaction.WxID = wxID
					}

					sourceKeyExtractData[sourceKey] = extractData
					result.extractData[token] = sourceKeyExtractData
				}
			}

		}
	}
	result.Success = success
}

//ExtractTxInput 提取交易单输入部分
func (bs *ETPBlockScanner) extractTxInput(trx *Transaction, result *ExtractResult, scanTargetFunc openwallet.BlockScanTargetFunc) (map[string]ExtractInput, map[string][]string, decimal.Decimal) {

	//vin := trx.Get("vin")

	var (
		totalAmount       = decimal.Zero
		tokenExtractInput = make(map[string]ExtractInput)
		from              = make(map[string][]string)
	)

	createAt := time.Now().Unix()
	for i, output := range trx.Vins {

		//in := vin[i]

		txid := output.TxID
		vout := output.Vout

		amount, _ := decimal.NewFromString(output.Value)
		amount = amount.Shift(-bs.wm.Decimal())
		addr := output.Addr
		sourceKey, ok := scanTargetFunc(openwallet.ScanTarget{
			Address:          addr,
			Symbol:           bs.wm.Symbol(),
			BalanceModelType: openwallet.BalanceModelTypeAddress})
		if ok {

			//填充主币
			if output.IsToken {

				contractId := openwallet.GenContractID(bs.wm.Symbol(), output.AssetAttachment.Symbol)

				input := openwallet.TxInput{}
				input.SourceTxID = txid
				input.SourceIndex = vout
				input.TxID = result.TxID
				input.Address = addr
				//transaction.AccountID = a.AccountID
				input.Amount = output.AssetAttachment.Quantity
				input.Coin = openwallet.Coin{
					Symbol:     bs.wm.Symbol(),
					IsContract: true,
					ContractID: contractId,
					Contract: openwallet.SmartContract{
						ContractID: contractId,
						Address:    output.AssetAttachment.Symbol,
						Symbol:     bs.wm.Symbol(),
					},
				}
				input.Index = output.N
				input.Sid = openwallet.GenTxInputSID(txid, bs.wm.Symbol(), contractId, uint64(i))
				input.CreateAt = createAt
				//在哪个区块高度时消费
				input.BlockHeight = trx.BlockHeight
				input.BlockHash = trx.BlockHash

				sourceKeyExtractInput := tokenExtractInput[output.AssetAttachment.Symbol]
				if sourceKeyExtractInput == nil {
					sourceKeyExtractInput = make(ExtractInput)
				}

				extractInput := sourceKeyExtractInput[sourceKey]
				if extractInput == nil {
					extractInput = make([]*openwallet.TxInput, 0)
				}

				extractInput = append(extractInput, &input)

				sourceKeyExtractInput[sourceKey] = extractInput
				tokenExtractInput[output.AssetAttachment.Symbol] = sourceKeyExtractInput

			} else {

				input := openwallet.TxInput{}
				input.SourceTxID = txid
				input.SourceIndex = vout
				input.TxID = result.TxID
				input.Address = addr
				//transaction.AccountID = a.AccountID
				input.Amount = amount.String()
				input.Coin = openwallet.Coin{
					Symbol:     bs.wm.Symbol(),
					IsContract: false,
				}
				input.Index = output.N
				input.Sid = openwallet.GenTxInputSID(txid, bs.wm.Symbol(), "", uint64(i))
				//input.Sid = base64.StdEncoding.EncodeToString(crypto.SHA1([]byte(fmt.Sprintf("input_%s_%d_%s", result.TxID, i, addr))))
				input.CreateAt = createAt
				//在哪个区块高度时消费
				input.BlockHeight = trx.BlockHeight
				input.BlockHash = trx.BlockHash

				sourceKeyExtractInput := tokenExtractInput[bs.wm.Symbol()]
				if sourceKeyExtractInput == nil {
					sourceKeyExtractInput = make(ExtractInput)
				}

				extractInput := sourceKeyExtractInput[sourceKey]
				if extractInput == nil {
					extractInput = make([]*openwallet.TxInput, 0)
				}

				extractInput = append(extractInput, &input)

				sourceKeyExtractInput[sourceKey] = extractInput
				tokenExtractInput[bs.wm.Symbol()] = sourceKeyExtractInput

			}
		}

		if output.IsToken {
			af := from[output.AssetAttachment.Symbol]
			if af == nil {
				af = make([]string, 0)
			}
			af = append(af, addr+":"+output.AssetAttachment.Quantity)
			from[output.AssetAttachment.Symbol] = af
		} else {
			af := from[bs.wm.Symbol()]
			if af == nil {
				af = make([]string, 0)
			}
			af = append(af, addr+":"+amount.String())
			from[bs.wm.Symbol()] = af
		}

		totalAmount = totalAmount.Add(amount) //用于计算手续费

	}
	return tokenExtractInput, from, totalAmount
}

//ExtractTxInput 提取交易单输入部分
func (bs *ETPBlockScanner) extractTxOutput(trx *Transaction, result *ExtractResult, scanTargetFunc openwallet.BlockScanTargetFunc) (map[string]ExtractOutput, map[string][]string, decimal.Decimal) {

	var (
		totalAmount        = decimal.Zero
		tokenExtractOutput = make(map[string]ExtractOutput)
		to                 = make(map[string][]string)
	)

	//confirmations := trx.Confirmations
	vout := trx.Vouts
	txid := trx.TxID
	//bs.wm.Log.Debug("vout:", vout.Array())
	createAt := time.Now().Unix()
	for _, output := range vout {

		amount, _ := decimal.NewFromString(output.Value)
		amount = amount.Shift(-bs.wm.Decimal())
		n := output.N
		addr := output.Addr
		sourceKey, ok := scanTargetFunc(openwallet.ScanTarget{
			Address:          addr,
			Symbol:           bs.wm.Symbol(),
			BalanceModelType: openwallet.BalanceModelTypeAddress})
		if ok {

			if output.IsToken {

				contractId := openwallet.GenContractID(bs.wm.Symbol(), output.AssetAttachment.Symbol)

				outPut := openwallet.TxOutPut{}
				outPut.TxID = txid
				outPut.Address = addr
				outPut.Amount = output.AssetAttachment.Quantity
				outPut.Coin = openwallet.Coin{
					Symbol:     bs.wm.Symbol(),
					IsContract: true,
					ContractID: contractId,
					Contract: openwallet.SmartContract{
						ContractID: contractId,
						Address:    output.AssetAttachment.Symbol,
						Symbol:     bs.wm.Symbol(),
					},
				}
				outPut.Index = n
				outPut.Sid = openwallet.GenTxOutPutSID(txid, bs.wm.Symbol(), contractId, n)
				outPut.CreateAt = createAt
				//在哪个区块高度时消费
				outPut.BlockHeight = trx.BlockHeight
				outPut.BlockHash = trx.BlockHash

				sourceKeyExtractOutput := tokenExtractOutput[output.AssetAttachment.Symbol]
				if sourceKeyExtractOutput == nil {
					sourceKeyExtractOutput = make(ExtractOutput)
				}

				extractOutput := sourceKeyExtractOutput[sourceKey]
				if extractOutput == nil {
					extractOutput = make([]*openwallet.TxOutPut, 0)
				}

				extractOutput = append(extractOutput, &outPut)

				sourceKeyExtractOutput[sourceKey] = extractOutput
				tokenExtractOutput[output.AssetAttachment.Symbol] = sourceKeyExtractOutput

			} else {

				outPut := openwallet.TxOutPut{}
				outPut.TxID = txid
				outPut.Address = addr
				//transaction.AccountID = a.AccountID
				outPut.Amount = amount.String()
				outPut.Coin = openwallet.Coin{
					Symbol:     bs.wm.Symbol(),
					IsContract: false,
				}
				outPut.Index = n
				outPut.Sid = openwallet.GenTxOutPutSID(txid, bs.wm.Symbol(), "", n)
				outPut.CreateAt = createAt
				outPut.BlockHeight = trx.BlockHeight
				outPut.BlockHash = trx.BlockHash

				sourceKeyExtractOutput := tokenExtractOutput[bs.wm.Symbol()]
				if sourceKeyExtractOutput == nil {
					sourceKeyExtractOutput = make(ExtractOutput)
				}

				extractOutput := sourceKeyExtractOutput[sourceKey]
				if extractOutput == nil {
					extractOutput = make([]*openwallet.TxOutPut, 0)
				}

				extractOutput = append(extractOutput, &outPut)

				sourceKeyExtractOutput[sourceKey] = extractOutput
				tokenExtractOutput[bs.wm.Symbol()] = sourceKeyExtractOutput

			}

		}

		if output.IsToken {
			af := to[output.AssetAttachment.Symbol]
			if af == nil {
				af = make([]string, 0)
			}
			af = append(af, addr+":"+output.AssetAttachment.Quantity)
			to[output.AssetAttachment.Symbol] = af
		} else {
			af := to[bs.wm.Symbol()]
			if af == nil {
				af = make([]string, 0)
			}
			af = append(af, addr+":"+amount.String())
			to[bs.wm.Symbol()] = af
		}

		totalAmount = totalAmount.Add(amount)

	}

	return tokenExtractOutput, to, totalAmount
}

//newExtractDataNotify 发送通知
func (bs *ETPBlockScanner) newExtractDataNotify(height uint64, tokenExtractData map[string]ExtractData) error {

	for o, _ := range bs.Observers {

		for _, extractData := range tokenExtractData {
			for key, data := range extractData {
				bs.wm.Log.Infof("newExtractDataNotify txid: %s", data.Transaction.TxID)
				err := o.BlockExtractDataNotify(key, data)
				if err != nil {
					bs.wm.Log.Error("BlockExtractDataNotify unexpected error:", err)
					//记录未扫区块
					unscanRecord := openwallet.NewUnscanRecord(height, "", "ExtractData Notify failed.", bs.wm.Symbol())
					err = bs.SaveUnscanRecord(unscanRecord)
					if err != nil {
						bs.wm.Log.Std.Error("block height: %d, save unscan record failed. unexpected error: %v", height, err.Error())
					}

				}
			}
		}
	}

	return nil
}

//GetCurrentBlockHeader 获取当前区块高度
func (bs *ETPBlockScanner) GetCurrentBlockHeader() (*openwallet.BlockHeader, error) {

	header, err := bs.wm.GetBlockHeader()
	if err != nil {
		return nil, err
	}

	return header, nil
}

func (bs *ETPBlockScanner) GetGlobalMaxBlockHeight() uint64 {
	header, err := bs.wm.GetBlockHeader()
	if err != nil {
		bs.wm.Log.Std.Info("get global max block height error;unexpected error:%v", err)
		return 0
	}
	return header.Height
}

//GetScannedBlockHeight 获取已扫区块高度
func (bs *ETPBlockScanner) GetScannedBlockHeight() uint64 {
	localHeight, _, _ := bs.GetLocalBlockHead()
	return localHeight
}


func (bs *ETPBlockScanner) ExtractTransactionData(txid string, scanTargetFunc openwallet.BlockScanTargetFunc) (map[string][]*openwallet.TxExtractData, error) {

	trx, err := bs.wm.GetTransaction(txid)
	if err != nil {
		return nil, err
	}

	header, err := bs.wm.GetBlockHeader(trx.BlockHeight)
	if err != nil {
		return nil, err
	}

	result := bs.ExtractTransaction(header.Height, header.Hash, trx, scanTargetFunc)
	if !result.Success {
		return nil, fmt.Errorf("extract transaction failed")
	}
	extData := make(map[string][]*openwallet.TxExtractData)

	for _, extractData := range result.extractData {
		for key, data := range extractData {
			txs := extData[key]
			if txs == nil {
				txs = make([]*openwallet.TxExtractData, 0)
			}
			txs = append(txs, data)
			extData[key] = txs
		}
	}
	return extData, nil
}

////GetTxOut 获取交易单输出信息，用于追溯交易单输入源头
//func (wm *WalletManager) GetTxOut(txid string, vout uint64) (*Vout, error) {
//
//	if wm.Config.RPCServerType == RPCServerExplorer {
//		//return wm.getTxOutByExplorer(txid, vout)
//		return nil, nil
//	} else {
//		return wm.getTxOutByCore(txid, vout)
//	}
//}

//GetAssetsAccountBalanceByAddress 查询账户相关地址的交易记录
func (bs *ETPBlockScanner) GetBalanceByAddress(address ...string) ([]*openwallet.Balance, error) {

	addrBalanceArr := make([]*openwallet.Balance, 0)
	for _, addr := range address {

		obj := &openwallet.Balance{
			Symbol:           bs.wm.Symbol(),
			Address:          addr,
			Balance:          "0",
			UnconfirmBalance: "0",
			ConfirmBalance:   "0",
		}

		etpBalance, err := bs.wm.GetAddressETP(addr)
		if err == nil {
			available, _ := decimal.NewFromString(etpBalance.Available)
			available = available.Shift(-bs.wm.Decimal())
			obj.Balance = available.String()
			obj.ConfirmBalance = available.String()
		}

		addrBalanceArr = append(addrBalanceArr, obj)
	}

	return addrBalanceArr, nil

}


//SupportBlockchainDAI 支持外部设置区块链数据访问接口
//@optional
func (bs *ETPBlockScanner) SupportBlockchainDAI() bool {
	return true
}