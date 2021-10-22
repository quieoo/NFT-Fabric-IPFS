package chaincode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	shell "github.com/ipfs/go-ipfs-api"
	"strings"
	"time"
)

const AdmintMSPID = "Org1MSP"
const NFTPrefix="tokenID~CID~Oaccount"
const BidPrefix="tokenID~currentPrice~killPrice"
const BalancePrefix="account~balance"

const NFTBidListsPrefix="tokenID~tokenID~~"
const NFTListsPrefix="account~tokenID~tokenID~~"

const MINT_FEE=10
const MAX_LIFETIME=3*24*60
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
	CurrentOwner string
	KillPrice uint64
	CreateTime time.Time
	LifeTime time.Duration
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

func(s *SmartContract) AddBid(ctx contractapi.TransactionContextInterface, tokenID string, lowerPrice uint64, upPrice uint64, lifeMinute uint64)(*NFTBid,error){
	exists,_:=bidExists(ctx,tokenID)
	if exists{
		return nil,fmt.Errorf("Bid already exists\n")
	}
	if lifeMinute>MAX_LIFETIME{
		return nil,fmt.Errorf("failed to AddBid, life time exceed max time(%d min)\n",MAX_LIFETIME)
	}
	// check operator==NFT.Owner
	operator, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return nil,fmt.Errorf("failed to get client id: %v", err)
	}
	nft,err:=getNFT(ctx,tokenID)
	if err!=nil{
		return nil,fmt.Errorf("failed to get nft %v\n",err)
	}
	if nft.Owner!=operator{
		return nil,fmt.Errorf("failed to AddBid, not Owner\n")
	}

	life:=time.Duration(lifeMinute)*time.Minute
	newbid:=&NFTBid{
		TokenID: tokenID,
		CurrentPrice: lowerPrice,
		CurrentOwner: "暂无竞拍",
		KillPrice: upPrice,
		CreateTime: time.Now(),
		LifeTime: life,
	}
	key,err:=ctx.GetStub().CreateCompositeKey(BidPrefix,[]string{tokenID})
	if err!=nil{
		return nil,fmt.Errorf("failed to create composite key %v\n",err)
	}

	jvalue,err:=json.Marshal(newbid)
	if err!=nil{
		return nil,fmt.Errorf("failed to marshal data\n")
	}
	err=ctx.GetStub().PutState(key,jvalue)
	fmt.Printf("^^^^^^add bid (%v,%v)\n",key,jvalue)

	if err!=nil{
		return nil,fmt.Errorf("falied to add new Bid %v\n",err)
	}

	err=addBidsToList(ctx,tokenID)
	if err!=nil{
		return nil,fmt.Errorf("failed to add new bid to list %v\n",err)
	}
	return newbid,nil
}

func (s *SmartContract) GetBidByIndex(ctx contractapi.TransactionContextInterface, index uint64)(*NFTBid,error){
	tokenIDs,err:=getBidsList(ctx)
	if err!=nil{
		return nil,fmt.Errorf("failed to get bid by index %v\n",err)
	}
	if index<0 || int(index) > len(tokenIDs){
		return nil,fmt.Errorf("index out of range [0,%d] %v \n",len(tokenIDs),err)
	}
	id:=tokenIDs[index]

	fmt.Printf("^^^^^^^^^^^^^^^GetBidByIndex %s\n",id)

	bid,err:=getBid(ctx,id)
	if err!=nil{
		return nil,fmt.Errorf("failed to get bid %v\n",err)
	}
	fmt.Printf("^^^^^^^^^^^^debug %v\n",bid)
	return bid,nil
}

func (s *SmartContract) UpdateBid(ctx contractapi.TransactionContextInterface,tokenID string, newPrice uint64) (*NFTBid,error){
	operator,_:=ctx.GetClientIdentity().GetID()
	ab,err:=getAccountBalance(ctx,operator)
	if err!=nil{
		return nil,fmt.Errorf("failed to getAccountBalance for UpdateBid: %v\n",err)
	}
	if ab.Balance<newPrice{
		return nil,fmt.Errorf("no enough balance for bid, remaining: %d, offer: %d\n",ab.Balance,newPrice)
	}

	bid,err:=getBid(ctx,tokenID)
	if err!=nil{
		return nil,fmt.Errorf("failed to getBid for UpdateBid: %v\n",err)
	}
	if newPrice<=bid.CurrentPrice{
		return nil,fmt.Errorf("failed to UpdateBid, not offer higher price\n")
	}
	bid.CurrentPrice=newPrice
	bid.CurrentOwner=operator

	value,err:=json.Marshal(bid)
	if err!=nil{
		return nil,fmt.Errorf("failed to marshal bid for UpdateBid: %v\n",err)
	}
	key,_:=ctx.GetStub().CreateCompositeKey(BidPrefix,[]string{tokenID})
	err=ctx.GetStub().PutState(key,value)
	if err!=nil{
		return nil,fmt.Errorf("failed to PutState for UpdateBid: %v\n",err)
	}
	return bid,nil
}

