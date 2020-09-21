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
	"github.com/blocktree/go-owcdrivers/mateverseTransaction"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/shopspring/decimal"
	"time"
)

type TransactionDecoder struct {
	openwallet.TransactionDecoderBase
	wm *WalletManager //钱包管理者
}

//NewTransactionDecoder 交易单解析器
func NewTransactionDecoder(wm *WalletManager) *TransactionDecoder {
	decoder := TransactionDecoder{}
	decoder.wm = wm
	return &decoder
}

//CreateRawTransaction 创建交易单
func (decoder *TransactionDecoder) CreateRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	if rawTx.Coin.IsContract {
		return decoder.CreateTokenRawTransaction(wrapper, rawTx)
	} else {
		return decoder.CreateETPRawTransaction(wrapper, rawTx)
	}
}

//SignRawTransaction 签名交易单
func (decoder *TransactionDecoder) SignRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	return decoder.SignETPRawTransaction(wrapper, rawTx)
}

//VerifyRawTransaction 验证交易单，验证交易单并返回加入签名后的交易单
func (decoder *TransactionDecoder) VerifyRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {
	return decoder.VerifyETPRawTransaction(wrapper, rawTx)
}

//CreateSummaryRawTransaction 创建汇总交易，返回原始交易单数组
func (decoder *TransactionDecoder) CreateSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransaction, error) {
	var (
		rawTxWithErrArray []*openwallet.RawTransactionWithError
		rawTxArray        = make([]*openwallet.RawTransaction, 0)
		err               error
	)
	if sumRawTx.Coin.IsContract {
		rawTxWithErrArray, err = decoder.CreateTokenSummaryRawTransaction(wrapper, sumRawTx)
	} else {
		rawTxWithErrArray, err = decoder.CreateETPSummaryRawTransaction(wrapper, sumRawTx)
	}
	if err != nil {
		return nil, err
	}
	for _, rawTxWithErr := range rawTxWithErrArray {
		if rawTxWithErr.Error != nil {
			continue
		}
		rawTxArray = append(rawTxArray, rawTxWithErr.RawTx)
	}
	return rawTxArray, nil
}

//SendRawTransaction 广播交易单
func (decoder *TransactionDecoder) SubmitRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) (*openwallet.Transaction, error) {

	if len(rawTx.RawHex) == 0 {
		return nil, fmt.Errorf("transaction hex is empty")
	}

	if !rawTx.IsCompleted {
		return nil, fmt.Errorf("transaction is not completed validation")
	}

	txid, err := decoder.wm.SendRawTx(rawTx.RawHex)
	if err != nil {
		decoder.wm.Log.Warningf("[Sid: %s] submit raw hex: %s", rawTx.Sid, rawTx.RawHex)
		return nil, err
	}

	rawTx.TxID = txid
	rawTx.IsSubmit = true

	decimals := int32(0)
	fees := "0"
	if rawTx.Coin.IsContract {
		decimals = int32(rawTx.Coin.Contract.Decimals)
		fees = "0"
	} else {
		decimals = decoder.wm.Decimal()
		fees = rawTx.Fees
	}

	//记录一个交易单
	tx := &openwallet.Transaction{
		From:       rawTx.TxFrom,
		To:         rawTx.TxTo,
		Amount:     rawTx.TxAmount,
		Coin:       rawTx.Coin,
		TxID:       rawTx.TxID,
		Decimal:    decimals,
		AccountID:  rawTx.Account.AccountID,
		Fees:       fees,
		SubmitTime: time.Now().Unix(),
	}

	tx.WxID = openwallet.GenTransactionWxID(tx)

	return tx, nil
}

////////////////////////// BTC implement //////////////////////////

