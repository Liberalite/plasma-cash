package main

import (
	"client"
	"context"
	"fmt"
	"log"
	"os"
	"time"
)

func main() {

	plasmaChain := os.Getenv("PLASMA_CHAIN")
	client.InitClients("http://localhost:8545")
	client.InitTokenClient("http://localhost:8545")

	var svc client.ChainServiceClient
	if plasmaChain == "PROTOTYPE_SERVER" {
		//		svc = client.NewChildChainService("http://localhost:8546")
	} else {
		svc = client.NewLoomChildChainService("http://localhost:46658/rpc", "http://localhost:46658/query")
	}

	alice := client.NewClient(svc, client.GetRootChain("alice"), client.GetTokenContract("alice"))

	bob := client.NewClient(svc, client.GetRootChain("bob"), client.GetTokenContract("bob"))
	charlie := client.NewClient(svc, client.GetRootChain("charlie"), client.GetTokenContract("charlie"))
	authority := client.NewClient(svc, client.GetRootChain("authority"),
		client.GetTokenContract("authority"))

	slots := []uint64{}
	alice.DebugCoinMetaData(slots)

	// Give alice 5 tokens
	err := alice.TokenContract.Register()
	if err != nil {
		log.Fatalf("failed registering -%v\n", err)
	}

	aliceTokensStart, err := alice.TokenContract.BalanceOf()
	log.Printf("Alice has %d tokens\n", aliceTokensStart)

	if aliceTokensStart != 5 {
		log.Fatalf("START: Alice has incorrect number of tokens")
	}
	bobTokensStart, err := bob.TokenContract.BalanceOf()
	exitIfError(err)
	log.Printf("Bob has %d tokens\n", bobTokensStart)
	if bobTokensStart != 0 {
		log.Fatalf("START: Bob has incorrect number of tokens")
	}
	charlieTokensStart, err := charlie.TokenContract.BalanceOf()
	exitIfError(err)
	log.Printf("Charlie has %d tokens\n", charlieTokensStart)
	if charlieTokensStart != 0 {
		log.Fatalf("START: Charlie has incorrect number of tokens")
	}

	// Alice deposits 3 of her coins to the plasma contract and gets 3 plasma nft
	// utxos in return
	tokenID := int64(1)
	txHash := alice.Deposit(tokenID)
	fmt.Printf("before deposit event data- %s\n", txHash)
	time.Sleep(1 * time.Second)
	depEvent, err := alice.RootChain.DepositEventData(txHash)
	exitIfError(err)
	depositSlot1 := depEvent.Slot
	fmt.Printf("after deposit event data-%d\n", depositSlot1)
	slots = append(slots, depEvent.Slot)
	alice.DebugCoinMetaData(slots)

	txHash = alice.Deposit(tokenID + 1)
	time.Sleep(1 * time.Second)
	depEvent, err = alice.RootChain.DepositEventData(txHash)
	exitIfError(err)
	depositSlot2 := depEvent.Slot
	slots = append(slots, depEvent.Slot)
	alice.DebugCoinMetaData(slots)

	txHash = alice.Deposit(tokenID + 2)
	time.Sleep(1 * time.Second)
	depEvent, err = alice.RootChain.DepositEventData(txHash)
	depositSlot3 := depEvent.Slot
	exitIfError(err)
	slots = append(slots, depEvent.Slot)
	alice.DebugCoinMetaData(slots)

	//Alice to Bob, and Alice to Charlie. We care about the Alice to Bob
	// transaction
	//	blkNum := int64(3)
	account, err := bob.TokenContract.Account()
	exitIfError(err)
	err = alice.SendTransaction(depositSlot3, int64(depositSlot3), 1, account.Address) //aliceToBob
	exitIfError(err)
	account, err = charlie.TokenContract.Account()
	exitIfError(err)
	err = alice.SendTransaction(depositSlot2, int64(depositSlot2), 1, account.Address) //randomTx
	exitIfError(err)
	authority.SubmitBlock()

	// Bob to Charlie
	blkNum := int64(1000)
	account, err = charlie.TokenContract.Account() // the prev transaction was included in block 1000
	exitIfError(err)
	err = bob.SendTransaction(depositSlot3, blkNum, 1, account.Address) //bobToCharlie
	exitIfError(err)
	authority.SubmitBlock()

	// Charlie should be able to submit an exit by referencing blocks 0 and 1 which
	// included his transaction.
	prevTxBlkNum := int64(1000)
	exitingTxBlkNum := int64(2000)
	charlie.DebugCoinMetaData(slots)
	fmt.Printf("Before start exit\n")
	_, err = charlie.StartExit(depositSlot3, prevTxBlkNum, exitingTxBlkNum)
	exitIfError(err)
	fmt.Printf("After start exit\n")
	charlie.DebugCoinMetaData(slots)

	// After 8 days pass, charlie's exit should be finalizable
	//increaseTime(w3, 8*24*3600)

	fmt.Printf("increase time\n")
	ganache, err := client.ConnectToGanache("http://localhost:8545")
	exitIfError(err)
	fmt.Printf("increase time2\n")
	timeAdj, err := ganache.IncreaseTime(context.TODO(), 8*24*3600)
	fmt.Printf("timeAdj-%v\n", timeAdj)
	exitIfError(err)

	err = authority.FinalizeExits()
	fmt.Printf("finalize exits\n")
	exitIfError(err)

	// Charlie should now be able to withdraw the utxo which included token 2 to his
	// wallet.

	fmt.Printf("withdraw-%d\n", depositSlot3)
	charlie.DebugCoinMetaData(slots)
	err = charlie.Withdraw(depositSlot3)
	exitIfError(err)

	aliceTokensEnd, err := alice.TokenContract.BalanceOf()
	exitIfError(err)
	log.Printf("Alice has %d tokens\n", aliceTokensEnd)
	if aliceTokensEnd != 2 {
		log.Fatal("END: Alice has incorrect number of tokens")
	}

	bobTokensEnd, err := bob.TokenContract.BalanceOf()
	exitIfError(err)
	log.Printf("Bob has %d tokens\n", bobTokensEnd)
	if bobTokensEnd != 0 {
		log.Fatal("END: Bob has incorrect number of tokens")
	}
	charlieTokensEnd, err := charlie.TokenContract.BalanceOf()
	exitIfError(err)
	log.Printf("Charlie has %d  tokens\n", charlieTokensEnd)
	if charlieTokensEnd != 1 {
		log.Fatal("END: Charlie has incorrect number of tokens")
	}

	log.Printf("Plasma Cash with ERC721 tokens success :)")

}

// not idiomatic go, but it cleans up this sample
func exitIfError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
