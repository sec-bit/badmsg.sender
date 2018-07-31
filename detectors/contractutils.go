package detectors

import (
	"log"
	"math/big"
	"minievm/accounts/abi"
	"minievm/common"
	"minievm/core"
	"minievm/core/state"
	"minievm/core/vm"
	"minievm/params"
	"strings"
	"time"

	"github.com/jinzhu/copier"
)

const (
	contractcreator  = "One key to rule them all"
	contractattacker = "Contract Attacker...maybe"
	emptyaddress     = "Empty Address"
)

type ContractUtils struct {
	ContractCreater, ContractAttacker common.Address
	state                             *state.StateDB
	context                           *vm.Context
	evm                               *vm.EVM
	ABI                               abi.ABI
	Contracts                         map[string]SimpleContract
	MainContract                      SimpleContract
	SkippedVars                       []string
	stateBackup                       *state.StateDB
}

type SimpleContract struct {
	Name    string
	Address common.Address
	BIN     string
	ABI     abi.ABI
	evm     *vm.EVM
}

//Call this contract's function
func (contract SimpleContract) Call(calleraddress common.Address, calldata []byte) (ret []byte, err error) {
	ret, _, err = contract.evm.Call(vm.AccountRef(calleraddress), contract.Address, calldata, uint64(100000000000), big.NewInt(0))
	return
}

//EVMCallback injects into evm instruction
func EVMCallback(args []interface{}) {
	callername := GetCallerName(2)
	log.Print(callername)
	loc := args[0].(common.Hash)
	val := args[1].(*big.Int)
	log.Printf("%02X, %02X", loc, val)
}

//NULLCallback do nothing
func NULLCallback(args []interface{}) {

}

//NewContract create a new contract and deploy to storage
func NewContract(path string) *ContractUtils {
	su := &ContractUtils{}
	su.DeployContracts(path)
	su.SetSkippedVars([]string{})
	return su
}

//DeployContracts deploys all contracts from source
func (cu *ContractUtils) DeployContracts(path string) {
	solcout, err := CompileContract(path)
	if err != nil {
		log.Print("Compile Contract err...", err, path)
	}

	cu.ContractCreater = common.StringToAddress(contractcreator)
	cu.ContractAttacker = common.StringToAddress(contractattacker)

	cu.state = state.New()
	cu.state.AddBalance(cu.ContractCreater, big.NewInt(int64(100)))
	cu.state.AddBalance(cu.ContractAttacker, big.NewInt(int64(100)))
	cu.state.SetNonce(cu.ContractCreater, uint64(20))
	cu.state.SetNonce(cu.ContractAttacker, uint64(20))

	cu.context = &vm.Context{
		Transfer:    core.Transfer,
		CanTransfer: core.CanTransfer,
		BlockNumber: big.NewInt(4370001),
		Time:        big.NewInt(time.Now().Unix()),
		GetHash:     func(in uint64) common.Hash { return common.BigToHash(big.NewInt(int64(in))) },
		GasPrice:    big.NewInt(100),
		Difficulty:  big.NewInt(100),
	}

	cu.evm = vm.NewEVM(*cu.context, cu.state, params.MainnetChainConfig, vm.Config{EnableJit: false, ForceJit: false, Debug: false})

	methodsandeventscount := 0
	cu.Contracts = make(map[string]SimpleContract)
	for name, contract := range solcout.Contracts {
		// log.Print("name:", name)
		code := common.Hex2Bytes(contract.Bin)
		_, caddr, _, err := cu.evm.Create(vm.AccountRef(cu.ContractCreater), code, uint64(100000000000), big.NewInt(0))
		cu.state.AddBalance(caddr, big.NewInt(int64(100)))
		if err != nil {
			log.Print("Create contract err...", err)
			continue
			// caddr = common.StringToAddress(emptyaddress)
		}
		abidecode, err := abi.JSON(strings.NewReader(contract.Abi))
		if err != nil {
			log.Print("ABI decode err...", err)
			continue
		}
		cu.Contracts[name] = SimpleContract{name, caddr, contract.Bin, abidecode, cu.evm}

		if len(abidecode.Methods)+len(abidecode.Events) > methodsandeventscount {
			cu.MainContract = cu.Contracts[name]
			methodsandeventscount = len(abidecode.Methods) + len(abidecode.Events)
		}
	}
	// log.Print("Main Contract:", cu.MainContract.Name)
}

