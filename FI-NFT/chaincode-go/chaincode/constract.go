package chaincode

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	shell "github.com/ipfs/go-ipfs-api"
	"strings"
)

const AdmintMSPID = "Org1MSP"
const NFTPrefix="tokenID~CID~Oaccount"
const BidPrefix="tokenID~maxBids"
const BalancePrefix="account~balance"
const NFTListsPrefix="tokenID~tokenID~~"

// SmartContract provides functions for transferring tokens between accounts
type SmartContract struct {
	contractapi.Contract
}


type NFT struct{
	ID string
	CID string
	Owner string
}
type NFTBid struct{
	TokenID string
	CurrentPrice uint64
	KillPrice uint64
}

type AccountBalance struct{
	Account string
	Balance uint64
}



func (s *SmartContract) ClientAccountID(ctx contractapi.TransactionContextInterface) (string, error) {

	// Get ID of submitting client identity
	clientAccountID, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "", fmt.Errorf("failed to get client id: %v", err)
	}

	return clientAccountID, nil
}

func (s *SmartContract) GetAllNFT(ctx contractapi.TransactionContextInterface)([]string,error){
	key,err:=ctx.GetStub().CreateCompositeKey(BidPrefix,[]string{""})
	if err!=nil{
		return nil,fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(key)
	if err!=nil{
		return nil,fmt.Errorf("failed to getstate for key: %s, %v",key,err)
	}
	var value []string
	err = json.Unmarshal(jvalue, value)
	if err != nil {
		return nil,fmt.Errorf("failed to unmarshal data %v",err)
	}
	fmt.Printf("get all nft %v\n",value)
	return value,nil
}

func addNewNFT(ctx contractapi.TransactionContextInterface, newTokenID string)error{
	key,err:=ctx.GetStub().CreateCompositeKey(BidPrefix,[]string{""})
	if err!=nil{
		return fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(key)
	if err!=nil{
		return fmt.Errorf("failed to getstate for key: %s, %v",key,err)
	}
	var value []string
	err = json.Unmarshal(jvalue, value)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data %v",err)
	}

	value=append(value,newTokenID)
	fmt.Printf("add new nft %v\n",value)
	jvalue, err =json.Marshal(value)
	if err!=nil{
		return fmt.Errorf("failed to marshal data %v",err)
	}
	return ctx.GetStub().PutState(key,jvalue)
}



func (s *SmartContract) GetAccount(ctx contractapi.TransactionContextInterface, account string) (*AccountBalance,error){
	key,err:=ctx.GetStub().CreateCompositeKey(BalancePrefix,[]string{account})
	if err!=nil{
		return nil,fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(key)
	fmt.Printf("Get Account (%v,%v)\n",key,jvalue)
	if err!=nil{
		return nil,fmt.Errorf("failed to getstate for key: %s, %v",key,err)
	}
	if len(jvalue)==0{
		return nil, fmt.Errorf("Account not exist\n")
	}
	value:=&AccountBalance{}
	err = json.Unmarshal(jvalue, value)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data %v",err)
	}
	return value,nil
}

func (s *SmartContract) UpdateAccount(ctx contractapi.TransactionContextInterface, account string, balance uint64) (*AccountBalance,error){
	//only authored operator can update account balance
	err := authorization(ctx)
	if err != nil {
		return nil,err
	}

	value:=&AccountBalance{
		Account: account,
		Balance: balance,
	}
	jvalue, err :=json.Marshal(value)
	if err!=nil{
		return nil,fmt.Errorf("failed to marshal data %v",err)
	}

	key,err:=ctx.GetStub().CreateCompositeKey(BalancePrefix,[]string{account})
	if err!=nil{
		return nil,fmt.Errorf("failed to create composite key %v\n",err)
	}
	err = ctx.GetStub().PutState(key, jvalue)
	fmt.Printf("Update Account (%v,%v)\n",key,value)
	if err!=nil{
		return nil,fmt.Errorf("failed to PutState %v\n",err)
	}
	return value,nil
}


func (s *SmartContract) GetBid(ctx contractapi.TransactionContextInterface, tokenID string) (*NFTBid,error){
	key,err:=ctx.GetStub().CreateCompositeKey(BidPrefix,[]string{tokenID})
	if err!=nil{
		return nil,fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(key)
	if err!=nil{
		return nil,fmt.Errorf("failed to getstate for key: %s, %v",key,err)
	}
	value:=&NFTBid{}
	err = json.Unmarshal(jvalue, value)
	fmt.Printf("Get Bid %v\n",value)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data %v",err)
	}
	return value,nil
}

func (s *SmartContract) UpdateBid(ctx contractapi.TransactionContextInterface, account string, balance uint64) error{
	//only authored operator can update account balance
	err := authorization(ctx)
	if err != nil {
		return err
	}

	value:=&AccountBalance{
		Account: account,
		Balance: balance,
	}
	jvalue, err :=json.Marshal(value)
	if err!=nil{
		return fmt.Errorf("failed to marshal data %v",err)
	}

	key,err:=ctx.GetStub().CreateCompositeKey(BalancePrefix,[]string{account})
	if err!=nil{
		return fmt.Errorf("failed to create composite key %v\n",err)
	}
	err = ctx.GetStub().PutState(key, jvalue)

	return err
}


func (s *SmartContract) MintWithFile (ctx contractapi.TransactionContextInterface, id uint64, file string) (string,error){
	/*
		file,ferr:=os.OpenFile(filepath,os.O_CREATE,0666)
		defer file.Close()
		_, ferr = file.Write([]byte("Hello NFT!"))
		if ferr != nil {
			return fmt.Errorf("failed to write file")
		}*/


	/*file,ferr:=os.Open(filepath)
	if ferr!=nil{
		fmt.Printf("ERR:%s\n",ferr.Error())
		return ferr
	}
	defer file.Close()

	r:=bufio.NewReader(file)*/

	sh := shell.NewShell("ipfs_host:5001")
	//cid, erripfs := sh.Add(r)
	cid,erripfs:=sh.Add(strings.NewReader(file))
	if erripfs != nil {
		fmt.Println(erripfs.Error())
		return "",fmt.Errorf("failed to add file %v",erripfs)
	}

	// Check minter authorization - this sample assumes Org1 is the central banker with privilege to mint new tokens
	err := authorization(ctx)
	if err != nil {
		return "",err
	}
	// Mint tokens
	operator, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return "",fmt.Errorf("failed to get client id: %v", err)
	}
	key:=fmt.Sprintf("%d",id)
	value:=&NFT{
		ID: key,
		CID: cid,
		Owner: operator,
	}
	jvalue, err :=json.Marshal(value)
	if err!=nil{
		return "",fmt.Errorf("failed to marshal data %v",err)
	}

	nftkey,err:=ctx.GetStub().CreateCompositeKey(NFTPrefix,[]string{key})
	if err!=nil{
		return "",fmt.Errorf("failed to create composite key %v\n",err)
	}
	err = ctx.GetStub().PutState(nftkey, jvalue)
	if err != nil {
		return "",fmt.Errorf("failed to putstate for key %s , %v",nftkey,err)
	}
	//operate_dc, err := base64.StdEncoding.DecodeString(operator)

	err = addNewNFT(ctx, key)
	if err != nil {
		return "", fmt.Errorf("failed to add new nft to list %v",err)
	}
	// Emit TransferSingle event
	return cid,nil
}

