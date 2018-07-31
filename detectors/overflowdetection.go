package detectors

import (
	"bytes"
	"log"
	"math/big"
	"minievm/accounts/abi"
	"minievm/common"
	"minievm/core"
	"minievm/core/state"
	"minievm/core/vm"
	"minievm/params"
	"strconv"
	"strings"
	"time"
)

/*
	We taint the storage by the overflow event then check if EVM reverted.
	If EVM doesn't revert and the sotrage is tainted by us, we may found an integer overflow bug.
*/
type OverFlowDetector struct {
	Owner, Attacker, ContractAddress, OtherOne common.Address
	contractName                               string
	victims                                    [10]common.Address
	state                                      *state.StateDB
	context                                    *vm.Context
	evm                                        *vm.EVM
	ABI                                        abi.ABI
	Executionresult                            error
	Detected                                   bool
	Reason                                     string
	Attackvector                               []byte
	Unpackedinput                              []interface{}
}

//InitExternalAccount init new accounts
func (ofd *OverFlowDetector) InitExternalAccount() {
	ofd.Attacker = common.BytesToAddress([]byte("Attacker"))
	ofd.Owner = common.BytesToAddress([]byte("Owner"))
	ofd.state.AddBalance(ofd.Attacker, big.NewInt(int64(100)))
	ofd.state.AddBalance(ofd.Owner, big.NewInt(int64(100)))
	ofd.state.SetNonce(ofd.Attacker, uint64(20))
	ofd.state.SetNonce(ofd.Owner, uint64(20))
	for i := 0; i < 10; i++ {
		addr := common.BytesToAddress([]byte("testaddress" + strconv.Itoa(i)))
		ofd.victims[i] = addr
		ofd.state.AddBalance(addr, big.NewInt(int64(100)))
	}
}

func (ofd *OverFlowDetector) Init(contractName, abijson string) {
	contractNameSlice := strings.Split(contractName, "/")
	ofd.contractName = contractNameSlice[len(contractNameSlice)-1]

	log.SetPrefix(ofd.contractName + " ")
	ofd.state = state.New()
	ofd.InitExternalAccount()
	abi, err := abi.JSON(strings.NewReader(abijson))
	if err != nil {
		log.Print("ABI decode error", err)
		return
	}
	ofd.ABI = abi

	ofd.context = &vm.Context{
		Transfer:    core.Transfer,
		CanTransfer: core.CanTransfer,
		BlockNumber: big.NewInt(4370001),
		Time:        big.NewInt(time.Now().Unix()),
		GetHash:     func(in uint64) common.Hash { return common.BigToHash(big.NewInt(int64(in))) },
		GasPrice:    big.NewInt(100),
		Difficulty:  big.NewInt(100),
	}
	ofd.evm = vm.NewEVM(*ofd.context, ofd.state, params.MainnetChainConfig, vm.Config{EnableJit: false, ForceJit: false, Debug: false})
	ofd.evm.Callback = NULLCallback
}

// CreateContract creates a new contract
func (ofd *OverFlowDetector) CreateContract(addr common.Address, code []byte, value *big.Int) (err error) {
	_, caddr, _, err := ofd.evm.Create(vm.AccountRef(addr), code, uint64(100000000000), value)
	if err != nil {
		return
	}
	ofd.ContractAddress = caddr
	return
}

func (ofd *OverFlowDetector) SetCallback(cb vm.CallbackFunc) {
	ofd.evm.Callback = cb
}

func (ofd *OverFlowDetector) GetState(key common.Hash) common.Hash {
	return ofd.state.GetState(ofd.ContractAddress, key)
}

func (ofd *OverFlowDetector) SetState(key, hash common.Hash) {
	ofd.state.SetState(ofd.ContractAddress, key, hash)
}

// FunctionCall call from addr to contract
func (ofd *OverFlowDetector) FunctionCall(from common.Address, to common.Address, input []byte, value *big.Int) (ret []byte, err error) {
	ret, _, err = ofd.evm.Call(vm.AccountRef(from), to, input, uint64(100000000000), value)
	ofd.Executionresult = err
	return
}

func (ofd *OverFlowDetector) CallFunctionStub(stubname string) (ret []byte, err error) {
	packed, err := ofd.ABI.Pack(stubname)
	if err != nil {
		return nil, err
	}
	ret, err = ofd.FunctionCall(ofd.Owner, ofd.ContractAddress, packed, big.NewInt(0))
	return
}

// ERC20GetAccountInfo gets account info such as totalSupply, balanceOf...
func (ofd *OverFlowDetector) ERC20GetAccountInfo(instruction string, args ...interface{}) (*big.Int, error) {
	packed, err := ofd.ABI.Pack(instruction)
	if err != nil {
		return nil, err
	}
	ret, err := ofd.FunctionCall(ofd.Owner, ofd.ContractAddress, packed, big.NewInt(0))
	res := new(big.Int)
	res.SetBytes(ret)
	return res, err
}