func deleteBid(ctx contractapi.TransactionContextInterface, tokenID string)error{
	exists,_:=bidExists(ctx,tokenID)
	if !exists {
		return fmt.Errorf("failed to DeleteBid, bid not exist\n")
	}
	return ctx.GetStub().DelState(tokenID)
}

func (s *SmartContract)GetAccountBalance(ctx contractapi.TransactionContextInterface)(*AccountBalance,error){
	account,_:=ctx.GetClientIdentity().GetID()
	return getAccountBalance(ctx,account)
}

func (s *SmartContract)BidEnd(ctx contractapi.TransactionContextInterface, tokenID string, offer uint64)error{
	bid,err:=getBid(ctx,tokenID)
	if err!=nil{
		return fmt.Errorf("failed to getBid for BidEnd: %v\n",err)
	}
	nft,err:=getNFT(ctx,tokenID)
	if err!=nil{
		return fmt.Errorf("failed to getBFT for BidEnd: %v\n",err)
	}
	//transfer balance
	newOwner:=bid.CurrentOwner
	newOwnerAccount, err := getAccountBalance(ctx, newOwner)
	if err != nil {
		return fmt.Errorf("failed to getAccountBalance for BidEnd: %v\n",err)
	}
	if newOwnerAccount.Balance<offer{
		return fmt.Errorf("failed to BidEnd, bidder cannot pay the price\n")
	}
	_,err=updateAccountBalance(ctx,newOwner,-1*int(offer))
	if err!=nil{
		return fmt.Errorf("failed to take out price from bidder: %v\n",err)
	}
	_,err=updateAccountBalance(ctx,nft.Owner,int(offer))
	if err!=nil{
		return fmt.Errorf("failed to put in price into owner: %v\n",err)
	}

	//clean bid
	err=deleteBid(ctx,tokenID)
	if err!=nil{
		return fmt.Errorf("failed to deleteBid for BidEnd: %v\n",err)
	}
	err=removeBidFromList(ctx,tokenID)
	if err!=nil{
		return err
	}

	//change nft owner
	nft.Owner=newOwner
	value,err:=json.Marshal(nft)
	if err!=nil{
		return fmt.Errorf("failed to marshal data for BidEnd: %v\n",err)
	}
	key,_:=ctx.GetStub().CreateCompositeKey(NFTPrefix,[]string{tokenID})
	return ctx.GetStub().PutState(key,value)
}



func (s *SmartContract) TotalBids(ctx contractapi.TransactionContextInterface)(int,error){
	tokenIDs,err:=getBidsList(ctx)
	if err!=nil{
		return 0,fmt.Errorf("failed to get bid list %v\n",err)
	}
	return len(tokenIDs),nil
}
func (s *SmartContract) TotalNFTs(ctx contractapi.TransactionContextInterface, account string)(int,error){
	tokenIDs,err:=getNFTList(ctx,account)
	if err!=nil{
		return 0,fmt.Errorf("failed to get nft list %v\n",err)
	}
	return len(tokenIDs),nil
}


