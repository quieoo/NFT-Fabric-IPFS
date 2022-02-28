package chaincode

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
	shell "github.com/ipfs/go-ipfs-api"
)

const AdmintMSPID = "Org1MSP"
const NFTPrefix = "tokenID~CID~Oaccount"
const BidPrefix = "tokenID~currentPrice~killPrice"
const BalancePrefix = "account~balance"

const NFTBidListsPrefix = "tokenID~tokenID~~"
const NFTListsPrefix = "account~tokenID~tokenID~~"

const MINT_FEE = 10
const MAX_LIFETIME = 3 * 24 * 60
const NonBidder = "暂无竞拍"

const ADDPREFIX = "fabric-uploader-local-file-"
const CATPREFIX = "fabric-check-file-exist-"

// SmartContract provides functions for transferring tokens between accounts
type SmartContract struct {
	contractapi.Contract
}

type NFT struct {
	ID       string
	CID      string
	Owner    string
	FileType string
}
type NFTBid struct {
	TokenID      string
	CurrentPrice uint64
	CurrentOwner string
	KillPrice    uint64
	CreateTime   uint64
	LifeTime     uint64
}
type AccountBalance struct {
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

func (s *SmartContract) Offer(ctx contractapi.TransactionContextInterface, Price uint64, tokenID string) error {
	operator, _ := ctx.GetClientIdentity().GetID()

	ab, err := getAccountBalance(ctx, operator)
	if err != nil {
		return fmt.Errorf("failed to getAccountBalance for Offer: %v\n", err)
	}
	if ab.Balance < Price {
		return fmt.Errorf("failed to Offer, no enough balance, has: %d, offer: %d\n", ab.Balance, Price)
	}

	bid, err := getBid(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to getBid for Offer: %v\n", err)
	}

	if Price < bid.CurrentPrice {
		return fmt.Errorf("failed to Offer, price lower than current max price\n")
	}

	bid.CurrentPrice = Price
	bid.CurrentOwner = operator
	key, _ := ctx.GetStub().CreateCompositeKey(BidPrefix, []string{tokenID})
	jvalue, err := json.Marshal(bid)
	if err != nil {
		return fmt.Errorf("failed to marshal json data for Offer: %v\n")
	}
	err = ctx.GetStub().PutState(key, jvalue)
	if err != nil {
		return fmt.Errorf("failed to PutState for Offer: %v\n", err)
	}
	return nil
}
func (s *SmartContract) FindBidToEnd(ctx contractapi.TransactionContextInterface, currentTime uint64) error {
	tokenIDs, err := getBidsList(ctx)
	if err != nil {
		return fmt.Errorf("failed to get bid by index %v\n", err)
	}

	//first, check bidList: end timeout bids
	for i := 0; i < len(tokenIDs); i++ {
		err := tryEndBid(ctx, tokenIDs[i], currentTime)
		if err != nil {
			return fmt.Errorf("failed to tryEndBid for FindBidToEnd: %v\n", err)
		}
	}
	return nil
}
func (s *SmartContract) TryEndBid(ctx contractapi.TransactionContextInterface, tokenID string, currentTime uint64) error {
	return tryEndBid(ctx, tokenID, currentTime)
}
func tryEndBid(ctx contractapi.TransactionContextInterface, tokenID string, currentTime uint64) error {
	bid, err := getBid(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to getBid for TryEndBid: %v\n", err)
	}
	if currentTime-bid.CreateTime > bid.LifeTime || bid.CurrentPrice >= bid.KillPrice {
		err := endBid(ctx, tokenID, bid.CurrentPrice)
		if err != nil {
			return fmt.Errorf("failed to endBid for TryEndBidv: %v\n", err)
		}
	}
	return nil
}

func (s *SmartContract) AddBid(ctx contractapi.TransactionContextInterface, tokenID string, lowerPrice uint64, upPrice uint64, createTime uint64, lifeMinute uint64) (*NFTBid, error) {
	exists, _ := bidExists(ctx, tokenID)
	if exists {
		return nil, fmt.Errorf("Bid already exists\n")
	}
	if lifeMinute > MAX_LIFETIME {
		return nil, fmt.Errorf("failed to AddBid, life time exceed max time(%d min)\n", MAX_LIFETIME)
	}
	// check operator==NFT.Owner
	operator, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client id: %v", err)
	}
	nft, err := getNFT(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to get nft %v\n", err)
	}
	if nft.Owner != operator {
		return nil, fmt.Errorf("failed to AddBid, not Owner\n")
	}

	life := lifeMinute * 60 * 1000

	fmt.Printf("%v AddBid, with lifeTime %v\n", createTime, life)
	newbid := &NFTBid{
		TokenID:      tokenID,
		CurrentPrice: lowerPrice,
		CurrentOwner: NonBidder,
		KillPrice:    upPrice,
		CreateTime:   createTime,
		LifeTime:     life,
	}
	key, err := ctx.GetStub().CreateCompositeKey(BidPrefix, []string{tokenID})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key %v\n", err)
	}

	jvalue, err := json.Marshal(newbid)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data\n")
	}
	err = ctx.GetStub().PutState(key, jvalue)
	fmt.Printf("AddBid {%s : %v}\n", key, newbid)
	if err != nil {
		return nil, fmt.Errorf("falied to add new Bid %v\n", err)
	}

	err = addBidsToList(ctx, tokenID)
	if err != nil {
		fmt.Println(err)
		return nil, fmt.Errorf("failed to add new bid to list %v\n", err)
	}
	return newbid, nil
}

