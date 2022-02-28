[//]: # (SPDX-License-Identifier: CC-BY-4.0)

# System Architecture
![image](https://github.com/quieoo/nft_fabric_ipfs/blob/main/architecture/architecture_nft_ipfs.drawio%20(2).png)

[Why build this app?](https://github.com/quieoo/nft_fabric_ipfs/blob/main/architecture/nft.pptx)
# Demo Start
## Clone repo locally
```bash
git clone --recurse-submodules https://github.com/quieoo/nft_fabric_ipfs.git
```
## Hyperledger Fabric and IPFS docker daemon bring up
make sure docker is running
```bash
cd nft_fabric_ipfs/test-network/
./000_bringDown.sh  
./001_bringUP.sh 
```
we modify the ipfs source code so as the "add" api can serve the request of a file name located at ipfs's mount dir, this is because we can't find where chaincode VM mounted.
Details can be found [here](https://github.com/quieoo/go-ipfs.git) at dev branch.
We rebuild the ipfs and upload it to docker hub with "quieoo/docker-ipfs"

This will clean the environment and build hyperledger fabric peers and ipfs docker daemon, deploy chaincode "FI-NFT"

## Start Web Server
````bash
cd nft_fabric_ipfs/web/Server/
npm install
./register.sh
node main.js
````
## Start Vue Frontend
````bash
npm install
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