//CreateRawTransaction 创建交易单
func (decoder *TransactionDecoder) CreateETPRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		totalSend           = decimal.New(0, 0)
		fees                = decimal.New(0, 0)
		accountID           = rawTx.Account.AccountID
		destination         = ""
		availableETPBalance *ETPBalance
		limit               = 2000
		receivers           = make(map[string]string)
	)

	address, err := wrapper.GetAddressList(0, limit, "AccountID", rawTx.Account.AccountID)
	if err != nil {
		return err
	}

	if len(address) == 0 {
		return openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", accountID)
		//return fmt.Errorf("[%s] have not addresses", accountID)
	}

	if len(rawTx.To) == 0 {
		return fmt.Errorf("Receiver addresses is empty!")
	}

	//计算总发送金额
	for addr, amount := range rawTx.To {
		//totalSend, _ = decimal.NewFromString(amount)
		//totalSend = totalSend
		//destination = addr
		//break
		sendAmount, _ := decimal.NewFromString(amount)
		totalSend = totalSend.Add(sendAmount)
		receivers[addr] = sendAmount.Shift(decoder.wm.Decimal()).String()
	}

	if len(rawTx.FeeRate) == 0 {
		fees = decoder.wm.Config.MinFees
	} else {
		fees, _ = decimal.NewFromString(rawTx.FeeRate)
	}

	for _, addr := range address {
		etpBalance, etpErr := decoder.wm.GetAddressETP(addr.Address)
		if etpErr != nil {
			continue
		}

		available, _ := decimal.NewFromString(etpBalance.Available)
		available = available.Shift(-decoder.wm.Decimal())

		if available.LessThan(totalSend.Add(fees)) {
			continue
		}

		availableETPBalance = etpBalance
		break

	}

	if availableETPBalance == nil {
		return openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "the balance is not enough! ")
	}

	rawHex, txErr := decoder.wm.CreateRawTx(
		[]string{availableETPBalance.Address},
		receivers,
		//map[string]string{destination: totalSend.Shift(decoder.wm.Decimal()).String()},
		"",
		fees.Shift(decoder.wm.Decimal()).String(),
		"",
		false)
	if txErr != nil {
		return txErr
	}

	etpTx, txErr := decoder.wm.DecodeRawTx(rawHex)
	if txErr != nil {
		return txErr
	}

	decoder.wm.Log.Std.Notice("-----------------------------------------------")
	decoder.wm.Log.Std.Notice("From Account: %s", accountID)
	decoder.wm.Log.Std.Notice("To Address: %s", destination)
	decoder.wm.Log.Std.Notice("Fees: %v", fees)
	decoder.wm.Log.Std.Notice("Receive: %v", totalSend.String())
	decoder.wm.Log.Std.Notice("-----------------------------------------------")

	rawTx.Fees = fees.String()

	err = decoder.createRawTransaction(wrapper, rawTx, etpTx)
	if err != nil {
		return err
	}

	return nil
}

//SignRawTransaction 签名交易单
func (decoder *TransactionDecoder) SignETPRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	if rawTx.Signatures == nil || len(rawTx.Signatures) == 0 {
		return fmt.Errorf("transaction signature is empty")
	}

	key, err := wrapper.HDKey()
	if err != nil {
		return err
	}

	//keySignatures := rawTx.Signatures[rawTx.Account.AccountID]
	for accountID, keySignatures := range rawTx.Signatures {
		decoder.wm.Log.Debug("accountID:", accountID)
		if keySignatures != nil {
			for _, keySignature := range keySignatures {

				childKey, err := key.DerivedKeyWithPath(keySignature.Address.HDPath, keySignature.EccType)
				keyBytes, err := childKey.GetPrivateKeyBytes()
				if err != nil {
					return err
				}

				signature, err := mateverseTransaction.SignTransaction(keySignature.Message, keyBytes)
				if err != nil {
					return err
				}

				keySignature.Signature = signature
			}
		}

		rawTx.Signatures[accountID] = keySignatures
	}

	decoder.wm.Log.Info("transaction hash sign success")

	return nil
}

