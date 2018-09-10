package detectors

import (
	"bytes"
	"log"
	"math/big"
	"minievm/common"
	"testing"
)

func TestFuzzContracts(t *testing.T) {
	path := "~/Documents/zeroklabs/gopath/src/minievm/erc20contracts/t_BEC.sol"
	logpath := "~/Documents/zeroklabs/gopath/src/minievm/fuzz_log"
	cf := NewContractFuzzer(path, logpath, true)
	cf.FuzzContracts()
}

func checkEvent(fi *FuzzInt) bool {
	overflowTopic := common.Hex2Bytes("FEE46111846A282E8199035721DD0334C2BC5C016AE4E72B924003431D6A8759")
	for _, logentry := range fi.contracts.state.Logs {
		logbytes := logentry.Topics[0].Bytes()
		if bytes.Equal(overflowTopic, logbytes) {
			return true
		}
	}
	return false
}
func TestSingleCall(t *testing.T) {
	path := "~/Documents/zeroklabs/gopath/src/minievm/erc20contracts/INT.sol"
	logpath := "~/Documents/zeroklabs/gopath/src/minievm/fuzz_log"
	fi := NewContractFuzzer(path, logpath, false)
	n := new(big.Int)
	n.Exp(big.NewInt(2), big.NewInt(255), nil)
	loc := fi.constantsLoc["sellPrice"]
	log.Printf("%02x, %02x\n", fi.contracts.GetStorage(loc), fi.constantsLoc["sellPrice"])
	fi.contracts.SetStorage(loc, n)
	log.Printf("%02x, %02x\n", fi.contracts.GetStorage(loc), fi.constantsLoc["sellPrice"])
	fi.contracts.BackupStates()
	ret, err := fi.maincontract.Call(fi.contracts.ContractCreater, common.Hex2Bytes("e4849b320000000000000000000000000000000000000000000000000000000000000008"))
	fi.contracts.RestoreStates()
	log.Print(ret, err)
	log.Print(checkEvent(fi))
}
