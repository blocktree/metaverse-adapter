package metaverse_addrdec

import (
	"github.com/blocktree/go-owaddress"
	"github.com/blocktree/go-owcdrivers/addressEncoder"
)

var (
	alphabet = addressEncoder.BTCAlphabet
)

var (

	ETP_mainnetAddressP2PKH         = addressEncoder.AddressType{"base58", alphabet, "doubleSHA256", "h160", 20, []byte{0x32}, nil}
	ETP_testnetAddressP2PKH         = addressEncoder.AddressType{"base58", alphabet, "doubleSHA256", "h160", 20, []byte{0x7f}, nil}
	ETP_mainnetAddressP2SH          = addressEncoder.AddressType{"base58", alphabet, "doubleSHA256", "h160", 20, []byte{0x05}, nil}
	ETP_testnetAddressP2SH          = addressEncoder.AddressType{"base58", alphabet, "doubleSHA256", "h160", 20, []byte{0xc4}, nil}

	Default = AddressDecoderV2{}
)

//AddressDecoderV2
type AddressDecoderV2 struct {
	IsTestNet bool
}

//AddressDecode 地址解析
func (dec *AddressDecoderV2) AddressDecode(addr string, opts ...interface{}) ([]byte, error) {

	cfg := ETP_mainnetAddressP2PKH
	if dec.IsTestNet {
		cfg = ETP_testnetAddressP2PKH
	}

	if len(opts) > 0 {
		for _, opt := range opts {
			if at, ok := opt.(addressEncoder.AddressType); ok {
				cfg = at
			}
		}
	}

	return addressEncoder.AddressDecode(addr, cfg)
}

//AddressEncode 地址编码
func (dec *AddressDecoderV2) AddressEncode(hash []byte, opts ...interface{}) (string, error) {

	cfg := ETP_mainnetAddressP2PKH
	if dec.IsTestNet {
		cfg = ETP_testnetAddressP2PKH
	}

	if len(opts) > 0 {
		for _, opt := range opts {
			if at, ok := opt.(addressEncoder.AddressType); ok {
				cfg = at
			}
		}
	}

	address := addressEncoder.AddressEncode(hash, cfg)
	return address, nil
}


// AddressVerify 地址校验
func (dec *AddressDecoderV2) AddressVerify(address string, opts ...interface{}) bool {
	valid, err := owaddress.Verify("etp", address)
	if err != nil {
		return false
	}
	return valid
}