func (s *SmartContract) Transfer(ctx contractapi.TransactionContextInterface, recipient string, id uint64) error {
	sender,err:=ctx.GetClientIdentity().GetID()
	if err != nil {
		return fmt.Errorf("failed to get client id: %v", err)
	}

	if sender == recipient {
		return fmt.Errorf("transfer to self")
	}

	key:=fmt.Sprintf("%d",id)
	nftkey,err:=ctx.GetStub().CreateCompositeKey(NFTPrefix,[]string{key})
	if err!=nil{
		return fmt.Errorf("failed to create composite key %v\n",err)
	}
	jv,err:=ctx.GetStub().GetState(nftkey)
	if err!=nil{
		return fmt.Errorf("failed to getstate for key: %s, %v",nftkey,err)
	}
	v:=&NFT{}
	err=json.Unmarshal(jv,v)
	if err!=nil{
		return fmt.Errorf("failed to unmarshal data %v",err)
	}
	if v.Owner!=sender{
		return fmt.Errorf("access denied, not the owner of this NFT")
	}
	v.Owner=recipient
	jv,err=json.Marshal(v)
	if err!=nil{
		return fmt.Errorf("failed to marshal data %v",err)
	}
	err=ctx.GetStub().PutState(nftkey,jv)
	if err!=nil{
		return fmt.Errorf("failed to putstate for key %s , %v",nftkey,err)
	}
	sender_dc, err := base64.StdEncoding.DecodeString(sender)
	if err != nil {
		return fmt.Errorf("failed to decode: %v",err)
	}
	recipient_dc, err := base64.StdEncoding.DecodeString(recipient)
	if err != nil {
		return fmt.Errorf("failed to decode: %v",err)
	}
	fmt.Printf("===successfully transfer nft from %s to %s==\n",sender_dc,recipient_dc)
	return nil
}