func (s *SmartContract) GetBidByIndex(ctx contractapi.TransactionContextInterface, index uint64) (*NFTBid, error) {
	tokenIDs, err := getBidsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bid by index %v\n", err)
	}
	if index < 0 || int(index) > len(tokenIDs) {
		return nil, fmt.Errorf("index out of range [0,%d] %v \n", len(tokenIDs), err)
	}
	id := tokenIDs[index]

	bid, err := getBid(ctx, id)
	if err != nil {
	}
	return bid, nil
}

func (s *SmartContract) UpdateBid(ctx contractapi.TransactionContextInterface, tokenID string, newPrice uint64) (*NFTBid, error) {
	operator, _ := ctx.GetClientIdentity().GetID()
	ab, err := getAccountBalance(ctx, operator)
	if err != nil {
		return nil, fmt.Errorf("failed to getAccountBalance for UpdateBid: %v\n", err)
	}
	if ab.Balance < newPrice {
		return nil, fmt.Errorf("no enough balance for bid, remaining: %d, offer: %d\n", ab.Balance, newPrice)
	}

	bid, err := getBid(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to getBid for UpdateBid: %v\n", err)
	}
	if newPrice <= bid.CurrentPrice {
		return nil, fmt.Errorf("failed to UpdateBid, not offer higher price\n")
	}
	bid.CurrentPrice = newPrice
	bid.CurrentOwner = operator

	value, err := json.Marshal(bid)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal bid for UpdateBid: %v\n", err)
	}
	key, _ := ctx.GetStub().CreateCompositeKey(BidPrefix, []string{tokenID})
	err = ctx.GetStub().PutState(key, value)
	if err != nil {
		return nil, fmt.Errorf("failed to PutState for UpdateBid: %v\n", err)
	}
	return bid, nil
}

func deleteBid(ctx contractapi.TransactionContextInterface, tokenID string) error {
	exists, _ := bidExists(ctx, tokenID)
	if !exists {
		return fmt.Errorf("failed to DeleteBid, bid not exist\n")
	}
	return ctx.GetStub().DelState(tokenID)
}

func (s *SmartContract) GetAccountBalance(ctx contractapi.TransactionContextInterface) (*AccountBalance, error) {
	account, _ := ctx.GetClientIdentity().GetID()
	return getAccountBalance(ctx, account)
}