//VerifyRawTransaction 验证交易单，验证交易单并返回加入签名后的交易单
func (decoder *TransactionDecoder) VerifyETPRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		emptyTrans = rawTx.RawHex
		transHash  = make(map[string]*openwallet.KeySignature)
	)

	if rawTx.Signatures == nil || len(rawTx.Signatures) == 0 {
		//this.wm.Log.Std.Error("len of signatures error. ")
		return fmt.Errorf("transaction signature is empty")
	}

	inputs, err := mateverseTransaction.GetInputsFromEmptyRawTransaction(emptyTrans)
	if err != nil {
		return err
	}

	etpTx, txErr := decoder.wm.DecodeRawTx(emptyTrans)
	if txErr != nil {
		return txErr
	}

	for accountID, keySignatures := range rawTx.Signatures {
		decoder.wm.Log.Debug("accountID Signatures:", accountID)
		for _, keySignature := range keySignatures {

			transHash[keySignature.Message] = keySignature

			decoder.wm.Log.Debug("Message:", keySignature.Message)
			decoder.wm.Log.Debug("Signature:", keySignature.Signature)
			decoder.wm.Log.Debug("PublicKey:", keySignature.Address.PublicKey)
		}
	}

	for i, input := range inputs {
		input.SetLockScript(etpTx.Vins[i].LockScript)
	}

	err = mateverseTransaction.GetSigHash(emptyTrans, &inputs)
	if err != nil {
		return err
	}

	for _, input := range inputs {
		keySignature := transHash[input.GetHash()]
		if keySignature != nil {
			input.SetPubKey(keySignature.Address.PublicKey)
			input.SetSignature(keySignature.Signature)
		}
	}

	/////////验证交易单
	pass, signedTrans := mateverseTransaction.VerifyAndCombineTransaction(emptyTrans, inputs)
	if pass {
		decoder.wm.Log.Debug("transaction verify passed")
		rawTx.IsCompleted = true
		rawTx.RawHex = signedTrans
	} else {
		decoder.wm.Log.Debug("transaction verify failed")
		rawTx.IsCompleted = false
	}

	return nil
}

//GetRawTransactionFeeRate 获取交易单的费率
func (decoder *TransactionDecoder) GetRawTransactionFeeRate() (feeRate string, unit string, err error) {
	rate := decoder.wm.Config.MinFees
	return rate.String(), "TX", nil
}

