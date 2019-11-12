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
		log.Error("CreateSummaryTransaction failed, unexpected error:", err)
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
		//"122CsmrZzGBecM5R456yCb7kMRo1aFUBUT",
		//"15dsWRHfrzkssRXvNtQQ28Y6sQVvfNSJhG",
		//"16ZpDQ45bo4kavBzsaD9GHjjnRuwJPQHuf",
		//"1D65CTVmVv4pmpAy65P8ZQKwWuqQBxHB4V",
		//"1KLSHGtEG5xk18T2Vzegkzu3qa5svPPLKb",
		//"1L1ayirgcx9o68XzSX4WcF9ZsPGLmUvRyR",

		"1BuHkUX3aLQtVgUgpKn1ddVafw8x8q8wTd",
	}


	tm := testInitWalletManager()
	walletID := "WCJCXnevTTBCPxfc2zS7kxCPLsH9S2Aqcf"
	accountID := "7XB8PtHt41sh7rEVdRChiPC8rhDqY2LpuqXfUt9xoqw4"
	//to := "1DdtdoAQcL21Kwa6xKGVmWFVuxi3j7Jjyo"

	testGetAssetsAccountBalance(tm, walletID, accountID)

	for _, to := range addrs {

		rawTx, err := testCreateTransactionStep(tm, walletID, accountID, to, "0.01", "", nil)
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
		//"122CsmrZzGBecM5R456yCb7kMRo1aFUBUT",
		//"15dsWRHfrzkssRXvNtQQ28Y6sQVvfNSJhG",
		//"16ZpDQ45bo4kavBzsaD9GHjjnRuwJPQHuf",
		//"1D65CTVmVv4pmpAy65P8ZQKwWuqQBxHB4V",
		//"1KLSHGtEG5xk18T2Vzegkzu3qa5svPPLKb",
		//"1L1ayirgcx9o68XzSX4WcF9ZsPGLmUvRyR",

		"1BuHkUX3aLQtVgUgpKn1ddVafw8x8q8wTd",
	}


	tm := testInitWalletManager()
	walletID := "WCJCXnevTTBCPxfc2zS7kxCPLsH9S2Aqcf"
	accountID := "7XB8PtHt41sh7rEVdRChiPC8rhDqY2LpuqXfUt9xoqw4"
	//accountID := "E9YzesiKWYYcddLyM87xjEMjRLPW5VJ6LzeMpSLsGRrq"
	//to := "1MasZiznUuxPNYCNGmSoesp1TqoNJCi4tC"

	contract := openwallet.SmartContract{
		Address:  "583244-1358-57077",
		Symbol:   "ETP",
		Name:     "CPC",
		Token:    "CPC",
		Decimals: 6,
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
	accountID := "E9YzesiKWYYcddLyM87xjEMjRLPW5VJ6LzeMpSLsGRrq"
	summaryAddress := "1MasZiznUuxPNYCNGmSoesp1TqoNJCi4tC"

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
	accountID := "E9YzesiKWYYcddLyM87xjEMjRLPW5VJ6LzeMpSLsGRrq"
	summaryAddress := "1MasZiznUuxPNYCNGmSoesp1TqoNJCi4tC"

	//walletID := "WAmTnvPKMWpJBqKk6cncFG3mTXz3iPmtzV"
	//accountID := "86uUBCjk4SqEtMGDt92SQfn7YLhCZEcNQGjD5GhNNtSa"
	//summaryAddress := "12kSR8J11Q1d8JiYwZn7DZsPoDoptME35y"

	contract := openwallet.SmartContract{
		Address:  "583244-1358-57077",
		Symbol:   "ETP",
		Name:     "CPC",
		Token:    "CPC",
		Decimals: 6,
	}

	//address: 1PGx3n1w1QR9UJ1t3U5dVDBfKEUYzcStuB
	feesSupport := openwallet.FeesSupportAccount{
		AccountID: "26vsBY44zmJXVAHT5w1byPJArvbhUGCvqbBArh9VwGXv",
		//AccountID: "21Vn4NEmXT6DRy2EfdPTAJCS2kYTACTuconBer8AQ1cz",
		//FixSupportAmount: "0.01",
		FeesSupportScale: "1",
	}

	testGetAssetsAccountBalance(tm, walletID, accountID)

	testGetAssetsAccountTokenBalance(tm, walletID, accountID, contract)

	rawTxArray, err := testCreateSummaryTransactionStep(tm, walletID, accountID,
		summaryAddress, "", "", "",
		0, 20, &contract, &feesSupport)
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