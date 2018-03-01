package main

import (
	"fmt"
	"math/big"
	"minievm/common"
	"minievm/core"
	st "minievm/core/state"
	"minievm/core/vm"
	"minievm/params"
)

/* contract
pragma solidity ^0.4.0;
contract Ballot {
    int x = 1;

    function test() public payable {
        x = x + 2;
        return;
    }
    function () public payable {
        x = x + 1;
        return;
    }
}
*/

/* function hashes
{
	"f8a8fd6d": "test()"
}
*/

/* contract bytecode
606060405260016000553415601357600080fd5b608f806100216000396000f300606060405260043610603f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063f8a8fd6d14604d575b600160005401600081905550005b60536055565b005b6002600054016000819055505600a165627a7a723058201ed0e5d8b252fc5b253e12827a7d046c87e2cabc8791eedcefa1738d3a438ee90029
*/

func TestExample01() {
	state := st.New()
	addr := common.BytesToAddress([]byte("test1"))
	state.AddBalance(addr, big.NewInt(int64(100)))
	state.SetNonce(addr, uint64(20))
	context := vm.Context{Transfer: core.Transfer, CanTransfer: core.CanTransfer, BlockNumber: big.NewInt(4370001)}
	evm := vm.NewEVM(context, state, params.MainnetChainConfig, vm.Config{EnableJit: false, ForceJit: false})

	code := common.Hex2Bytes("606060405260016000553415601357600080fd5b608f806100216000396000f300606060405260043610603f576000357c0100000000000000000000000000000000000000000000000000000000900463ffffffff168063f8a8fd6d14604d575b600160005401600081905550005b60536055565b005b6002600054016000819055505600a165627a7a723058201ed0e5d8b252fc5b253e12827a7d046c87e2cabc8791eedcefa1738d3a438ee90029")
	fmt.Printf("code: %v\n", code)
	value := big.NewInt(10)

	// test create contract
	newAddr := createAccount(evm, addr, code, big.NewInt(0))
	state.Print()

	// test default function
	call(evm, addr, newAddr, nil, value)
	state.Print()

	// test test function
	input := common.Hex2Bytes("f8a8fd6d")
	call(evm, addr, newAddr, input, value)
	state.Print()
	return
}