//end bid with offer
//if no bidder, simply remove bid
func endBid(ctx contractapi.TransactionContextInterface, tokenID string, offer uint64) error {
	bid, err := getBid(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to getBid for BidEnd: %v\n", err)
	}
	nft, err := getNFT(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to getBFT for BidEnd: %v\n", err)
	}

	newOwner := bid.CurrentOwner
	oldOwner := nft.Owner
	if newOwner != NonBidder {
		//transfer balance from bid.CurrentOwner to nft.Owner
		newOwnerAccount, err := getAccountBalance(ctx, newOwner)
		if err != nil {
			return fmt.Errorf("failed to getAccountBalance for BidEnd: %v\n", err)
		}
		if newOwnerAccount.Balance < offer {
			return fmt.Errorf("failed to BidEnd, bidder cannot pay the price\n")
		}
		_, err = updateAccountBalance(ctx, newOwner, -1*int(offer))
		if err != nil {
			return fmt.Errorf("failed to take out price from bidder: %v\n", err)
		}
		_, err = updateAccountBalance(ctx, oldOwner, int(offer))
		if err != nil {
			return fmt.Errorf("failed to put in price into owner: %v\n", err)
		}
		//change nft owner
		//add to new owner's list
		err = addNFTToList(ctx, newOwner, tokenID)
		if err != nil {
			return fmt.Errorf("failed to add nft to new owner's list for endbid:%v\n", err)
		}
		//remove for old owner's list
		err = removeNFTFromList(ctx, tokenID, oldOwner)
		if err != nil {
			return fmt.Errorf("failed to remove nft to old owner's list for endbid:%v\n", err)
		}
		nft.Owner = newOwner
		value, err := json.Marshal(nft)
		if err != nil {
			return fmt.Errorf("failed to marshal data for BidEnd: %v\n", err)
		}
		key, _ := ctx.GetStub().CreateCompositeKey(NFTPrefix, []string{tokenID})
		err = ctx.GetStub().PutState(key, value)
		if err != nil {
			return fmt.Errorf("failed to PutState for BidEnd: %v\n", err)
		}

	}
	//clean bid
	err = deleteBid(ctx, tokenID)
	if err != nil {
		return fmt.Errorf("failed to deleteBid for BidEnd: %v\n", err)
	}
	err = removeBidFromList(ctx, tokenID)
	if err != nil {
		return err
	}
	return nil
}

func (s *SmartContract) CanBidEnd(ctx contractapi.TransactionContextInterface, tokenID string, currentTime uint64) (bool, error) {
	exists, err := bidExists(ctx, tokenID)
	if err != nil {
		return false, fmt.Errorf("faled to check bid exists for IsBidTimeout: %v\n", err)
	}
	if !exists {
		return false, fmt.Errorf("cannot end bid, bid not exist\n")
	}
	bid, err := getBid(ctx, tokenID)
	if err != nil {
		return false, fmt.Errorf("failed to get bid for CanBidEnd: %v \n", err)
	}
	if currentTime-bid.CreateTime > bid.LifeTime {
		return true, nil
	}
	return false, nil
}

func (s *SmartContract) IsNFTOnSale(ctx contractapi.TransactionContextInterface, tokenID string) (bool, error) {
	nft_exists, err := nftExists(ctx, tokenID)
	if err != nil {
		return false, fmt.Errorf("failed to getNFT for checking IsNFTOnSale: %v\n", err)
	}
	if !nft_exists {
		return false, fmt.Errorf("nft not exists\n")
	}
	return bidExists(ctx, tokenID)

}

type TotalBidsWithTimeOutCheckResult struct {
	TotalAliveBid int
	HasTimeOutBid bool
}

func (s *SmartContract) TotalBidsWithTimeOutCheck(ctx contractapi.TransactionContextInterface, currentTime uint64) (*TotalBidsWithTimeOutCheckResult, error) {
	tokenIDs, err := getBidsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get bid by index %v\n", err)
	}
	result := &TotalBidsWithTimeOutCheckResult{0, false}
	var activeTokenIDs []string
	//first, check bidList: end timeout bids
	for i := 0; i < len(tokenIDs); i++ {
		tokenID := tokenIDs[i]
		bid, err := getBid(ctx, tokenID)
		if err != nil {
			return nil, fmt.Errorf("failed to getBid for TotalBidsWithTimeOutCheck: %v\n", err)
		}
		if currentTime-bid.CreateTime > bid.LifeTime {
			//timeout
			result.HasTimeOutBid = true

		} else {
			activeTokenIDs = append(activeTokenIDs, tokenID)
		}
	}
	result.TotalAliveBid = len(activeTokenIDs)
	return result, nil
}

