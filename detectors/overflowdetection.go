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
	"strconv"
	"strings"
	"time"
)

/*
	We taint the storage by the overflow event then check if EVM reverted.
	If EVM doesn't revert and the sotrage is tainted by us, we may found an integer overflow bug.
*/
type OverFlowDetector struct {
	owner, attacker, contractaddress, otherone common.Address
	victims                                    [10]common.Address
	state                                      *state.StateDB
	context                                    *vm.Context
	evm                                        *vm.EVM
	abi                                        abi.ABI
	Executionresult                            error
	Detected                                   bool
	Reason                                     string
	Attackvector                               []byte
	Unpackedinput                              []interface{}
}

//InitExternalAccount init new accounts
func (ofd *OverFlowDetector) InitExternalAccount() {
	ofd.attacker = common.BytesToAddress([]byte("Attacker"))
	ofd.owner = common.BytesToAddress([]byte("Owner"))
	ofd.state.AddBalance(ofd.attacker, big.NewInt(int64(100)))
	ofd.state.AddBalance(ofd.owner, big.NewInt(int64(100)))
	ofd.state.SetNonce(ofd.attacker, uint64(20))
	ofd.state.SetNonce(ofd.owner, uint64(20))
	for i := 0; i < 10; i++ {
		addr := common.BytesToAddress([]byte("testaddress" + strconv.Itoa(i)))
		ofd.victims[i] = addr
		ofd.state.AddBalance(addr, big.NewInt(int64(100)))
	}
}

func (ofd *OverFlowDetector) Init(abijson string) {
	ofd.state = state.New()
	ofd.InitExternalAccount()
	abi, err := abi.JSON(strings.NewReader(abijson))
	if err != nil {
		log.Panic("ABI decode error", err)
	}
	ofd.abi = abi

	ofd.context = &vm.Context{
		Transfer:    core.Transfer,
		CanTransfer: core.CanTransfer,
		BlockNumber: big.NewInt(4370001),
		Time:        big.NewInt(time.Now().Unix()),
		GetHash:     func(in uint64) common.Hash { return common.BigToHash(big.NewInt(int64(in))) },
		GasPrice:    big.NewInt(100),
		Difficulty:  big.NewInt(100),
	}
	ofd.evm = vm.NewEVM(*ofd.context, ofd.state, params.MainnetChainConfig, vm.Config{EnableJit: false, ForceJit: false})
}

// CreateContract creates a new contract
func (ofd *OverFlowDetector) CreateContract(addr common.Address, code []byte, value *big.Int) (err error) {
	_, caddr, _, err := ofd.evm.Create(vm.AccountRef(addr), code, uint64(100000000000), value)
	if err != nil {
		return
	}
	ofd.contractaddress = caddr
	return
}

// FunctionCall call from addr to contract
func (ofd *OverFlowDetector) FunctionCall(from common.Address, to common.Address, input []byte, value *big.Int) (ret []byte, err error) {
	ret, _, err = ofd.evm.Call(vm.AccountRef(from), to, input, uint64(100000000000), value)
	ofd.Executionresult = err
	return
}

// ERC20GetAccountInfo gets account info such as totalSupply, balanceOf...
func (ofd *OverFlowDetector) ERC20GetAccountInfo(instruction string, args ...interface{}) *big.Int {
	packed, err := ofd.abi.Pack(instruction)
	if err != nil {
		log.Panic("Pack error:", err)
	}
	ret, err := ofd.FunctionCall(ofd.owner, ofd.contractaddress, packed, big.NewInt(0))
	res := new(big.Int)
	res.SetBytes(ret)
	return res
}

// ERC20Transfer the caller do a transfer to receiver
func (ofd *OverFlowDetector) ERC20Transfer(caller, receiver common.Address, amount *big.Int) {
	packed, err := ofd.abi.Pack("transfer", receiver, amount)
	if err != nil {
		log.Panic("Pack error:", err)
	}
	_, err = ofd.FunctionCall(caller, ofd.contractaddress, packed, big.NewInt(0))
	if err != nil {
		log.Panic("FunctionCall error:", err)
	}
}

// InitAttackerToken gives some token away
func (ofd *OverFlowDetector) InitAttackerToken() {
	totalSupply := ofd.ERC20GetAccountInfo("totalSupply")
	ofd.ERC20Transfer(ofd.owner, ofd.attacker, new(big.Int).Div(totalSupply, big.NewInt(10000)))
}

// InitVictimsToken gives tokens to victims
func (ofd *OverFlowDetector) InitVictimsToken() {
	totalSupply := ofd.ERC20GetAccountInfo("totalSupply")

	// log.Println("Transfer 0.01% of totalSupply from owner to Victim")
	for i := 0; i < 10; i++ {
		ofd.ERC20Transfer(ofd.owner, ofd.victims[i], new(big.Int).Div(totalSupply, big.NewInt(10000)))
	}
}

//OverFlowDetector detectes the integer overflow
func (ofd *OverFlowDetector) OverFlowDetector(codeHex string, function string) {
	ofd.Detected = false
	ofd.Reason = ""
	ofd.Attackvector = nil
	ofd.Unpackedinput = nil

	ops := []string{add, sub, mul, div}
	code := common.Hex2Bytes(codeHex)
	err := ofd.CreateContract(ofd.owner, code, big.NewInt(0))
	if err != nil {
		ofd.Detected = false
		return
	}
	if err != nil {
		log.Println("ABI Decode Err:", err)
		return
	}

	ofd.InitAttackerToken()
	ofd.InitVictimsToken()

	c, _, _ := GenerateInputsByABI(ofd.abi, function)
	// bar := pb.StartNew(total)

	// log.Println("Types:", types, "Total:", total)
	// var args []interface{}
	for product := range c {
		// fmt.Println(product)
		// bar.Increment()
		packed, err := ofd.abi.Pack(function, product...)
		if err != nil {
			log.Println("Pack error:", err)
			continue
		}
		ret, err := ofd.FunctionCall(ofd.attacker, ofd.contractaddress, packed, big.NewInt(0))
		// log.Print("ret", ret, (new(big.Int).SetBytes(ret).Cmp(big.NewInt(1)) == 0))
		if err == nil {
			for _, op := range ops {
				res := ofd.state.GetState(ofd.contractaddress, common.StringToHash(op))
				emptyHash := common.Hash{}
				if res != emptyHash && (new(big.Int).SetBytes(ret).Cmp(big.NewInt(1)) == 0) {
					// log.Printf("Attack Vector: %02x\nUnpacked Input: %02x", packed, product)
					// log.Println(op[10:], "@PC =", res.Big())
					// log.Println()
					ofd.Detected = true
					ofd.Reason = op[10:] + " @PC = " + res.Big().String()
					ofd.Attackvector = packed
					ofd.Unpackedinput = product
					// return
				}
			}
		}
	}
}
