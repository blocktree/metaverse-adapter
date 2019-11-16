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

package openwtester

import (
	"github.com/blocktree/openwallet/openw"
	"testing"

	"github.com/blocktree/openwallet/log"
	"github.com/blocktree/openwallet/openwallet"
)

func testGetAssetsAccountBalance(tm *openw.WalletManager, walletID, accountID string) {
	balance, err := tm.GetAssetsAccountBalance(testApp, walletID, accountID)
	if err != nil {
		log.Error("GetAssetsAccountBalance failed, unexpected error:", err)
		return
	}
	log.Info("balance:", balance)
}

func testGetAssetsAccountTokenBalance(tm *openw.WalletManager, walletID, accountID string, contract openwallet.SmartContract) {
	balance, err := tm.GetAssetsAccountTokenBalance(testApp, walletID, accountID, contract)
	if err != nil {
		log.Error("GetAssetsAccountTokenBalance failed, unexpected error:", err)
		return
	}
	log.Info("token balance:", balance.Balance)
}

func testCreateTransactionStep(tm *openw.WalletManager, walletID, accountID, to, amount, feeRate string, contract *openwallet.SmartContract) (*openwallet.RawTransaction, error) {

	//err := tm.RefreshAssetsAccountBalance(testApp, accountID)
	//if err != nil {
	//	log.Error("RefreshAssetsAccountBalance failed, unexpected error:", err)
	//	return nil, err
	//}

	rawTx, err := tm.CreateTransaction(testApp, walletID, accountID, amount, to, feeRate, "", contract)

	if err != nil {
		log.Error("CreateTransaction failed, unexpected error:", err)
		return nil, err
	}

	return rawTx, nil
}

func testCreateSummaryTransactionStep(
	tm *openw.WalletManager,
	walletID, accountID, summaryAddress, minTransfer, retainedBalance, feeRate string,
	start, limit int,
	contract *openwallet.SmartContract,
	feeSupportAccount *openwallet.FeesSupportAccount) ([]*openwallet.RawTransactionWithError, error) {

	rawTxArray, err := tm.CreateSummaryRawTransactionWithError(testApp, walletID, accountID, summaryAddress, minTransfer,
		retainedBalance, feeRate, start, limit, contract, feeSupportAccount)

	if err != nil {
		return nil, err
	}

	return rawTxArray, nil
}

func testSignTransactionStep(tm *openw.WalletManager, rawTx *openwallet.RawTransaction) (*openwallet.RawTransaction, error) {

	_, err := tm.SignTransaction(testApp, rawTx.Account.WalletID, rawTx.Account.AccountID, "12345678", rawTx)
	if err != nil {
		log.Error("SignTransaction failed, unexpected error:", err)
		return nil, err
	}

	log.Infof("rawTx: %+v", rawTx)
	return rawTx, nil
}

func testVerifyTransactionStep(tm *openw.WalletManager, rawTx *openwallet.RawTransaction) (*openwallet.RawTransaction, error) {

	//log.Info("rawTx.Signatures:", rawTx.Signatures)

	_, err := tm.VerifyTransaction(testApp, rawTx.Account.WalletID, rawTx.Account.AccountID, rawTx)
	if err != nil {
		log.Error("VerifyTransaction failed, unexpected error:", err)
		return nil, err
	}

	log.Infof("rawTx: %+v", rawTx)
	return rawTx, nil
}

func testSubmitTransactionStep(tm *openw.WalletManager, rawTx *openwallet.RawTransaction) (*openwallet.RawTransaction, error) {

	tx, err := tm.SubmitTransaction(testApp, rawTx.Account.WalletID, rawTx.Account.AccountID, rawTx)
	if err != nil {
		log.Error("SubmitTransaction failed, unexpected error:", err)
		return nil, err
	}

	log.Std.Info("tx: %+v", tx)
	log.Info("wxID:", tx.WxID)
	log.Info("txID:", rawTx.TxID)

	return rawTx, nil
}

