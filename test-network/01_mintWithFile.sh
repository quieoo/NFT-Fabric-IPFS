export FABRIC_CFG_PATH=${PWD}/../config/
export PATH=${PWD}/../bin:$PATH

echo "===MINT==="
export FABRIC_CA_CLIENT_HOME=${PWD}/organizations/peerOrganizations/org1.example.com/

export CORE_PEER_TLS_ENABLED=true
export CORE_PEER_LOCALMSPID="Org1MSP"
export CORE_PEER_MSPCONFIGPATH=${PWD}/organizations/peerOrganizations/org1.example.com/users/minter@org1.example.com/msp
export CORE_PEER_TLS_ROOTCERT_FILE=${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt
export CORE_PEER_ADDRESS=localhost:7051
export TARGET_TLS_OPTIONS=(-o localhost:7050 --ordererTLSHostnameOverride orderer.example.com --tls --cafile "${PWD}/organizations/ordererOrganizations/example.com/orderers/orderer.example.com/msp/tlscacerts/tlsca.example.com-cert.pem" --peerAddresses localhost:7051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org1.example.com/peers/peer0.org1.example.com/tls/ca.crt" --peerAddresses localhost:9051 --tlsRootCertFiles "${PWD}/organizations/peerOrganizations/org2.example.com/peers/peer0.org2.example.com/tls/ca.crt")


#peer chaincode query -C mychannel -n erc1155 -c '{"function":"ClientAccountID","Args":[]}'

#peer chaincode invoke "${TARGET_TLS_OPTIONS[@]}" -C mychannel -n erc1155 -c "{\"function\":\"MintBatch\",\"Args\":[\"$P1\",\"[1,2,3,4,5,6]\",\"[100,200,300,150,100,100]\"]}" --waitForEvent
peer chaincode invoke "${TARGET_TLS_OPTIONS[@]}" -C mychannel -n erc1155 -c "{\"function\":\"MintWithFile\",\"Args\":[\"1\",\"myfile\"]}" --waitForEvent

