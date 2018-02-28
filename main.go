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

func main() {

	state := st.New()

	addr := common.BytesToAddress([]byte("test1"))
	state.AddBalance(addr, big.NewInt(int64(100)))
	state.SetNonce(addr, uint64(20))

	context := vm.Context{Transfer: core.Transfer, CanTransfer: core.CanTransfer, BlockNumber: big.NewInt(4370001)}
	evm := vm.NewEVM(context, state, params.MainnetChainConfig, vm.Config{EnableJit: false, ForceJit: false})
	code := common.Hex2Bytes("606060405260016000553415601357600080fd5b603e8060206000396000f30060606040526001600054016000819055500000a165627a7a72305820bbd996132409b9346189ceb2f0ccade8331848961899526d5287b1c58c9d2b800029")
	fmt.Printf("code: %v\n", code)
	value := big.NewInt(10)

	newAddr := createAccount(evm, addr, code, big.NewInt(0))
	state.Print()

	call(evm, addr, newAddr, nil, value)
	state.Print()
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