func TestTransfer(t *testing.T) {

	addrs := []string{
		"M8zhymCjZD9ZzSR9skirEhJnNDEdcJBb6c",
		//"MC3byQPhQS9dQYkY4ME5R94j5GksWvDYTR",
		//"MDgW56oXUFMRfSuRhPcyoWcGwSyj81hUnM",
		//"MHr2w1nQ2aiGuh7McpAvi5TMvqmzVLJeNC",
		//"MMRbpJdtxXeNmdwRZa4JjNgraL2XKUeg4e",
		//"MNVJdDfesiRdPMWz1QCnyvAXTbTcfdaBun",

		//"MSz3Ca3SJGDezZRXDNJG5pHGgbxQktikce", //手续费地址

		//"MPZd2vDvaFGwPW1L8qjN2CguFs7ZSRkY6u",
	}


	tm := testInitWalletManager()
	walletID := "WHVMNrUKoKqAQ8zUDKTo5FsRczs3jcyBhQ"
	accountID := "74Yy2VDrRCWhzA7NZ3foYNCdykoPjdYmE9A2RabmxMjN"

	testGetAssetsAccountBalance(tm, walletID, accountID)

	for _, to := range addrs {

		rawTx, err := testCreateTransactionStep(tm, walletID, accountID, to, "1", "", nil)
		if err != nil {
			return
		}

		_, err = testSignTransactionStep(tm, rawTx)
		if err != nil {
			return
		}

		_, err = testVerifyTransactionStep(tm, rawTx)
		if err != nil {
			return
		}

		_, err = testSubmitTransactionStep(tm, rawTx)
		if err != nil {
			return
		}

	}



}


func TestTransfer_Token(t *testing.T) {

	addrs := []string{
		"M8zhymCjZD9ZzSR9skirEhJnNDEdcJBb6c",
		"MC3byQPhQS9dQYkY4ME5R94j5GksWvDYTR",
		"MDgW56oXUFMRfSuRhPcyoWcGwSyj81hUnM",
		"MHr2w1nQ2aiGuh7McpAvi5TMvqmzVLJeNC",
		"MMRbpJdtxXeNmdwRZa4JjNgraL2XKUeg4e",
		"MNVJdDfesiRdPMWz1QCnyvAXTbTcfdaBun",

		//"MSz3Ca3SJGDezZRXDNJG5pHGgbxQktikce", //手续费地址

		//"MC4DKZjDUrcP7TkSCCCsdMyDbExaEC4mEZ",
		//"M8fUEbid4PMM377EbvHbU2LGTLCXqa5qAY",
		//"MHgBzPAczeZ26c8FWDcF5qA5CLzJoW4iz4",
		//"MRRm8xqkikTbkKTL36f1D4VcgUHEBpLWLs",
		//"MRhvNDsWKLzR6N7kD7MmxnJDRgRiPYUXKa",
		//"MLXtP3pmEAAAfbPSFy64YrQ3CFo6m41cRL",
		//"MHSV8RrA7JYcbzKkdzt4rUfB7qj97XZvbB",
		//"MKADoYLgBKSFTeuAVP1SPDsUfKhNV8ffe6",
		//"MMRjfzsNHSmXMmrmvgZuPuKNWgc8hNhmoL",
		//"MEumHH5pfEVEUgK9G56uhBMFBuJVxLSKiB",
	}


	tm := testInitWalletManager()
	walletID := "WHVMNrUKoKqAQ8zUDKTo5FsRczs3jcyBhQ"
	accountID := "74Yy2VDrRCWhzA7NZ3foYNCdykoPjdYmE9A2RabmxMjN"

	contract := openwallet.SmartContract{
		Address:  "DNA",
		Symbol:   "ETP",
		Name:     "DNA",
		Token:    "DNA",
		Decimals: 4,
	}

	testGetAssetsAccountBalance(tm, walletID, accountID)

	testGetAssetsAccountTokenBalance(tm, walletID, accountID, contract)

	for _, to := range addrs {
		rawTx, err := testCreateTransactionStep(tm, walletID, accountID, to, "0.123", "", &contract)
		if err != nil {
			return
		}

		_, err = testSignTransactionStep(tm, rawTx)
		if err != nil {
			return
		}

		_, err = testVerifyTransactionStep(tm, rawTx)
		if err != nil {
			return
		}

		_, err = testSubmitTransactionStep(tm, rawTx)
		if err != nil {
			return
		}
	}
}


