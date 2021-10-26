[//]: # (SPDX-License-Identifier: CC-BY-4.0)

# System Architecture
![image](https://github.com/quieoo/nft_fabric_ipfs/blob/main/architecture/architecture_nft_ipfs.drawio%20(2).png)

[Why build this app?](https://github.com/quieoo/nft_fabric_ipfs/blob/main/architecture/nft.pptx)

# Demo Start
## Clone repo locally
```bash
git clone https://github.com/quieoo/nft_fabric_ipfs.git
```
## Hyperledger Fabric and IPFS docker daemon bring up
```bash
cd nft_fabric_ipfs/test-network/
./000_bringDown.sh  
./001_bringUP.sh 
```
This will clean the environment and build hyperledger fabric peers and ipfs docker daemon, deploy chaincode "FI-NFT"

## Start Web Server
````bash
cd nft_fabric_ipfs/web/Server/
./register.sh
node main.js
````

## Start Vue Frontend
````bash
cd nft_fabric_ipfs/web/NFTAppOnVue/
npm run dev
````
## Visit Web Application
````bash
http://localhost:9527
````



# Credits
🙏 This project is a fork of hyperledger fabric SDK see docs/README.md

https://hyperledger-fabric.readthedocs.io/en/release-2.2/
> base version :
```bash
curl -sSL https://bit.ly/2ysbOFE | bash -s
```