//CreateETPSummaryRawTransaction 创建ETP汇总交易
func (decoder *TransactionDecoder) CreateETPSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {

	var (
		fees                 = decimal.New(0, 0)
		accountID            = sumRawTx.Account.AccountID
		minTransfer, _       = decimal.NewFromString(sumRawTx.MinTransfer)
		rawTxArray           = make([]*openwallet.RawTransactionWithError, 0)
		availableETPBalances = make([]*openwallet.Balance, 0)
		sumAmount            = decimal.Zero
	)

	address, err := wrapper.GetAddressList(sumRawTx.AddressStartIndex, sumRawTx.AddressLimit, "AccountID", sumRawTx.Account.AccountID)
	if err != nil {
		return nil, err
	}

	if len(address) == 0 {
		return nil, fmt.Errorf("[%s] have not addresses", accountID)
	}

	searchAddrs := make([]string, 0)
	for _, address := range address {
		searchAddrs = append(searchAddrs, address.Address)
	}

	addrBalanceArray, err := decoder.wm.Blockscanner.GetBalanceByAddress(searchAddrs...)
	if err != nil {
		return nil, err
	}

	//取得费率
	if len(sumRawTx.FeeRate) == 0 {
		fees = decoder.wm.Config.MinFees
	} else {
		fees, _ = decimal.NewFromString(sumRawTx.FeeRate)
	}

	for _, addrBalance := range addrBalanceArray {
		addrBalance_dec, _ := decimal.NewFromString(addrBalance.Balance)
		if addrBalance_dec.LessThan(minTransfer) || addrBalance_dec.LessThanOrEqual(decimal.Zero) {
			continue
		}

		sumAmount = sumAmount.Add(addrBalance_dec)
		availableETPBalances = append(availableETPBalances, addrBalance)
	}

	sumAmount = sumAmount.Sub(fees)

	decoder.wm.Log.Debugf("fees: %v", fees)
	decoder.wm.Log.Debugf("sumAmount: %v", sumAmount)

	if sumAmount.GreaterThan(decimal.Zero) {

		//创建一笔交易单
		rawTx := &openwallet.RawTransaction{
			Coin:     sumRawTx.Coin,
			Account:  sumRawTx.Account,
			FeeRate:  sumRawTx.FeeRate,
			To:       map[string]string{sumRawTx.SummaryAddress: sumAmount.String()},
			Fees:     fees.StringFixed(decoder.wm.Decimal()),
			Required: 1,
		}

		senders := make([]string, 0)
		for _, addr := range availableETPBalances {
			senders = append(senders, addr.Address)
		}

		rawHex, txErr := decoder.wm.CreateRawTx(
			senders,
			map[string]string{sumRawTx.SummaryAddress: sumAmount.Shift(decoder.wm.Decimal()).String()},
			sumRawTx.SummaryAddress,
			fees.Shift(decoder.wm.Decimal()).String(),
			"",
			false)
		if txErr != nil {
			return rawTxArray, nil
		}

		etpTx, txErr := decoder.wm.DecodeRawTx(rawHex)
		if txErr != nil {
			return nil, txErr
		}

		createErr := decoder.createRawTransaction(wrapper, rawTx, etpTx)
		if createErr != nil {
			return nil, createErr
		}

		rawTxWithErr := &openwallet.RawTransactionWithError{
			RawTx: rawTx,
			Error: openwallet.ConvertError(createErr),
		}

		//创建成功，添加到队列
		rawTxArray = append(rawTxArray, rawTxWithErr)

	}

	return rawTxArray, nil
}

//createRawTransaction 创建原始交易单
func (decoder *TransactionDecoder) createRawTransaction(
	wrapper openwallet.WalletDAI,
	rawTx *openwallet.RawTransaction,
	etpTx *Transaction,
) error {

	var (
		err              error
		accountTotalSent = decimal.Zero
		txFrom           = make([]string, 0)
		txTo             = make([]string, 0)
		accountID        = rawTx.Account.AccountID
	)

	isToken := rawTx.Coin.IsContract

	//计算总发送金额
	for _, output := range etpTx.Vouts {

		amount := decimal.Zero
		if isToken {
			amount, _ = decimal.NewFromString(output.AssetAttachment.Quantity)
		} else {
			//主币需要计算好精度
			amount, _ = decimal.NewFromString(output.Value)
			amount = amount.Shift(-decoder.wm.Decimal())
		}

		txTo = append(txTo, fmt.Sprintf("%s:%s", output.Addr, amount.String()))
		//计算账户的实际转账amount
		addresses, findErr := wrapper.GetAddressList(0, -1, "AccountID", accountID, "Address", output.Addr)
		if findErr != nil || len(addresses) == 0 {
			//amountDec, _ := decimal.NewFromString(amount)
			accountTotalSent = accountTotalSent.Add(amount)
		}
	}

	inputs, err := mateverseTransaction.GetInputsFromEmptyRawTransaction(etpTx.RawHex)
	if err != nil {
		return err
	}

	if len(etpTx.Vins) != len(inputs) {
		errStr := "inputs from raw hex is not equal to tx vins"
		decoder.wm.Log.Errorf(errStr)
		return fmt.Errorf(errStr)
	}

	//装配输入
	for i, input := range etpTx.Vins {

		amount := decimal.Zero
		if isToken {
			amount, _ = decimal.NewFromString(input.AssetAttachment.Quantity)
		} else {
			//主币需要计算好精度
			amount, _ = decimal.NewFromString(input.Value)
			amount = amount.Shift(-decoder.wm.Decimal())
		}

		txFrom = append(txFrom, fmt.Sprintf("%s:%s", input.Addr, amount.String()))

		// 设定锁定脚本
		inputs[i].SetLockScript(input.LockScript)
	}

	// 2 . 获取待签哈希
	err = mateverseTransaction.GetSigHash(etpTx.RawHex, &inputs)
	if err != nil {
		return err
	}

	rawTx.RawHex = etpTx.RawHex

	if rawTx.Signatures == nil {
		rawTx.Signatures = make(map[string][]*openwallet.KeySignature)
	}

	//装配签名
	signatures := rawTx.Signatures
	if signatures == nil {
		signatures = make(map[string][]*openwallet.KeySignature)
	}

	for i, input := range etpTx.Vins {

		//获取hash值
		beSignHex := inputs[i].GetHash()

		decoder.wm.Log.Std.Debug("txHash[%d]: %s", i, beSignHex)
		//beSignHex := transHash[i]

		addr, err := wrapper.GetAddress(input.Addr)
		if err != nil {
			return err
		}

		signature := &openwallet.KeySignature{
			EccType: decoder.wm.Config.CurveType,
			Nonce:   "",
			Address: addr,
			Message: beSignHex,
		}

		keySigs := signatures[addr.AccountID]
		if keySigs == nil {
			keySigs = make([]*openwallet.KeySignature, 0)
		}

		//装配签名
		keySigs = append(keySigs, signature)

		signatures[addr.AccountID] = keySigs

	}

	feesDec, _ := decimal.NewFromString(rawTx.Fees)
	accountTotalSent = accountTotalSent.Add(feesDec)
	accountTotalSent = decimal.Zero.Sub(accountTotalSent)

	rawTx.Signatures = signatures
	rawTx.IsBuilt = true
	rawTx.TxAmount = accountTotalSent.StringFixed(decoder.wm.Decimal())
	rawTx.TxFrom = txFrom
	rawTx.TxTo = txTo

	return nil
}