func (s *SmartContract) IsNFTExist(ctx contractapi.TransactionContextInterface, tokenID string) (bool, error) {
	return nftExists(ctx, tokenID)
}
func (s *SmartContract) TotalNFTs(ctx contractapi.TransactionContextInterface) (int, error) {
	account, _ := ctx.GetClientIdentity().GetID()
	tokenIDs, err := getNFTList(ctx, account)
	if err != nil {
		return 0, fmt.Errorf("failed to get nft list %v\n", err)
	}
	return len(tokenIDs), nil
}

func getBidsList(ctx contractapi.TransactionContextInterface) ([]string, error) {
	key, err := ctx.GetStub().CreateCompositeKey(NFTBidListsPrefix, []string{""})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key %v\n", err)
	}
	jvalue, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to getstate for key: %s, %v", key, err)
	}
	value := string(jvalue)
	strs := strings.Fields(value)
	fmt.Printf("get all bids %v\n", strs)
	return strs, nil
}

func removeBidFromList(ctx contractapi.TransactionContextInterface, tokenID string) error {
	key, err := ctx.GetStub().CreateCompositeKey(NFTBidListsPrefix, []string{""})
	if err != nil {
		return fmt.Errorf("failed to create composite key %v\n", err)
	}
	jvalue, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to getstate for key: %s, %v", key, err)
	}
	value := string(jvalue)
	strs := strings.Fields(value)

	newstring := ""
	skip := false
	for i := 0; i < len(strs); i++ {
		if strs[i] != tokenID {
			newstring += strs[i] + " "
		} else {
			skip = true
		}
	}

	if !skip {
		return fmt.Errorf("failed to removeBidFromList, tokenID not in BidList\n")
	}
	return ctx.GetStub().PutState(key, []byte(newstring))
}

func removeNFTFromList(ctx contractapi.TransactionContextInterface, tokenID string, account string) error {
	key, err := ctx.GetStub().CreateCompositeKey(NFTListsPrefix, []string{account})
	if err != nil {
		return fmt.Errorf("failed to create composite key %v\n", err)
	}
	jvalue, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to getstate for key: %s, %v", key, err)
	}
	value := string(jvalue)
	strs := strings.Fields(value)
	fmt.Printf("@@@@@@removeNFTFromList: %v from %v\n", tokenID, strs)
	newstring := ""
	skip := false
	for i := 0; i < len(strs); i++ {
		if strs[i] != tokenID {
			newstring += strs[i] + " "
		} else {
			skip = true
		}
	}

	if !skip {
		return fmt.Errorf("failed to removeNFTFromList, tokenID not in NFTList\n")
	}
	return ctx.GetStub().PutState(key, []byte(newstring))
}

func getNFTList(ctx contractapi.TransactionContextInterface, account string) ([]string, error) {
	key, _ := ctx.GetStub().CreateCompositeKey(NFTListsPrefix, []string{account})
	jvalue, err := ctx.GetStub().GetState(key)
	if err != nil {
		return nil, fmt.Errorf("failed to getstate for key: %s, %v", key, err)
	}
	value := string(jvalue)
	strs := strings.Fields(value)
	return strs, nil
}
func addBidsToList(ctx contractapi.TransactionContextInterface, newTokenID string) error {
	key, err := ctx.GetStub().CreateCompositeKey(NFTBidListsPrefix, []string{""})
	if err != nil {
		return fmt.Errorf("failed to create composite key %v\n", err)
	}
	jvalue, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to getstate for key: %s, %v", key, err)
	}
	value := string(jvalue)
	//check existence
	bids := strings.Fields(value)
	for i := 0; i < len(bids); i++ {
		if bids[i] == newTokenID {
			return nil
		}
	}
	value += " " + newTokenID
	jvalue = []byte(value)
	fmt.Printf("AddBidToList, %v\n", value)
	return ctx.GetStub().PutState(key, jvalue)
}

