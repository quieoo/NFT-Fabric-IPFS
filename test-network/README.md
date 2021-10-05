## Running the test network for nft_ipfs

```bash
./000_bringDown.sh
```
Shut down ipfs container and the rest in test_network
```bash
./001_bringUP.sh
```
Bring up the env, including create containers, deploy chaincode, register clients
```bash
./01_mintWithFile.sh
```
Use `Miner` to call create NFT token
```bash
./02_getRecipientID.sh
```
Get ClientID of `Recipient`, inject the output as input to `03_transferToken.sh`, so as chaincode knows the target account of transfer
```bash
./03_transferToken.sh
```
`Minter` transfer the minted nft token to `Recipient`
```bash
./04_query_request.sh
```
`Recipient` now check the owner of transferred token and request the token as owner
