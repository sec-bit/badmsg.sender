package creplayer

import (
	"log"
	"testing"
)

func TestGetContractBin(t *testing.T) {
	// t.Fatal(GetContractBin("0xca6378fcdf24ef34b4062dda9f1862ea59bafd4d"))
}

func TestDeployContract(t *testing.T) {
	cr := &CReplayer{}
	cr.DeployContract()
}

func TestFetchAllTxs(t *testing.T) {
	cr := &CReplayer{}
	cr.Init()
	txs := cr.FetchAllTxs(6176235, 6182468, "0xca6378fcdf24ef34b4062dda9f1862ea59bafd4d")
	// txs := cr.FetchAllTxs(6182468, 6182468, "0xca6378fcdf24ef34b4062dda9f1862ea59bafd4d")
	for _, tx := range txs {
		log.Print(tx.From, tx.To)
	}
}