func (cu *ContractUtils) BackupStates() {
	cu.stateBackup = state.New()
	copier.Copy(cu.stateBackup, cu.state)
	// cu.stateBackup.Print()
}

func (cu *ContractUtils) RestoreStates() {
	copier.Copy(cu.state, cu.stateBackup)
	// cu.state.Print()
}

//SetSkippedVars skips vars we don't care
func (cu *ContractUtils) SetSkippedVars(names []string) {
	if len(names) > 0 {
		cu.SkippedVars = names
	} else {
		cu.SkippedVars = []string{"name", "symbol", "totalSupply", "decimals"}
	}
}

//FilterOutConstantNumABI returns all constant ABI and has 0 inputs
func (cu *ContractUtils) FilterOutConstantNumABI() map[string]abi.Method {
	constantMethods := make(map[string]abi.Method)
	for methodName, method := range cu.MainContract.ABI.Methods {
		if method.Const == true && len(method.Inputs) == 0 {
			if len(method.Outputs) == 1 {
				if method.Outputs[0].Type.T == UintTy || method.Outputs[0].Type.T == IntTy {
					constantMethods[methodName] = method
				}
			}
		}
	}

	return constantMethods
}

func (cu *ContractUtils) SetStorage(loc common.Hash, value *big.Int) {
	cu.state.SetState(cu.MainContract.Address, loc, common.BytesToHash(abi.U256(value)))
}

func (cu *ContractUtils) GetStorage(loc common.Hash) common.Hash {
	return cu.state.GetState(cu.MainContract.Address, loc)
}

//Call evm.Call proxy
func (cu *ContractUtils) Call(calleraddress common.Address, contractaddr common.Address, calldata []byte, gaslimit uint64, value *big.Int) (ret []byte, leftOverGas uint64, err error) {
	ret, leftOverGas, err = cu.evm.Call(vm.AccountRef(calleraddress), contractaddr, calldata, gaslimit, value)
	return
}

//SimpleCall presets gaslimit and value
func (cu *ContractUtils) SimpleCall(calleraddress common.Address, contractaddr common.Address, calldata []byte) (ret []byte, err error) {
	ret, _, err = cu.evm.Call(vm.AccountRef(calleraddress), contractaddr, calldata, uint64(100000000000), big.NewInt(0))
	return
}

//TouchSload returns all constant loc in stateDB
func (cu *ContractUtils) TouchSload(contractaddr common.Address, abiin abi.Method) (common.Hash, error) {
	touched := false
	prevCb := cu.evm.Callback
	var storageLoc common.Hash
	cu.evm.Callback = func(args []interface{}) {
		if !touched {
			callername := GetCallerName(2)
			if callername == "minievm/core/vm.opSload" {
				storageLoc = args[0].(common.Hash)
				touched = true
			}
		}
	}
	_, _, err := cu.evm.Call(vm.AccountRef(cu.ContractCreater), contractaddr, abiin.Id(), uint64(100000000000), big.NewInt(0))
	if err != nil {
		log.Print("EVM Call Error...", err)
	}
	cu.evm.Callback = prevCb
	return storageLoc, err
}

//GetStorageLoc returns {name:loc} pair
func (cu *ContractUtils) GetStorageLoc() map[string]common.Hash {
	storageLoc := make(map[string]common.Hash)
	constantABI := cu.FilterOutConstantNumABI()
	for c, abiin := range constantABI {
		if StringInSlice(c, cu.SkippedVars) {
			continue
		}
		loc, _ := cu.TouchSload(cu.MainContract.Address, abiin)
		storageLoc[c] = loc
	}
	return storageLoc
}
