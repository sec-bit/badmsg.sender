// Copyright 2016 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package abi

import (
	"crypto/rand"
	"math/big"
	"minievm/common"
	"minievm/common/math"
	"reflect"

	fuzz "github.com/google/gofuzz"
)

func packFuzzedBytesSlice(fuzzed []byte, fuzzer *fuzz.Fuzzer) []byte {
	l := len(fuzzed) / 32
	len := packNum(reflect.ValueOf(l))
	return append(len, common.RightPadBytes(fuzzed, (l+31)/32*32)...)
}

// fuzzBytesSlice fuzzs the given bytes as [L, V] as the canonical representation
// bytes slice
func fuzzBytesSlice(t Type, fuzzer *fuzz.Fuzzer) []byte {
	switch t.T {
	case StringTy:
		var str string
		fuzzer.Fuzz(&str)
		return packBytesSlice([]byte(str), len(str))
	case BytesTy:
		var byteslice []byte
		fuzzer.Fuzz(&byteslice)
		return packBytesSlice(byteslice, len(byteslice))
	}
	return []byte{}
}

func fuzzAddress(t Type, fuzzer *fuzz.Fuzzer) []byte {
	var address [20]byte
	fuzzer.Fuzz(&address)
	return packElement(t, reflect.ValueOf(address))
}

func fuzzBool(t Type, fuzzer *fuzz.Fuzzer) []byte {
	var val bool
	fuzzer.Fuzz(&val)
	if val {
		return math.PaddedBigBytes(common.Big1, 32)
	}
	return math.PaddedBigBytes(common.Big0, 32)
}

// fuzzElement fuzzs the given reflect value according to the abi specification in
// t.
func fuzzElement(t Type, fuzzer *fuzz.Fuzzer) []byte {
	switch t.T {
	case IntTy, UintTy:
		return fuzzNum(t, fuzzer)
	case StringTy:
		return fuzzBytesSlice(t, fuzzer)
	case AddressTy:
		return common.LeftPadBytes(fuzzAddress(t, fuzzer), 32)
	case BoolTy:
		return fuzzBool(t, fuzzer)
	case BytesTy:
		return fuzzBytesSlice(t, fuzzer)
	case FixedBytesTy, FunctionTy:
		return fuzzBytesSlice(t, fuzzer)
	default:
		panic("abi fuzz: fatal error")
	}
}

func genRandomInSpecialDist() *big.Int {
	maxrange := int64(16)
	chooseRange, err := rand.Int(rand.Reader, big.NewInt(4))
	if err != nil {
		//error handling
	}

	n, _ := rand.Int(rand.Reader, big.NewInt(maxrange))
	/*
		[0, 2^256-1] split into 3 parts
		[0, 2^10-1], [2^255-100, 2^255+99], [2^256-2^10, 2^256-1]
	*/
	// log.Printf("Mutating Storage %s\n", name)

	if chooseRange.Cmp(big.NewInt(0)) == 0 {
		// n = n
	} else if chooseRange.Cmp(big.NewInt(1)) == 0 {
		offset := new(big.Int)
		offset.Exp(big.NewInt(2), big.NewInt(255), nil).Sub(offset, big.NewInt(maxrange))
		n.Add(offset, n)
	} else if chooseRange.Cmp(big.NewInt(2)) == 0 {
		offset := new(big.Int)
		offset.Exp(big.NewInt(2), big.NewInt(255), nil)
		n.Add(offset, n)
	} else if chooseRange.Cmp(big.NewInt(3)) == 0 {
		offset := new(big.Int)
		offset.Exp(big.NewInt(2), big.NewInt(256), nil).Sub(offset, big.NewInt(maxrange))
		n.Add(offset, n)
	}
	return n
}

// fuzzNum fuzzs the given number (using the reflect value) and will cast it to appropriate number representation
func fuzzNum(t Type, fuzzer *fuzz.Fuzzer) []byte {
	switch t.Kind {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		var value uint64
		fuzzer.Fuzz(&value)
		return U256(new(big.Int).SetUint64(value))
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		var value int64
		fuzzer.Fuzz(&value)
		return U256(big.NewInt(value))
	case reflect.Ptr:
		n := genRandomInSpecialDist()
		return U256(n)
	default:
		panic("abi fuzz: fatal error")
	}

}