// ERC20Transfer the caller do a transfer to receiver
func (ofd *OverFlowDetector) ERC20Transfer(caller, receiver common.Address, amount *big.Int) {
	packed, err := ofd.ABI.Pack("transfer", receiver, amount)
	if err != nil {
		return
		log.Println("Pack error:", err)
	}
	_, err = ofd.FunctionCall(caller, ofd.ContractAddress, packed, big.NewInt(0))
	if err != nil {
		return
		// log.Println("FunctionCall error:", err)
	}
}

// InitAttackerToken gives some token away
func (ofd *OverFlowDetector) InitAttackerToken() {
	totalSupply, err := ofd.ERC20GetAccountInfo("totalSupply")
	if err != nil {
		// log.Println("not support totalSupply")
		return
	}
	ofd.ERC20Transfer(ofd.Owner, ofd.Attacker, new(big.Int).Div(totalSupply, big.NewInt(10000)))
}

// InitVictimsToken gives tokens to victims
func (ofd *OverFlowDetector) InitVictimsToken() {
	totalSupply, err := ofd.ERC20GetAccountInfo("totalSupply")
	if err != nil {
		// log.Println("not support totalSupply")
		return
	}
	for i := 0; i < 10; i++ {
		ofd.ERC20Transfer(ofd.Owner, ofd.victims[i], new(big.Int).Div(totalSupply, big.NewInt(10000)))
	}
}

//OverFlowDetector detectes the integer overflow
func (ofd *OverFlowDetector) OverFlowDetector(codeHex, function string, initAttacker, initVictims bool) {
	ofd.Detected = false
	ofd.Reason = ""
	ofd.Attackvector = nil
	ofd.Unpackedinput = nil

	// ops := []string{add, sub, mul, div}
	code := common.Hex2Bytes(codeHex)
	err := ofd.CreateContract(ofd.Owner, code, big.NewInt(0))
	if err != nil {
		ofd.Detected = false
		return
	}
	if err != nil {
		log.Println("ABI Decode Err:", err)
		return
	}

	if initAttacker {
		ofd.InitAttackerToken()
	}
	if initVictims {
		ofd.InitVictimsToken()
	}

	c, _, _ := GenerateInputsByABI(ofd.ABI, function)
	if c == nil {
		return
	}
	// bar := pb.StartNew(total)

	// log.Println("Types:", types, "Total:", total)
	// var args []interface{}
	for product := range c {
		// fmt.Println(product)
		// bar.Increment()
		packed, err := ofd.ABI.Pack(function, product...)
		if err != nil {
			log.Println("Pack error:", err)
			continue
		}
		_, err = ofd.FunctionCall(ofd.Attacker, ofd.ContractAddress, packed, big.NewInt(0))
		// log.Print("ret", ret, (new(big.Int).SetBytes(ret).Cmp(big.NewInt(1)) == 0))
		if err == nil {
			// for _, op := range ops {
			// 	res := ofd.state.GetState(ofd.ContractAddress, common.StringToHash(op))
			// 	emptyHash := common.Hash{}
			// 	if res != emptyHash && (new(big.Int).SetBytes(ret).Cmp(big.NewInt(1)) == 0) {
			// 		// log.Printf("Attack Vector: %02x\nUnpacked Input: %02x", packed, product)
			// 		// log.Println(op[10:], "@PC =", res.Big())
			// 		// log.Println()
			// 		ofd.Detected = true
			// 		ofd.Reason = op[10:] + " @PC = " + res.Big().String()
			// 		ofd.Attackvector = packed
			// 		ofd.Unpackedinput = product
			// 		// return
			// 	}
			// }
			// ret, err = ofd.CallFunctionStub("stub_overflowdetection")
			foundOverflow := false
			overflowTopic := common.Hex2Bytes("FEE46111846A282E8199035721DD0334C2BC5C016AE4E72B924003431D6A8759")
			for _, logentry := range ofd.state.Logs {
				logbytes := logentry.Topics[0].Bytes()
				if bytes.Equal(overflowTopic, logbytes) {
					foundOverflow = true
				}

				// if reflect.DeepEqual(logentry.Topics[0], common.StringToHash("FEE46111846A282E8199035721DD0334C2BC5C016AE4E72B924003431D6A8759")) {
				// 	foundOverflow = true
				// }
			}
			// if err != nil {
			// 	return
			// }
			// stubRet := new(big.Int).SetBytes(ret)
			// if stubRet.Cmp(big.NewInt(1)) == 0 {
			if foundOverflow {
				ofd.Attackvector = packed
				ofd.Detected = true
				ofd.Unpackedinput = product
			}

		}
	}
}
