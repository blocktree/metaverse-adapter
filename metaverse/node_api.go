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
	"github.com/blocktree/openwallet/v2/log"
	"github.com/blocktree/openwallet/v2/openwallet"
	"github.com/imroc/req"
	"github.com/tidwall/gjson"
)

type ClientInterface interface {
	Call(path string, request []interface{}) (*gjson.Result, error)
}

// A Client is a Bitcoin RPC client. It performs RPCs over HTTP using JSON
// request and responses. A Client must be configured with a secret token
// to authenticate with other Cores on the network.
type Client struct {
	BaseURL string
	Debug   bool
	client  *req.Req
}

func NewClient(url string, debug bool) *Client {
	c := Client{
		BaseURL: url,
		Debug:   debug,
	}

	api := req.New()
	c.client = api

	return &c
}

// Call calls a remote procedure on another node, specified by the path.
func (c *Client) Call(path string, request []interface{}) (*gjson.Result, *openwallet.Error) {

	var (
		body = make(map[string]interface{}, 0)
	)

	if c.client == nil {
		return nil, openwallet.Errorf(openwallet.ErrUnknownException, "API url is not setup. ")
	}

	authHeader := req.Header{
		"Accept":       "application/json",
		"Content-Type": "application/json",
	}

	if request == nil {
		request = []interface{}{}
	}

	//json-rpc
	body["jsonrpc"] = "2.0"
	body["id"] = "1"
	body["method"] = path
	body["params"] = request

	if c.Debug {
		log.Std.Info("Start Request API...")
	}

	r, err := c.client.Post(c.BaseURL, req.BodyJSON(&body), authHeader)

	if c.Debug {
		log.Std.Info("Request API Completed")
	}

	if c.Debug {
		log.Std.Info("%+v", r)
	}

	if err != nil {
		return nil, openwallet.ConvertError(err)
	}

	resp := gjson.ParseBytes(r.Bytes())
	respErr := isError(&resp)
	if respErr != nil {
		return nil, respErr
	}

	result := resp.Get("result")

	return &result, nil
}

//isError 是否报错
func isError(result *gjson.Result) *openwallet.Error {
	var (
		err *openwallet.Error
	)

	/*
		//failed 返回错误
		{
			"result": null,
			"error": {
				"code": -8,
				"message": "Block height out of range"
			},
			"id": "foo"
		}
	*/

	if !result.Get("error").IsObject() {

		if !result.Get("result").Exists() {
			return openwallet.Errorf(openwallet.ErrUnknownException, "Response is empty! ")
		}

		return nil
	}

	err = openwallet.Errorf(result.Get("error.code").Uint(), result.Get("error.message").String())

	return err
}
