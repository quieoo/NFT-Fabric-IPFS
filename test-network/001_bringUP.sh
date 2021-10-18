echo "== Starting network with CA, create channel =="
sleep 3
./network.sh up createChannel -ca
echo ""
echo "== start network successfully=="


#TODO: create docker network, join the ipfs_host into network and copy assigned ip to contract

echo "== Deploying ChainCode =="
sleep 3
./network.sh deployCC -ccn finft -ccp ../FI-NFT/chaincode-go/ -ccl go
#./network.sh deployCC -ccn token_erc721 -ccp ../token-erc-721/chaincode-javascript/ -ccl javascript

echo ""
echo "==deploy chaincode successfully=="

sleep 3

echo "==register, enroll clients=="
export FABRIC_CFG_PATH=${PWD}/../config/
export PATH=${PWD}/../bin:$PATH

#miner in org1
#export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/
#fabric-ca-client register --caname ca-org1 --id.name minter --id.secret minterpw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
#fabric-ca-client enroll -u https://minter:minterpw@localhost:7054 --caname ca-org1 -M "${PWD}/organizations/peerOrganizations/org1.example.com/users/minter@org1.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org1/tls-cert.pem"
#cp "${PWD}/organizations/peerOrganizations/org1.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org1.example.com/users/minter@org1.example.com/msp/config.yaml"

#recipient in org2
#export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org2.example.com/
#fabric-ca-client register --caname ca-org2 --id.name recipient --id.secret recipientpw --id.type client --tls.certfiles "${PWD}/organizations/fabric-ca/org2/tls-cert.pem"
#fabric-ca-client enroll -u https://recipient:recipientpw@localhost:8054 --caname ca-org2 -M "${PWD}/organizations/peerOrganizations/org2.example.com/users/recipient@org2.example.com/msp" --tls.certfiles "${PWD}/organizations/fabric-ca/org2/tls-cert.pem"
#cp "${PWD}/organizations/peerOrganizations/org2.example.com/msp/config.yaml" "${PWD}/organizations/peerOrganizations/org2.example.com/users/recipient@org2.example.com/msp/config.yaml"

echo ""
echo "==ready for chaincode invoke=="

sleep 3
echo ""
echo "==monitoring containers log=="
./monitordocker.sh fabric_test