func (s *SmartContract)Query(ctx contractapi.TransactionContextInterface, id uint64) (*NFT, error){
	key:=fmt.Sprintf("%d",id)
	nftkey,err:=ctx.GetStub().CreateCompositeKey(NFTPrefix,[]string{key})
	if err!=nil{
		return nil,fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(nftkey)
	if err!=nil{
		return nil,fmt.Errorf("failed to getstate for key: %s, %v",nftkey,err)
	}
	value:=&NFT{}
	err = json.Unmarshal(jvalue, value)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data %v",err)
	}
	result:=fmt.Sprintf("{tokenID:%v,CID:%v,Owner:%v}",value.ID,value.CID,value.Owner)
	fmt.Println("======query "+result)
	return value,nil
}

func (s *SmartContract)Request(ctx contractapi.TransactionContextInterface, id uint64) (string,error){
	//get target nft
	key:=fmt.Sprintf("%d",id)
	nftkey,err:=ctx.GetStub().CreateCompositeKey(NFTPrefix,[]string{key})
	if err!=nil{
		return "",fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(nftkey)
	if err!=nil{
		return "",fmt.Errorf("failed to getstate for key: %s, %v",nftkey,err)
	}
	value:=&NFT{}
	err = json.Unmarshal(jvalue, value)
	if err != nil {
		return "",fmt.Errorf("failed to unmarshal data %v",err)
	}

	// check if operator has the permission to request data
	operator,_:=ctx.GetClientIdentity().GetID()
	if operator!=value.Owner{
		return "",fmt.Errorf("failed to request data, operator is not the owner {"+operator+" , "+value.Owner+"}")
	}

	//fetch data from ipfs
	cid:=value.CID
	sh := shell.NewShell("ipfs_host:5001")
	reader,err:=sh.Cat(cid)
	if err!=nil{
		return "",fmt.Errorf("failed to get data with cid %s from ipfs %v",cid,err)
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	fmt.Printf("===read file content, {CID:%s, Content:%dB}===\n",cid,buf.Len())
	return buf.String(),nil
}



// authorizationHelper checks minter authorization - this sample assumes Org1 is the central banker with privilege to mint new tokens
func authorization(ctx contractapi.TransactionContextInterface) error {

	clientMSPID, err := ctx.GetClientIdentity().GetMSPID()
	if err != nil {
		return fmt.Errorf("failed to get MSPID: %v", err)
	}
	if clientMSPID != AdmintMSPID {
		return fmt.Errorf("client is not authorized")
	}

	return nil
}