func addNFTToList(ctx contractapi.TransactionContextInterface, account string, tokenID string) error {
	key, _ := ctx.GetStub().CreateCompositeKey(NFTListsPrefix, []string{account})

	jvalue, err := ctx.GetStub().GetState(key)
	if err != nil {
		return fmt.Errorf("failed to GetState for addNFTToList: %v\n", err)
	}
	value := string(jvalue)
	nfts := strings.Fields(value)
	for i := 0; i < len(nfts); i++ {
		if nfts[i] == tokenID {
			return nil
		}
	}

	value += " " + tokenID

	jvalue = []byte(value)
	return ctx.GetStub().PutState(key, jvalue)
}

func bidExists(ctx contractapi.TransactionContextInterface, tokenID string) (bool, error) {
	lst, err := getBidsList(ctx)
	if err != nil {
		return false, fmt.Errorf("faled to getBidtList for IsNFTOnSale: %v\n", err)
	}
	bidexist := false
	for _, b := range lst {
		if b == tokenID {
			bidexist = true
		}
	}
	return bidexist, nil
}

func nftExists(ctx contractapi.TransactionContextInterface, tokenID string) (bool, error) {
	operator, _ := ctx.GetClientIdentity().GetID()
	lst, err := getNFTList(ctx, operator)
	if err != nil {
		return false, fmt.Errorf("faled to getBidtList for IsNFTOnSale: %v\n", err)
	}
	bidexist := false
	for _, b := range lst {
		if b == tokenID {
			bidexist = true
		}
	}
	return bidexist, nil
}
func getBid(ctx contractapi.TransactionContextInterface, tokenID string) (*NFTBid, error) {
	nftkey, err := ctx.GetStub().CreateCompositeKey(BidPrefix, []string{tokenID})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key %v\n", err)
	}
	jvalue, err := ctx.GetStub().GetState(nftkey)

	if err != nil {
		return nil, fmt.Errorf("failed to getstate for key: %s, %v", nftkey, err)
	}
	value := &NFTBid{}
	err = json.Unmarshal(jvalue, value)
	fmt.Printf("GetBid (%s: %v)\n", nftkey, jvalue)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data %v", err)
	}
	return value, nil
}

func (s *SmartContract) InitAccountBalance(ctx contractapi.TransactionContextInterface, account string, balance uint64) error {
	err := authorization(ctx)
	if err != nil {
		return fmt.Errorf("failed to InitAccountBalance, not authenticated: %v\n", err)
	}

	ab := &AccountBalance{account, balance}
	key, _ := ctx.GetStub().CreateCompositeKey(BalancePrefix, []string{account})
	jvalue, err := json.Marshal(ab)
	if err != nil {
		return fmt.Errorf("failed to marshal data for newAccountBalance %v", err)
	}
	err = ctx.GetStub().PutState(key, jvalue)
	if err != nil {
		return fmt.Errorf("failed to PutState for newAccountBalance: %v\n", err)
	}
	return nil
}

func getAccountBalance(ctx contractapi.TransactionContextInterface, account string) (*AccountBalance, error) {
	key, err := ctx.GetStub().CreateCompositeKey(BalancePrefix, []string{account})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key %v\n", err)
	}
	jvalue, err := ctx.GetStub().GetState(key)
	fmt.Printf("Get Account (%v,%v)\n", key, jvalue)
	if err != nil {
		return nil, fmt.Errorf("failed to getstate for key: %s, %v", key, err)
	}
	if len(jvalue) == 0 {
		return nil, fmt.Errorf("Account not exist\n")
	}
	value := &AccountBalance{}
	err = json.Unmarshal(jvalue, value)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data %v", err)
	}
	return value, nil
}

func updateAccountBalance(ctx contractapi.TransactionContextInterface, account string, balance int) (*AccountBalance, error) {
	/*
		//only authored operator can update account balance
		err := authorization(ctx)
		if err != nil {
			return nil,err
		}*/

	oldAccount, err := getAccountBalance(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("failed to getAccountBalance: %v\n", err)
	}
	newbalance := int64(oldAccount.Balance) + int64(balance)
	if newbalance < 0 {
		newbalance = 0
	}
	value := &AccountBalance{
		Account: account,
		Balance: uint64(newbalance),
	}
	jvalue, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data %v", err)
	}

	key, err := ctx.GetStub().CreateCompositeKey(BalancePrefix, []string{account})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key %v\n", err)
	}
	err = ctx.GetStub().PutState(key, jvalue)
	fmt.Printf("Update Account (%v,%v)\n", key, value)
	if err != nil {
		return nil, fmt.Errorf("failed to PutState %v\n", err)
	}
	return value, nil
}

