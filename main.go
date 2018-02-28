package main

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"minievm/common"
	"minievm/core"
	st "minievm/core/state"
	"minievm/core/vm"
	"minievm/params"
)

func main() {

	state := st.New()

	addr := common.BytesToAddress([]byte("test1"))
	state.AddBalance(addr, big.NewInt(int64(100)))
	state.SetNonce(addr, uint64(20))

	context := vm.Context{Transfer: core.Transfer, CanTransfer: core.CanTransfer}
	evm := vm.NewEVM(context, state, params.TestChainConfig, vm.Config{EnableJit: false, ForceJit: false})
	code, _ := hex.DecodeString("60606040526001600060005055604c8060186000396000f360606040526000357c010000000000000000000000000000000000000000000000000000000090048063f8a8fd6d146039576035565b6002565b604460048050506046565b005b604a565b56")
	value := big.NewInt(10)
	createAccount(evm, addr, code, value)

	fmt.Printf("statedb: \n")
	for _, s := range state.StateMap {
		fmt.Printf("%s\n", s)
	}
	return
}

func createAccount(evm *vm.EVM, addr common.Address, code []byte, value *big.Int) common.Address {
	ret, newAddr, leftOverGas, err := evm.Create(vm.AccountRef(addr), code, uint64(100000000000), value)
	fmt.Printf("createAccount:\nret: %x\nnewAddr: %x\nleftGas: %d\nerr: %v\n", ret, newAddr, leftOverGas, err)
	return newAddr
}

/*
func main() {
	ret, _, err := runtime.Execute(common.Hex2Bytes("60606040526001600060005055604c8060186000396000f360606040526000357c010000000000000000000000000000000000000000000000000000000090048063f8a8fd6d146039576035565b6002565b604460048050506046565b005b604a565b56"), nil, nil)
	fmt.Printf("ret: %x\nerr: %v\n\n", ret, err)
}
*/

/*
	input, _ := hex.DecodeString("")
	ret, err := vm.Run(evm, contract, input)
	fmt.Printf("ret: %x\nerr: %v\n\n", ret, err)
*/