func TestSummary(t *testing.T) {
	tm := testInitWalletManager()
	walletID := "WCJCXnevTTBCPxfc2zS7kxCPLsH9S2Aqcf"
	accountID := "CDpf4PEZGWhbzevnRVTiDqACmnsrwJEKbzdSnpwwL1vz"
	summaryAddress := "MUsTC2PCF52yNvAeGNXJUKy9CfLVHV9yYj"

	//accountID := "2XWy8sjUxyn6zXqz3oeN9GZscGQ1pJ4dtTmmaFhdwiUa"

	testGetAssetsAccountBalance(tm, walletID, accountID)

	rawTxArray, err := testCreateSummaryTransactionStep(tm, walletID, accountID,
		summaryAddress, "", "", "",
		0, 100, nil, nil)
	if err != nil {
		log.Errorf("CreateSummaryTransaction failed, unexpected error: %v", err)
		return
	}

	//执行汇总交易
	for _, rawTxWithErr := range rawTxArray {

		if rawTxWithErr.Error != nil {
			log.Error(rawTxWithErr.Error.Error())
			continue
		}

		_, err = testSignTransactionStep(tm, rawTxWithErr.RawTx)
		if err != nil {
			return
		}

		_, err = testVerifyTransactionStep(tm, rawTxWithErr.RawTx)
		if err != nil {
			return
		}

		_, err = testSubmitTransactionStep(tm, rawTxWithErr.RawTx)
		if err != nil {
			return
		}
	}

}



func TestSummary_Token(t *testing.T) {

	tm := testInitWalletManager()
	walletID := "WCJCXnevTTBCPxfc2zS7kxCPLsH9S2Aqcf"
	accountID := "CDpf4PEZGWhbzevnRVTiDqACmnsrwJEKbzdSnpwwL1vz"
	summaryAddress := "MUsTC2PCF52yNvAeGNXJUKy9CfLVHV9yYj"

	contract := openwallet.SmartContract{
		Address:  "DNA",
		Symbol:   "ETP",
		Name:     "DNA",
		Token:    "DNA",
		Decimals: 4,
	}

	//address: MSz3Ca3SJGDezZRXDNJG5pHGgbxQktikce
	feesSupport := openwallet.FeesSupportAccount{
		AccountID: "J2RnFngmFXSTcW4eGaR19LaZGCzTrRuJshcyJQsfw7TF",
		FeesSupportScale: "1",
	}

	testGetAssetsAccountBalance(tm, walletID, accountID)

	testGetAssetsAccountTokenBalance(tm, walletID, accountID, contract)

	rawTxArray, err := testCreateSummaryTransactionStep(tm, walletID, accountID,
		summaryAddress, "", "", "",
		0, 200, &contract, &feesSupport)
	if err != nil {
		log.Errorf("CreateSummaryTransaction failed, unexpected error: %v", err)
		return
	}

	//执行汇总交易
	for _, rawTxWithErr := range rawTxArray {

		if rawTxWithErr.Error != nil {
			log.Error(rawTxWithErr.Error.Error())
			continue
		}

		_, err = testSignTransactionStep(tm, rawTxWithErr.RawTx)
		if err != nil {
			return
		}

		_, err = testVerifyTransactionStep(tm, rawTxWithErr.RawTx)
		if err != nil {
			return
		}

		_, err = testSubmitTransactionStep(tm, rawTxWithErr.RawTx)
		if err != nil {
			return
		}
	}

}