////////////////////////// tokencore implement //////////////////////////

//CreateTokenRawTransaction 创建Token交易单
func (decoder *TransactionDecoder) CreateTokenRawTransaction(wrapper openwallet.WalletDAI, rawTx *openwallet.RawTransaction) error {

	var (
		totalSend             = decimal.New(0, 0)
		fees                  = decimal.New(0, 0)
		accountID             = rawTx.Account.AccountID
		destination           = ""
		availableETPBalance   *ETPBalance
		availableTokenBalance *TokenBalance
		limit                 = 2000
		receivers             = make(map[string]string)
	)

	tokenAddress := rawTx.Coin.Contract.Address
	tokenDecimals := int32(rawTx.Coin.Contract.Decimals)

	address, err := wrapper.GetAddressList(0, limit, "AccountID", rawTx.Account.AccountID)
	if err != nil {
		return err
	}

	if len(address) == 0 {
		return openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", accountID)
		//return fmt.Errorf("[%s] have not addresses", accountID)
	}

	if len(rawTx.To) == 0 {
		return fmt.Errorf("Receiver addresses is empty!")
	}

	//计算总发送金额
	for addr, amount := range rawTx.To {
		//totalSend, _ = decimal.NewFromString(amount)
		//totalSend = totalSend
		//destination = addr
		//break
		sendAmount, _ := decimal.NewFromString(amount)
		totalSend = totalSend.Add(sendAmount)
		receivers[addr] = sendAmount.Shift(tokenDecimals).String()
	}

	if len(rawTx.FeeRate) == 0 {
		fees = decoder.wm.Config.MinFees
	} else {
		fees, _ = decimal.NewFromString(rawTx.FeeRate)
	}

	for _, addr := range address {
		etpBalance, etpErr := decoder.wm.GetAddressETP(addr.Address)
		if etpErr != nil {
			continue
		}

		available, _ := decimal.NewFromString(etpBalance.Available)
		available = available.Shift(-decoder.wm.Decimal())

		if available.LessThan(fees) {
			continue
		}

		tokenBalance, etpErr := decoder.wm.GetAddressAsset(addr.Address, tokenAddress)
		if etpErr != nil {
			continue
		}

		availableToken, _ := decimal.NewFromString(tokenBalance.Quantity)
		availableToken = availableToken.Shift(-tokenDecimals)

		if availableToken.LessThan(totalSend) {
			continue
		}

		availableETPBalance = etpBalance
		availableTokenBalance = tokenBalance
		break

	}

	if availableTokenBalance == nil {
		return openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "the token balance is not enough! ")
	}

	if availableETPBalance == nil {
		return openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "the %s balance is not enough! ", decoder.wm.Symbol())
	}

	rawHex, txErr := decoder.wm.CreateRawTx(
		[]string{availableTokenBalance.Address},
		receivers,
		//map[string]string{destination: totalSend.Shift(tokenDecimals).String()},
		"",
		fees.Shift(decoder.wm.Decimal()).String(),
		tokenAddress,
		true)
	if txErr != nil {
		return txErr
	}

	etpTx, txErr := decoder.wm.DecodeRawTx(rawHex)
	if txErr != nil {
		return txErr
	}

	decoder.wm.Log.Std.Notice("-----------------------------------------------")
	decoder.wm.Log.Std.Notice("From Account: %s", accountID)
	decoder.wm.Log.Std.Notice("To Address: %s", destination)
	decoder.wm.Log.Std.Notice("Token Address: %s", tokenAddress)
	decoder.wm.Log.Std.Notice("%v  Fees: %v", fees, decoder.wm.Symbol())
	decoder.wm.Log.Std.Notice("Receive: %v", totalSend.String())
	decoder.wm.Log.Std.Notice("-----------------------------------------------")

	rawTx.Fees = "0"

	err = decoder.createRawTransaction(wrapper, rawTx, etpTx)
	if err != nil {
		return err
	}

	return nil
}