func (s *SmartContract) GetBid(ctx contractapi.TransactionContextInterface, tokenID string) (*NFTBid, error) {
	return getBid(ctx, tokenID)
}

func (s *SmartContract) MintWithFile(ctx contractapi.TransactionContextInterface, tokenID string, ftype string, hash string) (*NFT, error) {
	//check operator balance
	operator, err := ctx.GetClientIdentity().GetID()
	if err != nil {
		return nil, fmt.Errorf("failed to get client id: %v", err)
	}
	balance, err := getAccountBalance(ctx, operator)
	if err != nil {
		return nil, err
	}
	if balance.Balance < MINT_FEE {
		return nil, fmt.Errorf("failed to MintWithFile, no enough balance. has: %d, need at least: %d\n", balance, MINT_FEE)
	}

	sh := shell.NewShell("ipfs_host:5001")

	cid, erripfs := sh.Add(strings.NewReader(ADDPREFIX + tokenID + "." + ftype))
	fmt.Printf("ADD to IPFS: %s%s.%s", ADDPREFIX, tokenID, ftype)
	if erripfs != nil {
		fmt.Println(erripfs.Error())
		fmt.Println("trying to find file in IPFS network...")
		cat, err := sh.Cat(CATPREFIX + hash)
		if err != nil {
			fmt.Println("failed to cat file: " + err.Error())
			return nil, fmt.Errorf("can't add or cat file: %v\n", err.Error())
		}
		buf := make([]byte, 1<<10)
		_, err = cat.Read(buf)
		if err != nil {
			fmt.Println(err.Error())
			return nil, fmt.Errorf("read buffer error: %v\n", err.Error())
		}
		err = cat.Close()
		if err != nil {
			fmt.Println(err.Error())
			return nil, fmt.Errorf("failed to close reader: %v\n", err.Error())
		}

		providers := strings.Split(string(buf), " ")
		if providers[0] != "find" {
			return nil, fmt.Errorf("failed to find providers: %v\n", err.Error())
		}
		//return nil, fmt.Errorf("failed to add file %v", erripfs)
	}else{
		//add successfully, means the local file exists in server
		if cid!=hash{
			fmt.Println("Mint Error, since file content has changed")
			return nil, fmt.Errorf("Mint Error, since file content has changed")
		}
	}


	// Mint tokens
	value := &NFT{
		ID:       tokenID,
		CID:      cid,
		Owner:    operator,
		FileType: ftype,
	}
	jvalue, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data %v", err)
	}

	nftkey, err := ctx.GetStub().CreateCompositeKey(NFTPrefix, []string{tokenID})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key %v\n", err)
	}
	err = ctx.GetStub().PutState(nftkey, jvalue)
	if err != nil {
		return nil, fmt.Errorf("failed to PutState for MintWithFile %v\n", err)
	}
	_, err = updateAccountBalance(ctx, operator, -1*MINT_FEE)
	if err != nil {
		return nil, fmt.Errorf("failed to updateAccountBalance for MintWithFile: %v\n", err)
	}
	err = addNFTToList(ctx, operator, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to addNFTToList: %v", err)
	}
	// Emit TransferSingle event
	return value, nil
}

