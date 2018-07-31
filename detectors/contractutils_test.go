package detectors

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"log"
	"minievm/crypto"
	"testing"

	"github.com/google/gofuzz"
)

func TestGetStorageLoc(t *testing.T) {
	su := &ContractUtils{}
	su.DeployContracts("/Users/dynm/Documents/zeroklabs/gopath/src/minievm/erc20contracts/INT.sol")
	su.SetSkippedVars([]string{})
	for name, loc := range su.GetStorageLoc() {
		val := su.evm.StateDB.GetState(su.MainContract.Address, loc)
		t.Logf("%s: %02X, %02X\n", name, loc, val)
	}
}

func TestABIFuzzing(t *testing.T) {
	su := &ContractUtils{}
	su.DeployContracts("/Users/dynm/Documents/zeroklabs/gopath/src/minievm/erc20contracts/INT.sol")
	transferFunc, _ := su.MainContract.ABI.Methods["changename"]
	nameGetter, _ := su.MainContract.ABI.Methods["name"]

	fuzzer := fuzz.New()
	bytes, err := transferFunc.Fuzz(fuzzer)
	if err != nil {
		t.Fatal(err)
	}
	// t.Logf("%02X", bytes)

	ret, err := su.MainContract.Call(su.ContractCreater, nameGetter.Id())
	if err != nil {
		log.Print(err)
	}
	log.Printf("ret0: %s", ret)

	ret, err = su.MainContract.Call(su.ContractCreater, bytes)
	if err != nil {
		log.Print(err)
	}

	ret, err = su.MainContract.Call(su.ContractCreater, nameGetter.Id())
	if err != nil {
		log.Print(err)
	}
	log.Printf("ret1: %s", ret)
}

func TestAddressSliceFuzzing(t *testing.T) {
	su := &ContractUtils{}
	path := "/Users/dynm/Documents/zeroklabs/ContractsDB/etherscan/0xc5d105e63711398af9bbff092d4b6769c82f793d.sol"
	su.DeployContracts(path)
	batchTransferFunc, _ := su.MainContract.ABI.Methods["batchTransfer"]

	fuzzer := fuzz.New()
	bytes, err := batchTransferFunc.Fuzz(fuzzer)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%02X", bytes)

	ret, err := su.MainContract.Call(su.ContractCreater, bytes)
	if err != nil {
		log.Print(err)
	}
	log.Printf("ret0: %s", ret)
}

// func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *keystore.Key {
// 	id := uuid.NewRandom()
// 	key := &Key{
// 		Id:         id,
// 		Address:    crypto.PubkeyToAddress(privateKeyECDSA.PublicKey),
// 		PrivateKey: privateKeyECDSA,
// 	}
// 	return key
// }

// func newKey(rand io.Reader) (*keystore.Key, error) {
// 	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), rand)
// 	if err != nil {
// 		return nil, err
// 	}
// 	return newKeyFromECDSA(privateKeyECDSA), nil
// }

func bruteforceer() {
	for true {
		privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), crand.Reader)
		if err != nil {
			continue
		}
		contractCreater := crypto.PubkeyToAddress(privateKeyECDSA.PublicKey)

		// contractCreater := common.HexToAddress("0x5c92dcc352a677d8f379ec09443bc1f08e04f053")

		caddr := crypto.CreateAddress(contractCreater, 1)
		if (caddr[0] == 0xde) && (caddr[1] == 0xad) && (caddr[2] == 0xbe) && (caddr[3] == 0xef) {
			log.Printf("%02x\n", caddr)
			log.Printf("%s, %s\n", privateKeyECDSA.X.String(), privateKeyECDSA.Y.String())
			log.Fatal()
		}
	}
}
func TestAddrGen(t *testing.T) {
	for i := 0; i < 4; i++ {
		go bruteforceer()
	}
}
