package creplayer

import (
	"fmt"
	"log"
	"math/big"
	"minievm/common"
	"minievm/core"
	"minievm/core/state"
	"minievm/core/vm"
	"minievm/params"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
)

type CReplayer struct {
	ContractCreater common.Address
	state           *state.StateDB
	context         *vm.Context
	evm             *vm.EVM
	caddr           common.Address
	rpcclient       *rpc.Client
	txs             []rpcTransaction
}

type rpcTransaction struct {
	BlockHash        string `json:"blockHash"`
	BlockNumber      string `json:"blockNumber"`
	From             string `json:"from"`
	Gas              string `json:"gas"`
	GasPrice         string `json:"gasPrice"`
	Hash             string `json:"hash"`
	Input            string `json:"input"`
	Nonce            string `json:"nonce"`
	To               string `json:"to"`
	TransactionIndex string `json:"transactionIndex"`
	Value            string `json:"value"`
	V                string `json:"v"`
	R                string `json:"r"`
	S                string `json:"s"`
}

type rpcBlock struct {
	Hash         common.Hash      `json:"hash"`
	Transactions []rpcTransaction `json:"transactions"`
	UncleHashes  []common.Hash    `json:"uncles"`
}

func (cr *CReplayer) Init() {
	// cr.rpcclient, _ = rpc.Dial("https://mainnet.infura.io/dAAvz05eGXx5IRc6SX6d")
	cr.rpcclient, _ = rpc.Dial("http://192.168.1.4:8545")
}

func (cr *CReplayer) GetContractBin(addr string) []byte {
	var response string

	err := cr.rpcclient.Call(&response, "eth_getCode", common.HexToAddress(addr), "latest")
	if err != nil {
		log.Print(err)
	}

	return common.Hex2Bytes(response[2:])
}

func (cr *CReplayer) DeployContract() {

	cr.ContractCreater = common.HexToAddress("0x802df0c73eb17e540b39f1ae73c13dcea5a1caaa")

	cr.state = state.New()
	cr.state.AddBalance(cr.ContractCreater, big.NewInt(int64(100)))
	cr.state.SetNonce(cr.ContractCreater, uint64(0))

	cr.context = &vm.Context{
		Transfer:    core.Transfer,
		CanTransfer: core.CanTransfer,
		BlockNumber: big.NewInt(4370001),
		Time:        big.NewInt(time.Now().Unix()),
		GetHash:     func(in uint64) common.Hash { return common.BigToHash(big.NewInt(int64(in))) },
		GasPrice:    big.NewInt(100),
		Difficulty:  big.NewInt(100),
	}

	cr.evm = vm.NewEVM(*cr.context, cr.state, params.MainnetChainConfig, vm.Config{EnableJit: false, ForceJit: false, Debug: false, NoRecursion: false})
	code := common.Hex2Bytes(godgame[2:])
	_, caddr, _, err := cr.evm.Create(vm.AccountRef(cr.ContractCreater), code, uint64(100000000000), big.NewInt(0))
	if err != nil {
		log.Fatal(err)
	}
	cr.caddr = caddr
	log.Printf("0x%02x", caddr)
}

func (cr *CReplayer) Call(calleraddress, calldata, gaslimit, value string) ([]byte, uint64, error) {
	gas := new(big.Int)
	gas.SetString(gaslimit, 0)
	val := new(big.Int)
	val.SetString(value, 0)
	return cr.evm.Call(vm.AccountRef(common.HexToAddress(calleraddress)), cr.caddr, common.Hex2Bytes(calldata[2:]), gas.Uint64(), val)
}

func (cr *CReplayer) TransactionExecuter() {
	// var txs []rpcTransaction

	for _, tx := range cr.txs {
		log.Printf("%s %s %s %s %s\n", tx.Hash, tx.From, tx.Input, tx.Gas, tx.Value)
		cr.Call(tx.From, tx.Input, tx.Gas, tx.Value)
		ret, _, _ := cr.Call(tx.From, "0x4b750334", tx.Gas, "0x00")
		log.Printf("\nsell price: [%02x]", ret)
		ret, _, _ = cr.Call(tx.From, "0x8620410b", tx.Gas, "0x00")
		log.Printf("\n buy price: [%02x]\n\n", ret)

		// cr.state.Print()
	}
}

func (cr *CReplayer) FetchAllTxs(begin, end int) []rpcTransaction {
	path := fmt.Sprintf("cache_%d-%d.gob", begin, end)
	err := Load(path, &cr.txs)
	if err != nil {
		log.Print(err, "Cache file not found. Trying to fetch online")
		for i := begin; i <= end; i++ {
			// var raw json.RawMessage
			var raw rpcBlock
			log.Print(fmt.Sprintf("Fetching block: %d", i))
			err := cr.rpcclient.Call(&raw, "eth_getBlockByNumber", fmt.Sprintf("0x%x", i), true)
			if err != nil {
				log.Print(err)
			}
			// log.Print(raw.Transactions[1])
			for _, tx := range raw.Transactions {
				cr.txs = append(cr.txs, tx)
			}
		}
		Save(path, cr.txs)
	}

	return cr.txs
}

func (cr *CReplayer) AddAllMoney() {
	addresses := make(map[string]bool)
	for _, tx := range cr.txs {
		addresses[tx.From] = true
	}
	for address := range addresses {
		ether := new(big.Int)
		ether = ether.Exp(big.NewInt(10), big.NewInt(18), nil)
		cr.state.AddBalance(common.HexToAddress(address), ether.Mul(ether, big.NewInt(1000)))
	}
}
