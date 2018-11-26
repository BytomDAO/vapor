package common

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/vapor/common/bech32"
	"github.com/vapor/consensus"
)

func TestAddresses(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		encoded string
		valid   bool
		result  Address
		f       func() (Address, error)
		net     *consensus.Params
	}{
		// Segwit address tests.
		{
			name:    "segwit mainnet p2wpkh v0",
			addr:    "VBM1QZAPS6230EM2WVR4STEMF46DX2GK9V0LXZ9WEEL",
			encoded: "vbm1qzaps6230em2wvr4stemf46dx2gk9v0lxz9weel",
			valid:   true,
			result: tstAddressWitnessPubKeyHash(
				0,
				[20]byte{
					0x17, 0x43, 0x0d, 0x2a, 0x2f, 0xce, 0xd4, 0xe6, 0x0e, 0xb0,
					0x5e, 0x76, 0x9a, 0xe9, 0xa6, 0x52, 0x2c, 0x56, 0x3f, 0xe6},
				consensus.MainNetParams.Bech32HRPSegwit),
			f: func() (Address, error) {
				pkHash := []byte{
					0x17, 0x43, 0x0d, 0x2a, 0x2f, 0xce, 0xd4, 0xe6, 0x0e, 0xb0,
					0x5e, 0x76, 0x9a, 0xe9, 0xa6, 0x52, 0x2c, 0x56, 0x3f, 0xe6}
				return NewAddressWitnessPubKeyHash(pkHash, &consensus.MainNetParams)
			},
			net: &consensus.MainNetParams,
		},
		{
			name:    "segwit mainnet p2wsh v0",
			addr:    "VBM1QSG9QUUNLD2SY752QZ3G8V6NCGHAKZ7GUWSL5HE8PM3CUZXKQQ2AQ3767CL",
			encoded: "vbm1qsg9quunld2sy752qz3g8v6ncghakz7guwsl5he8pm3cuzxkqq2aq3767cl",
			valid:   true,
			result: tstAddressWitnessScriptHash(
				0,
				[32]byte{
					0x82, 0x0a, 0x0e, 0x72, 0x7f, 0x6a, 0xa0, 0x4f,
					0x51, 0x40, 0x14, 0x50, 0x76, 0x6a, 0x78, 0x45,
					0xfb, 0x61, 0x79, 0x1c, 0x74, 0x3f, 0x4b, 0xe4,
					0xe1, 0xdc, 0x71, 0xc1, 0x1a, 0xc0, 0x02, 0xba},
				consensus.MainNetParams.Bech32HRPSegwit),
			f: func() (Address, error) {
				scriptHash := []byte{
					0x82, 0x0a, 0x0e, 0x72, 0x7f, 0x6a, 0xa0, 0x4f,
					0x51, 0x40, 0x14, 0x50, 0x76, 0x6a, 0x78, 0x45,
					0xfb, 0x61, 0x79, 0x1c, 0x74, 0x3f, 0x4b, 0xe4,
					0xe1, 0xdc, 0x71, 0xc1, 0x1a, 0xc0, 0x02, 0xba}
				return NewAddressWitnessScriptHash(scriptHash, &consensus.MainNetParams)
			},
			net: &consensus.MainNetParams,
		},
		{
			name:    "segwit testnet p2wpkh v0",
			addr:    "vtm1qpzap59m2kadjwumj8nv4qevfjqwraytjnxap6u",
			encoded: "vtm1qpzap59m2kadjwumj8nv4qevfjqwraytjnxap6u",
			valid:   true,
			result: tstAddressWitnessPubKeyHash(
				0,
				[20]byte{
					0x08, 0xba, 0x1a, 0x17, 0x6a, 0xb7, 0x5b, 0x27, 0x73, 0x72,
					0x3c, 0xd9, 0x50, 0x65, 0x89, 0x90, 0x1c, 0x3e, 0x91, 0x72},
				consensus.TestNetParams.Bech32HRPSegwit),
			f: func() (Address, error) {
				pkHash := []byte{
					0x08, 0xba, 0x1a, 0x17, 0x6a, 0xb7, 0x5b, 0x27, 0x73, 0x72,
					0x3c, 0xd9, 0x50, 0x65, 0x89, 0x90, 0x1c, 0x3e, 0x91, 0x72}
				return NewAddressWitnessPubKeyHash(pkHash, &consensus.TestNetParams)
			},
			net: &consensus.TestNetParams,
		},
		{
			name:    "segwit testnet p2wsh v0",
			addr:    "vtm1qf5cwj7fydpgdqgcf5lmwp55zdvykwy5qnrtk08w3uefurkwqztcqgkzap0",
			encoded: "vtm1qf5cwj7fydpgdqgcf5lmwp55zdvykwy5qnrtk08w3uefurkwqztcqgkzap0",
			valid:   true,
			result: tstAddressWitnessScriptHash(
				0,
				[32]byte{
					0x4d, 0x30, 0xe9, 0x79, 0x24, 0x68, 0x50, 0xd0,
					0x23, 0x09, 0xa7, 0xf6, 0xe0, 0xd2, 0x82, 0x6b,
					0x09, 0x67, 0x12, 0x80, 0x98, 0xd7, 0x67, 0x9d,
					0xd1, 0xe6, 0x53, 0xc1, 0xd9, 0xc0, 0x12, 0xf0},
				consensus.TestNetParams.Bech32HRPSegwit),
			f: func() (Address, error) {
				scriptHash := []byte{
					0x4d, 0x30, 0xe9, 0x79, 0x24, 0x68, 0x50, 0xd0,
					0x23, 0x09, 0xa7, 0xf6, 0xe0, 0xd2, 0x82, 0x6b,
					0x09, 0x67, 0x12, 0x80, 0x98, 0xd7, 0x67, 0x9d,
					0xd1, 0xe6, 0x53, 0xc1, 0xd9, 0xc0, 0x12, 0xf0}
				return NewAddressWitnessScriptHash(scriptHash, &consensus.TestNetParams)
			},
			net: &consensus.TestNetParams,
		},
		/*
			{
				name:    "segwit testnet p2wsh witness v0",
				addr:    "tm1qqqqqp399et2xygdj5xreqhjjvcmzhxw4aywxecjdzew6hylgvsesvkesyk",
				encoded: "tm1qqqqqp399et2xygdj5xreqhjjvcmzhxw4aywxecjdzew6hylgvsesvkesyk",
				valid:   true,
				result: tstAddressWitnessScriptHash(
					0,
					[32]byte{
						0x00, 0x00, 0x00, 0xc4, 0xa5, 0xca, 0xd4, 0x62,
						0x21, 0xb2, 0xa1, 0x87, 0x90, 0x5e, 0x52, 0x66,
						0x36, 0x2b, 0x99, 0xd5, 0xe9, 0x1c, 0x6c, 0xe2,
						0x4d, 0x16, 0x5d, 0xab, 0x93, 0xe8, 0x64, 0x33},
					consensus.TestNetParams.Bech32HRPSegwit),
				f: func() (Address, error) {
					scriptHash := []byte{
						0x00, 0x00, 0x00, 0xc4, 0xa5, 0xca, 0xd4, 0x62,
						0x21, 0xb2, 0xa1, 0x87, 0x90, 0x5e, 0x52, 0x66,
						0x36, 0x2b, 0x99, 0xd5, 0xe9, 0x1c, 0x6c, 0xe2,
						0x4d, 0x16, 0x5d, 0xab, 0x93, 0xe8, 0x64, 0x33}
					return NewAddressWitnessScriptHash(scriptHash, &consensus.TestNetParams)
				},
				net: &consensus.TestNetParams,
			},
			// Unsupported witness versions (version 0 only supported at this point)
			{
				name:  "segwit mainnet witness v1",
				addr:  "bm1pw508d6qejxtdg4y5r3zarvary0c5xw7kw508d6qejxtdg4y5r3zarvary0c5xw7k7grplx",
				valid: false,
				net:   &consensus.MainNetParams,
			},
			{
				name:  "segwit mainnet witness v16",
				addr:  "BM1SW50QA3JX3S",
				valid: false,
				net:   &consensus.MainNetParams,
			},
			{
				name:  "segwit mainnet witness v2",
				addr:  "bm1zw508d6qejxtdg4y5r3zarvaryvg6kdaj",
				valid: false,
				net:   &consensus.MainNetParams,
			},
			// Invalid segwit addresses
			{
				name:  "segwit invalid hrp",
				addr:  "tc1qw508d6qejxtdg4y5r3zarvary0c5xw7kg3g4ty",
				valid: false,
				net:   &consensus.TestNetParams,
			},
			{
				name:  "segwit invalid checksum",
				addr:  "bm1qw508d6qejxtdg4y5r3zarvary0c5xw7kv8f3t5",
				valid: false,
				net:   &consensus.MainNetParams,
			},
			{
				name:  "segwit invalid witness version",
				addr:  "BM13W508D6QEJXTDG4Y5R3ZARVARY0C5XW7KN40WF2",
				valid: false,
				net:   &consensus.MainNetParams,
			},
			{
				name:  "segwit invalid program length",
				addr:  "bm1rw5uspcuh",
				valid: false,
				net:   &consensus.MainNetParams,
			},
			{
				name:  "segwit invalid program length",
				addr:  "bm10w508d6qejxtdg4y5r3zarvary0c5xw7kw508d6qejxtdg4y5r3zarvary0c5xw7kw5rljs90",
				valid: false,
				net:   &consensus.MainNetParams,
			},
			{
				name:  "segwit invalid program length for witness version 0 (per BIP141)",
				addr:  "BM1QR508D6QEJXTDG4Y5R3ZARVARYV98GJ9P",
				valid: false,
				net:   &consensus.MainNetParams,
			},
			{
				name:  "segwit mixed case",
				addr:  "tm1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3q0sL5k7",
				valid: false,
				net:   &consensus.TestNetParams,
			},
			{
				name:  "segwit zero padding of more than 4 bits",
				addr:  "tm1pw508d6qejxtdg4y5r3zarqfsj6c3",
				valid: false,
				net:   &consensus.TestNetParams,
			},
			{
				name:  "segwit non-zero padding in 8-to-5 conversion",
				addr:  "tm1qrp33g0q5c5txsp9arysrx4k6zdkfs4nce4xj0gdcccefvpysxf3pjxtptv",
				valid: false,
				net:   &consensus.TestNetParams,
			},
		*/
	}

	for _, test := range tests {
		// Decode addr and compare error against valid.
		decoded, err := DecodeAddress(test.addr, test.net)
		if (err == nil) != test.valid {
			t.Errorf("%v: decoding test failed: %v", test.name, err)
			return
		}

		if err == nil {
			// Ensure the stringer returns the same address as the
			// original.

			if decodedStringer, ok := decoded.(fmt.Stringer); ok {
				addr := test.addr

				// For Segwit addresses the string representation
				// will always be lower case, so in that case we
				// convert the original to lower case first.
				if strings.Contains(test.name, "segwit") {
					addr = strings.ToLower(addr)
				}

				if addr != decodedStringer.String() {
					t.Errorf("%v: String on decoded value does not match expected value: %v != %v",
						test.name, test.addr, decodedStringer.String())
					return
				}

			}

			// Encode again and compare against the original.
			encoded := decoded.EncodeAddress()
			if test.encoded != encoded {
				t.Errorf("%v: decoding and encoding produced different addressess: %v != %v",
					test.name, test.encoded, encoded)
				return
			}

			// Perform type-specific calculations.
			var saddr []byte
			switch decoded.(type) {

			case *AddressWitnessPubKeyHash:
				saddr = tstAddressSegwitSAddr(encoded)
			case *AddressWitnessScriptHash:
				saddr = tstAddressSegwitSAddr(encoded)
			}

			// Check script address, as well as the Hash160 method for P2PKH and
			// P2SH addresses.
			if !bytes.Equal(saddr, decoded.ScriptAddress()) {
				t.Errorf("%v: script addresses do not match:\n%x != \n%x",
					test.name, saddr, decoded.ScriptAddress())
				return
			}
			switch a := decoded.(type) {

			case *AddressWitnessPubKeyHash:
				if hrp := a.Hrp(); test.net.Bech32HRPSegwit != hrp {
					t.Errorf("%v: hrps do not match:\n%x != \n%x",
						test.name, test.net.Bech32HRPSegwit, hrp)
					return
				}

				expVer := test.result.(*AddressWitnessPubKeyHash).WitnessVersion()
				if v := a.WitnessVersion(); v != expVer {
					t.Errorf("%v: witness versions do not match:\n%x != \n%x",
						test.name, expVer, v)
					return
				}

				if p := a.WitnessProgram(); !bytes.Equal(saddr, p) {
					t.Errorf("%v: witness programs do not match:\n%x != \n%x",
						test.name, saddr, p)
					return
				}

			case *AddressWitnessScriptHash:
				if hrp := a.Hrp(); test.net.Bech32HRPSegwit != hrp {
					t.Errorf("%v: hrps do not match:\n%x != \n%x",
						test.name, test.net.Bech32HRPSegwit, hrp)
					return
				}

				expVer := test.result.(*AddressWitnessScriptHash).WitnessVersion()
				if v := a.WitnessVersion(); v != expVer {
					t.Errorf("%v: witness versions do not match:\n%x != \n%x",
						test.name, expVer, v)
					return
				}

				if p := a.WitnessProgram(); !bytes.Equal(saddr, p) {
					t.Errorf("%v: witness programs do not match:\n%x != \n%x",
						test.name, saddr, p)
					return
				}
			}

			// Ensure the address is for the expected network.
			if !decoded.IsForNet(test.net) {
				t.Errorf("%v: calculated network does not match expected",
					test.name)
				return
			}
		}

		if !test.valid {
			// If address is invalid, but a creation function exists,
			// verify that it returns a nil addr and non-nil error.
			if test.f != nil {
				_, err := test.f()
				if err == nil {
					t.Errorf("%v: address is invalid but creating new address succeeded",
						test.name)
					return
				}
			}
			continue
		}

		// Valid test, compare address created with f against expected result.
		addr, err := test.f()
		if err != nil {
			t.Errorf("%v: address is valid but creating new address failed with error %v",
				test.name, err)
			return
		}

		if !reflect.DeepEqual(addr, test.result) {
			t.Errorf("%v: created address does not match expected result",
				test.name)
			return
		}
	}
}

// TstAddressWitnessPubKeyHash creates an AddressWitnessPubKeyHash, initiating
// the fields as given.
func tstAddressWitnessPubKeyHash(version byte, program [20]byte,
	hrp string) *AddressWitnessPubKeyHash {

	return &AddressWitnessPubKeyHash{
		hrp:            hrp,
		witnessVersion: version,
		witnessProgram: program,
	}
}

// TstAddressWitnessScriptHash creates an AddressWitnessScriptHash, initiating
// the fields as given.
func tstAddressWitnessScriptHash(version byte, program [32]byte,
	hrp string) *AddressWitnessScriptHash {

	return &AddressWitnessScriptHash{
		hrp:            hrp,
		witnessVersion: version,
		witnessProgram: program,
	}
}

// TstAddressSegwitSAddr returns the expected witness program bytes for
// bech32 encoded P2WPKH and P2WSH bitcoin addresses.
func tstAddressSegwitSAddr(addr string) []byte {
	_, data, err := bech32.Bech32Decode(addr)
	if err != nil {
		return []byte{}
	}

	// First byte is version, rest is base 32 encoded data.
	data, err = bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return []byte{}
	}
	return data
}
