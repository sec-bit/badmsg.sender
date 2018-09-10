package main

import (
	"minievm/creplayer"
)

func main4() {
	cr := &creplayer.CReplayer{}
	cr.Init()
	cr.DeployContract()
	cr.FetchAllTxs(6176235, 6182468)
	// txs := cr.FetchAllTxs(6182468, 6182468, "0xca6378fcdf24ef34b4062dda9f1862ea59bafd4d")
	// for _, tx := range txs {
	// 	// log.Print(tx.From, tx.To)
	// }
	cr.AddAllMoney()
	cr.TransactionExecuter()
}