//CreateTokenSummaryRawTransaction 创建Token汇总交易
func (decoder *TransactionDecoder) CreateTokenSummaryRawTransaction(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {

	var (
		fees                     = decimal.New(0, 0)
		accountID                = sumRawTx.Account.AccountID
		minTransfer, _           = decimal.NewFromString(sumRawTx.MinTransfer)
		rawTxArray               = make([]*openwallet.RawTransactionWithError, 0)
		availableTokenBalances   = make([]*TokenBalance, 0)
		sumAmount                = decimal.Zero
		feesSupportETPBalance    *ETPBalance
		feesSupportTokenBalances []*TokenBalance
		change                   string
	)

	//代币编号
	tokenAddress := sumRawTx.Coin.Contract.Address
	tokenDecimals := int32(sumRawTx.Coin.Contract.Decimals)

	if len(sumRawTx.FeeRate) == 0 {
		fees = decoder.wm.Config.MinFees
	} else {
		fees, _ = decimal.NewFromString(sumRawTx.FeeRate)
	}

	// 如果有提供手续费账户，检查账户是否存在
	if feesAcount := sumRawTx.FeesSupportAccount; feesAcount != nil {

		etpBalance, tokenBalances, getErr := decoder.getFeeSupportAccountAvailableETPAndTokens(wrapper, feesAcount.AccountID, fees, tokenAddress)
		if getErr != nil {
			return nil, getErr
		}

		feesSupportETPBalance = etpBalance
		feesSupportTokenBalances = tokenBalances

		if feesSupportETPBalance != nil {
			change = feesSupportETPBalance.Address
		}
	}

	if len(sumRawTx.Coin.Contract.Address) == 0 {
		return nil, fmt.Errorf("contract address is empty")
	}

	address, err := wrapper.GetAddressList(sumRawTx.AddressStartIndex, sumRawTx.AddressLimit, "AccountID", sumRawTx.Account.AccountID)
	if err != nil {
		return nil, err
	}

	if len(address) == 0 {
		return nil, fmt.Errorf("[%s] have not addresses", accountID)
	}

	for _, addr := range address {

		etpBalance, _ := decoder.wm.GetAddressETP(addr.Address)
		if etpBalance != nil {
			available, _ := decimal.NewFromString(etpBalance.Available)
			available = available.Shift(-decoder.wm.Decimal())
			if available.GreaterThan(fees) {
				feesSupportETPBalance = etpBalance
				change = sumRawTx.SummaryAddress
			}
		}

		tokenBalance, etpErr := decoder.wm.GetAddressAsset(addr.Address, tokenAddress)
		if etpErr != nil {
			continue
		}

		availableToken, _ := decimal.NewFromString(tokenBalance.Quantity)
		availableToken = availableToken.Shift(-tokenDecimals)

		if availableToken.LessThan(minTransfer) || availableToken.LessThanOrEqual(decimal.Zero) {
			continue
		}

		availableTokenBalances = append(availableTokenBalances, tokenBalance)
		sumAmount = sumAmount.Add(availableToken)

	}

	//如果手续费支持账户有代币也进行汇总
	if feesSupportTokenBalances != nil {
		for _, tokenBalance := range feesSupportTokenBalances {

			availableToken, _ := decimal.NewFromString(tokenBalance.Quantity)
			availableToken = availableToken.Shift(-tokenDecimals)

			availableTokenBalances = append(availableTokenBalances, tokenBalance)
			sumAmount = sumAmount.Add(availableToken)
		}
	}

	//没有token汇总
	if sumAmount.LessThanOrEqual(decimal.Zero) {
		return rawTxArray, nil
	}

	senders := make([]string, 0)
	for _, addr := range availableTokenBalances {
		senders = append(senders, addr.Address)
	}

	//没有足够的手续费支持
	if feesSupportETPBalance == nil {
		return nil, openwallet.Errorf(openwallet.ErrInsufficientFees, "the %s balance is not enough to pay fees", decoder.wm.Symbol())
	}

	senders = append(senders, feesSupportETPBalance.Address)

	rawHex, txErr := decoder.wm.CreateRawTx(
		senders,
		map[string]string{sumRawTx.SummaryAddress: sumAmount.Shift(tokenDecimals).String()},
		change,
		fees.Shift(decoder.wm.Decimal()).String(),
		tokenAddress,
		true)
	if txErr != nil {
		return nil, txErr
	}

	etpTx, txErr := decoder.wm.DecodeRawTx(rawHex)
	if txErr != nil {
		return nil, txErr
	}

	decoder.wm.Log.Debugf("%s fees: %v", decoder.wm.Symbol(), fees)
	decoder.wm.Log.Debugf("sumTokenAmount: %v", sumAmount.String())

	//创建一笔交易单
	rawTx := &openwallet.RawTransaction{
		Coin:     sumRawTx.Coin,
		Account:  sumRawTx.Account,
		To:       map[string]string{sumRawTx.SummaryAddress: sumAmount.String()},
		Fees:     "0",
		Required: 1,
	}

	createTxErr := decoder.createRawTransaction(wrapper, rawTx, etpTx)
	rawTxWithErr := &openwallet.RawTransactionWithError{
		RawTx: rawTx,
		Error: openwallet.ConvertError(createTxErr),
	}

	//创建成功，添加到队列
	rawTxArray = append(rawTxArray, rawTxWithErr)

	return rawTxArray, nil
}

// CreateSummaryRawTransactionWithError 创建汇总交易，返回能原始交易单数组（包含带错误的原始交易单）
func (decoder *TransactionDecoder) CreateSummaryRawTransactionWithError(wrapper openwallet.WalletDAI, sumRawTx *openwallet.SummaryRawTransaction) ([]*openwallet.RawTransactionWithError, error) {
	if sumRawTx.Coin.IsContract {
		return decoder.CreateTokenSummaryRawTransaction(wrapper, sumRawTx)
	} else {
		return decoder.CreateETPSummaryRawTransaction(wrapper, sumRawTx)
	}
}

//getFeeSupportAccountAvailableETP
func (decoder *TransactionDecoder) getFeeSupportAccountAvailableETPAndTokens(wrapper openwallet.WalletDAI, accountID string, fees decimal.Decimal, tokenAddress string) (*ETPBalance, []*TokenBalance, *openwallet.Error) {

	var (
		availableETP    *ETPBalance
		availableTokens = make([]*TokenBalance, 0)
	)

	address, err := wrapper.GetAddressList(0, -1, "AccountID", accountID)
	if err != nil {
		return nil, nil, openwallet.Errorf(openwallet.ErrAccountNotAddress, err.Error())
	}

	if len(address) == 0 {
		return nil, nil, openwallet.Errorf(openwallet.ErrAccountNotAddress, "[%s] have not addresses", accountID)
	}

	for _, addr := range address {
		etpBalance, _ := decoder.wm.GetAddressETP(addr.Address)
		if etpBalance != nil {
			available, _ := decimal.NewFromString(etpBalance.Available)
			available = available.Shift(-decoder.wm.Decimal())

			if available.GreaterThanOrEqual(fees) {
				availableETP = etpBalance
			}
		}

		tokenBalance, _ := decoder.wm.GetAddressAsset(addr.Address, tokenAddress)
		if tokenBalance != nil {
			//TODO: 大于0的才添加
			tokenAmount, _ := decimal.NewFromString(tokenBalance.Quantity)
			if tokenAmount.GreaterThan(decimal.Zero) {
				availableTokens = append(availableTokens, tokenBalance)
			}
		}
	}

	return availableETP, availableTokens, nil
}

//// getAssetsAccountUnspentSatisfyAmount
//func (decoder *TransactionDecoder) getUTXOSatisfyAmount(unspents []*Unspent, amount decimal.Decimal) (*Unspent, *openwallet.Error) {
//
//	var utxo *Unspent
//
//	if unspents != nil {
//		for _, u := range unspents {
//			if u.Spendable {
//				ua, _ := decimal.NewFromString(u.Amount)
//				if ua.GreaterThanOrEqual(amount) {
//					utxo = u
//					break
//				}
//			}
//		}
//	}
//
//	if utxo == nil {
//		return nil, openwallet.Errorf(openwallet.ErrInsufficientBalanceOfAccount, "account have not available utxo")
//	}
//
//	return utxo, nil
//}
//
//// removeUTXO
//func removeUTXO(slice []*Unspent, elem *Unspent) []*Unspent {
//	if len(slice) == 0 {
//		return slice
//	}
//	for i, v := range slice {
//		if v == elem {
//			slice = append(slice[:i], slice[i+1:]...)
//			return removeUTXO(slice, elem)
//			break
//		}
//	}
//	return slice
//}
//
//func appendOutput(output map[string]decimal.Decimal, address string, amount decimal.Decimal) map[string]decimal.Decimal {
//	if origin, ok := output[address]; ok {
//		origin = origin.Add(amount)
//		output[address] = origin
//	} else {
//		output[address] = amount
//	}
//	return output
//}
//
////根据交易输入地址顺序重排交易hash
//func resetTransHashFunc(origins []cxcTransaction.TxHash, addr string, start int) []cxcTransaction.TxHash {
//	newHashs := make([]cxcTransaction.TxHash, start)
//	copy(newHashs, origins[:start])
//	end := 0
//	for i := start; i < len(origins); i++ {
//		h := origins[i]
//		txAddress := h.GetNormalTxAddress()
//		if txAddress == addr {
//			newHashs = append(newHashs, h)
//			end = i
//			break
//		}
//	}
//
//	newHashs = append(newHashs, origins[start:end]...)
//	newHashs = append(newHashs, origins[end+1:]...)
//	return newHashs
//}