func (s *SmartContract) TransferNFT(ctx contractapi.TransactionContextInterface, recipientToken string, tokenID string) error {
	// only authorized account has right to transfer NFT (admin in org1)
	err := authorization(ctx)
	if err != nil {
		return fmt.Errorf("failed to TransfetNFT, not authenticated: %v\n", err)
	}
	nftkey, err := ctx.GetStub().CreateCompositeKey(NFTPrefix, []string{tokenID})
	if err != nil {
		return fmt.Errorf("failed to create composite key %v\n", err)
	}
	jv, err := ctx.GetStub().GetState(nftkey)
	if err != nil {
		return fmt.Errorf("failed to getstate for key: %s, %v", nftkey, err)
	}
	v := &NFT{}
	err = json.Unmarshal(jv, v)
	if err != nil {
		return fmt.Errorf("failed to unmarshal data %v", err)
	}
	v.Owner = recipientToken
	jv, err = json.Marshal(v)
	if err != nil {
		return fmt.Errorf("failed to marshal data %v", err)
	}
	err = ctx.GetStub().PutState(nftkey, jv)
	if err != nil {
		return fmt.Errorf("failed to putstate for key %s , %v", nftkey, err)
	}
	fmt.Printf("===successfully transfer nft to %s==\n", recipientToken)
	return nil
}

func (s *SmartContract) GetNFTByIndex(ctx contractapi.TransactionContextInterface, index uint64) (*NFT, error) {
	operator, _ := ctx.GetClientIdentity().GetID()
	nfts, err := getNFTList(ctx, operator)
	if err != nil {
		return nil, fmt.Errorf("failed to getNFTList for GetNFTByIndex: %v\n", err)
	}
	if len(nfts) < 0 {
		return nil, fmt.Errorf("failed to getNFTByIndex, no nfts in current account\n")
	}
	if index < 0 || int(index) > len(nfts) {
		return nil, fmt.Errorf("getNFTByIndex, index out of range [0,%d] %v \n", len(nfts), err)
	}
	id := nfts[index]
	nft, err := getNFT(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to getNFT for GetNFTByIndex: %v\n", err)
	}
	return nft, nil
}

func getNFT(ctx contractapi.TransactionContextInterface, id string) (*NFT, error) {
	nftkey, err := ctx.GetStub().CreateCompositeKey(NFTPrefix, []string{id})
	if err != nil {
		return nil, fmt.Errorf("failed to create composite key %v\n", err)
	}
	jvalue, err := ctx.GetStub().GetState(nftkey)
	if err != nil {
		return nil, fmt.Errorf("failed to getstate for key: %s, %v", nftkey, err)
	}
	value := &NFT{}
	err = json.Unmarshal(jvalue, value)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal data %v", err)
	}
	return value, nil
}

func (s *SmartContract) GetNFTByID(ctx contractapi.TransactionContextInterface, tokenID string) (*NFT, error) {
	value, err := getNFT(ctx, tokenID)
	if err != nil {
		return nil, fmt.Errorf("failed to getNFT %v\n", err)
	}
	result := fmt.Sprintf("{tokenID:%v,CID:%v,Owner:%v}", value.ID, value.CID, value.Owner)
	fmt.Println("======query " + result)
	return value, nil
}

func (s *SmartContract) Request(ctx contractapi.TransactionContextInterface, tokenID string) (string, error) {
	//get target nft
	nftkey, err := ctx.GetStub().CreateCompositeKey(NFTPrefix, []string{tokenID})
	if err != nil {
		return "", fmt.Errorf("failed to create composite key %v\n", err)
	}
	jvalue, err := ctx.GetStub().GetState(nftkey)
	if err != nil {
		return "", fmt.Errorf("failed to getstate for key: %s, %v", nftkey, err)
	}
	value := &NFT{}
	err = json.Unmarshal(jvalue, value)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal data %v", err)
	}

	/*
		// check if operator has the permission to request data
		operator,_:=ctx.GetClientIdentity().GetID()
		if operator!=value.Owner{
			return "",fmt.Errorf("failed to request data, operator is not the owner {"+operator+" , "+value.Owner+"}")
		}
	*/
	//fetch data from ipfs
	cid := value.CID
	sh := shell.NewShell("ipfs_host:5001")
	reader, err := sh.Cat(cid)
	if err != nil {
		return "", fmt.Errorf("failed to get data with cid %s from ipfs %v", cid, err)
	}
	defer reader.Close()

	buf := new(bytes.Buffer)
	buf.ReadFrom(reader)
	fmt.Printf("===read file content, {CID:%s, Content:%dB}===\n", cid, buf.Len())
	return buf.String(), nil
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
