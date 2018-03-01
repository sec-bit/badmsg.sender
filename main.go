package main

import (
	"fmt"
	"math/big"
	"minievm/common"
	"minievm/core/vm"
)

func main() {

	//TestExample01()
	TestERC20()
	return
}

func createAccount(evm *vm.EVM, addr common.Address, code []byte, value *big.Int) common.Address {
	ret, newAddr, leftOverGas, err := evm.Create(vm.AccountRef(addr), code, uint64(100000000000), value)
	fmt.Printf("createAccount:\nret: %x\nnewAddr: %x\nleftGas: %d\nerr: %v\n", ret, newAddr, leftOverGas, err)
	return newAddr
}

func call(evm *vm.EVM, addr common.Address, to common.Address, input []byte, value *big.Int) {
	ret, leftOverGas, err := evm.Call(vm.AccountRef(addr), to, input, uint64(100000000000), value)
	fmt.Printf("call:\nret: %x\nleftGas: %d\nerr: %v\n", ret, leftOverGas, err)
	return
}
