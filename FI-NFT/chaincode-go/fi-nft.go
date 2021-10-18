package main

import "github.com/hyperledger/fabric-contract-api-go/contractapi"
import "fi-nft/chaincode"

func main() {
	smartContract := new(chaincode.SmartContract)

	cc, err := contractapi.NewChaincode(smartContract)

	if err != nil {
		panic(err.Error())
	}

	if err := cc.Start(); err != nil {
		panic(err.Error())
	}
}