func getBidsList(ctx contractapi.TransactionContextInterface)([]string,error){
	key,err:=ctx.GetStub().CreateCompositeKey(NFTBidListsPrefix,[]string{""})
	if err!=nil{
		return nil,fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(key)
	if err!=nil{
		return nil,fmt.Errorf("failed to getstate for key: %s, %v",key,err)
	}
	value:=string(jvalue)
	strs:=strings.Split(value," ")
	fmt.Printf("get all bids %v\n",strs)
	return strs,nil
}

func removeBidFromList(ctx contractapi.TransactionContextInterface, tokenID string)error{
	key,err:=ctx.GetStub().CreateCompositeKey(NFTBidListsPrefix,[]string{""})
	if err!=nil{
		return fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(key)
	if err!=nil{
		return fmt.Errorf("failed to getstate for key: %s, %v",key,err)
	}
	value:=string(jvalue)
	strs:=strings.Split(value," ")

	newstring:=""
	skip:=false
	for i:=0;i<len(strs);i++{
		if strs[i]!=tokenID{
			newstring+=strs[i]+" "
		}else{
			skip=true
		}
	}

	if !skip{
		return fmt.Errorf("failed to removeBidFromList, tokenID not in BidList\n")
	}
	return ctx.GetStub().PutState(key,[]byte(newstring))
}

func getNFTList(ctx contractapi.TransactionContextInterface,account string)([]string,error){
	key,_:=ctx.GetStub().CreateCompositeKey(NFTListsPrefix,[]string{account})
	jvalue,err:=ctx.GetStub().GetState(key)
	if err!=nil{
		return nil,fmt.Errorf("failed to getstate for key: %s, %v",key,err)
	}
	value:=string(jvalue)
	strs:=strings.Split(value," ")
	fmt.Printf("get all nfts %v\n",strs)
	return strs,nil
}
func addBidsToList(ctx contractapi.TransactionContextInterface, newTokenID string)error{
	key,err:=ctx.GetStub().CreateCompositeKey(NFTBidListsPrefix,[]string{""})
	if err!=nil{
		return fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(key)
	if err!=nil{
		return fmt.Errorf("failed to getstate for key: %s, %v",key,err)
	}
	value:=string(jvalue)
	//check existence
	bids:=strings.Split(value," ")
	for i:=0;i<len(bids);i++{
		if bids[i]==newTokenID{
			return nil
		}
	}
	value=value+newTokenID+" "
	jvalue=[]byte(value)
	return ctx.GetStub().PutState(key,jvalue)
}

func addNFTToList(ctx contractapi.TransactionContextInterface, account string, tokenID string) error{
	key,_:=ctx.GetStub().CreateCompositeKey(NFTListsPrefix,[]string{account})

	jvalue,err:=ctx.GetStub().GetState(key)
	if err!=nil{
		return fmt.Errorf("failed to GetState for addNFTToList: %v\n",err)
	}
	value:=string(jvalue)
	nfts:=strings.Split(value," ")
	for i:=0;i<len(nfts);i++{
		if nfts[i]==tokenID{
			return nil
		}
	}
	value=value+tokenID+" "
	jvalue=[]byte(value)
	return ctx.GetStub().PutState(key,jvalue)
}



func bidExists(ctx contractapi.TransactionContextInterface, tokenID string)(bool,error){
	nftkey,_:=ctx.GetStub().CreateCompositeKey(BidPrefix,[]string{tokenID})
	jvalue,err:=ctx.GetStub().GetState(nftkey)
	if err!=nil{
		return false,fmt.Errorf("failed to read from world state: %v", err)
	}
	return jvalue!=nil,nil
}
func getBid(ctx contractapi.TransactionContextInterface, tokenID string)(*NFTBid, error){
	nftkey,err:=ctx.GetStub().CreateCompositeKey(BidPrefix,[]string{tokenID})
	if err!=nil{
		return nil,fmt.Errorf("failed to create composite key %v\n",err)
	}
	jvalue,err:=ctx.GetStub().GetState(nftkey)
	fmt.Printf("^^^^^^get bid (%v,%v)\n",nftkey,jvalue)

	if err!=nil{
		return nil,fmt.Errorf("failed to getstate for key: %s, %v",nftkey,err)
	}
	value:=&NFTBid{}
	err = json.Unmarshal(jvalue, value)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data %v",err)
	}
	return value,nil
}

func(s *SmartContract) InitAccountBalance(ctx contractapi.TransactionContextInterface, account string, balance uint64) error{
	err:=authorization(ctx)
	if err!=nil{
		return fmt.Errorf("failed to InitAccountBalance, not authenticated: %v\n",err)
	}

	ab:=&AccountBalance{account,balance}
	key,_:=ctx.GetStub().CreateCompositeKey(BalancePrefix,[]string{account})
	jvalue, err :=json.Marshal(ab)
	if err!=nil{
		return fmt.Errorf("failed to marshal data for newAccountBalance %v",err)
	}
	err=ctx.GetStub().PutState(key,jvalue)
	if err!=nil{
		return fmt.Errorf("failed to PutState for newAccountBalance: %v\n",err)
	}
	return nil
}

func getAccountBalance(ctx contractapi.TransactionContextInterface, account string) (*AccountBalance,error){
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

func updateAccountBalance(ctx contractapi.TransactionContextInterface, account string, balance int) (*AccountBalance,error){
	/*
	//only authored operator can update account balance
	err := authorization(ctx)
	if err != nil {
		return nil,err
	}*/

	oldAccount,err:=getAccountBalance(ctx,account)
	if err!=nil{
		return nil,fmt.Errorf("failed to getAccountBalance: %v\n",err)
	}
	newbalance:=int64(oldAccount.Balance)+int64(balance)
	if newbalance<0{
		newbalance=0
	}
	value:=&AccountBalance{
		Account: account,
		Balance: uint64(newbalance),
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
	return getBid(ctx,tokenID)
}



func (s *SmartContract) MintWithFile (ctx contractapi.TransactionContextInterface, tokenID string, content string) (*NFT,error){
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
	//check operator balance
	operator, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return nil,fmt.Errorf("failed to get client id: %v", err)
	}
	balance,err:=getAccountBalance(ctx,operator)
	if err!=nil{
		return nil,err
	}
	if balance.Balance<MINT_FEE{
		return nil,fmt.Errorf("failed to MintWithFile, no enough balance. has: %d, need at least: %d\n",balance,MINT_FEE)
	}

	sh := shell.NewShell("ipfs_host:5001")

	cid,erripfs:=sh.Add(strings.NewReader(content))
	if erripfs != nil {
		fmt.Println(erripfs.Error())
		return nil,fmt.Errorf("failed to add file %v",erripfs)
	}
	// Mint tokens
	value:=&NFT{
		ID: tokenID,
		CID: cid,
		Owner: operator,
	}
	jvalue, err :=json.Marshal(value)
	if err!=nil{
		return nil,fmt.Errorf("failed to marshal data %v",err)
	}

	nftkey,err:=ctx.GetStub().CreateCompositeKey(NFTPrefix,[]string{tokenID})
	if err!=nil{
		return nil,fmt.Errorf("failed to create composite key %v\n",err)
	}
	err = ctx.GetStub().PutState(nftkey, jvalue)
	if err != nil {
		return nil,fmt.Errorf("failed to PutState for MintWithFile %v\n",err)
	}
	_,err=updateAccountBalance(ctx,operator,-1*MINT_FEE)
	if err!=nil{
		return nil,fmt.Errorf("failed to updateAccountBalance for MintWithFile: %v\n",err)
	}
	err=addNFTToList(ctx,operator,tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to addNFTToList: %v",err)
	}
	// Emit TransferSingle event
	return value,nil
}

func (s *SmartContract) TransferNFT(ctx contractapi.TransactionContextInterface, recipientToken string, tokenID string) error {
	// only authorized account has right to transfer NFT (admin in org1)
	err:=authorization(ctx)
	if err!=nil{
		return fmt.Errorf("failed to TransfetNFT, not authenticated: %v\n",err)
	}
	nftkey,err:=ctx.GetStub().CreateCompositeKey(NFTPrefix,[]string{tokenID})
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
	v.Owner=recipientToken
	jv,err=json.Marshal(v)
	if err!=nil{
		return fmt.Errorf("failed to marshal data %v",err)
	}
	err=ctx.GetStub().PutState(nftkey,jv)
	if err!=nil{
		return fmt.Errorf("failed to putstate for key %s , %v",nftkey,err)
	}
	fmt.Printf("===successfully transfer nft to %s==\n",recipientToken)
	return nil
}

func (s *SmartContract) GetNFTByIndex(ctx contractapi.TransactionContextInterface, index uint64)(*NFT,error){
	operator,_:=ctx.GetClientIdentity().GetID()
	nfts,err:=getNFTList(ctx,operator)
	if err!=nil{
		return nil,fmt.Errorf("failed to getNFTList for GetNFTByIndex: %v\n",err)
	}
	if index<0 || int(index) > len(nfts){
		return nil,fmt.Errorf("getNFTByIndex, index out of range [0,%d] %v \n",len(nfts),err)
	}
	id:=nfts[index]
	nft,err:=getNFT(ctx,id)
	if err!=nil{
		return nil,fmt.Errorf("failed to getNFT for GetNFTByIndex: %v\n",err)
	}
	return nft,nil
}

func getNFT(ctx contractapi.TransactionContextInterface, id string)(*NFT,error){
	nftkey,err:=ctx.GetStub().CreateCompositeKey(NFTPrefix,[]string{id})
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
	return value,nil
}

func (s *SmartContract)GetNFTByID(ctx contractapi.TransactionContextInterface, tokenID string) (*NFT, error){
	value,err:=getNFT(ctx,tokenID)
	if err!=nil{
		return nil,fmt.Errorf("failed to getNFT %v\n",err)
	}
	result:=fmt.Sprintf("{tokenID:%v,CID:%v,Owner:%v}",value.ID,value.CID,value.Owner)
	fmt.Println("======query "+result)
	return value,nil
}


func (s *SmartContract)Request(ctx contractapi.TransactionContextInterface, tokenID string) (string,error){
	//get target nft
	nftkey,err:=ctx.GetStub().CreateCompositeKey(NFTPrefix,[]string{tokenID})
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

	/*
	// check if operator has the permission to request data
	operator,_:=ctx.GetClientIdentity().GetID()
	if operator!=value.Owner{
		return "",fmt.Errorf("failed to request data, operator is not the owner {"+operator+" , "+value.Owner+"}")
	}
*/
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
