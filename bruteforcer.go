package main

import (
	"crypto/ecdsa"
	crand "crypto/rand"
	"log"
	"minievm/crypto"
	"sync"
)

func bruteforceer(id int) {
	cnt := 0
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
			log.Printf("%s, %s\n", privateKeyECDSA.D.String())
			log.Fatal()
		}
		if cnt%1000 == 0 {
			log.Printf("worker id: %d\n", id)
		}
		cnt++
		cnt = cnt % 1000
	}
}

func main3() {
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go bruteforceer(i)
	}
	wg.Wait()
}
