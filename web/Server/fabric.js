const { Gateway, Wallets } = require('fabric-network');
const FabricCAServices = require('fabric-ca-client');
const path = require('path');
const { buildCAClient, registerAndEnrollUser, enrollAdmin } = require('../../test-application/javascript/CAUtil.js');
const { buildCCPOrg1, buildCCPOrg2, buildWallet } = require('../../test-application/javascript/AppUtil.js');
const fs = require("fs");

const channelName = 'mychannel';
const chaincodeName = 'finft';
const mspOrg2 = 'Org2MSP';
function prettyJSONString(inputString) {
    return JSON.stringify(JSON.parse(inputString), null, 2);
}



// org1 only has one account: admin
// auction users belong to org2
async function Login(clientID){
    try{
        let ccp=buildCCPOrg1()
        let walletPath=path.join(__dirname, 'wallet/org2');
        const wallet = await buildWallet(Wallets, walletPath);

        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        let account=await contract.evaluateTransaction('ClientAccountID')
        let result=await contract.evaluateTransaction('GetAccount',account.toString())

        let jsonresult=JSON.parse(result.toString())
        gateway.disconnect()
        return jsonresult
    }
    catch(err){
        //console.error(`******** FAILED to Login: ${err}`)
        throw new Error(`******** FAILED to Login: ${err}`)
    }
}


//use org1.admin as operator to registe an account (uploadAccountBalance with 100)
async function Register(clientID){
    try{
        // registe wallet
        // all registered user belong to org2
        const ccp2 = buildCCPOrg2()
        const caClient2 = buildCAClient(FabricCAServices, ccp2, 'ca.org2.example.com');
        const walletPath2=path.join(__dirname,'wallet/org2')
        const wallet2 = await buildWallet(Wallets, walletPath2);
        await enrollAdmin(caClient2, wallet2, mspOrg2);
        await registerAndEnrollUser(caClient2, wallet2, mspOrg2, clientID, 'org2.department1');
        const gateway2 = new Gateway();
        await gateway2.connect(ccp2, {
            wallet: wallet2,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true }
        });
        const network2 = await gateway2.getNetwork(channelName)
        const contract2 = network2.getContract(chaincodeName);
        let account=await contract2.evaluateTransaction('ClientAccountID')  // get account for clientID
        gateway2.disconnect()

        // use admin account, put account~balance(100) to smart contract
        let ccp;
        let walletPath;
        ccp=buildCCPOrg1()
        walletPath=path.join(__dirname, 'wallet/org1');
        const wallet = await buildWallet(Wallets, walletPath);
        const gateway = new Gateway();
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: 'admin',
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });
        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        await contract.submitTransaction('InitAccountBalance',account.toString(),'100')
        gateway.disconnect()
    }catch(err){
        throw new Error(`******** FAILED to Registe: ${err}`)
    }
}

async function GetAccountBalance(clientID,org){
    try{
        let ccp;
        let walletPath;
        if (org==='org1'){
            ccp=buildCCPOrg1()
            walletPath=path.join(__dirname, 'wallet/org1');
        }else{
            ccp=buildCCPOrg2()
            walletPath=path.join(__dirname, 'wallet/org2');
        }
        const wallet = await buildWallet(Wallets, walletPath);

        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);
        let result = await contract.evaluateTransaction('GetAccountBalance')
        return JSON.parse(result.toString())
    }catch(err){
        throw err
    }
}

async function Mint(clientID, org, tokenID, filePath){
    try{

        let data = fs.readFileSync(filePath)
        //let fileURI = data.toString()
        let fileURI = data.toString()
        let ccp;
        let walletPath;
        if (org==='org1'){
            ccp=buildCCPOrg1()
            walletPath=path.join(__dirname, 'wallet/org1');
        }else{
            ccp=buildCCPOrg2()
            walletPath=path.join(__dirname, 'wallet/org2');
        }
        const wallet = await buildWallet(Wallets, walletPath);

        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        let result = await contract.submitTransaction('MintWithFile',tokenID,fileURI)

        gateway.disconnect()
        return result
    }catch (error) {
        console.error(`******** FAILED to mint token: ${error}`)
    }
}

async function Transfer(clientID, org, tokenID, targetID){
    try{
        let ccp;
        let walletPath;
        if (org==='org1'){
            ccp=buildCCPOrg1()
            walletPath=path.join(__dirname, 'wallet/org1');
        }else{
            ccp=buildCCPOrg2()
            walletPath=path.join(__dirname, 'wallet/org2');
        }
        const wallet = await buildWallet(Wallets, walletPath);
        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        let result = await contract.submitTransaction('Transfer',targetID,tokenID)
        gateway.disconnect()
        return result
    }catch (error) {
        console.error(`******** FAILED to transfer token: ${error}`)
    }
}

async function ClientAccountID(clientID, org){
    try{
        let ccp;
        let walletPath;
        if (org==='org1'){
            ccp=buildCCPOrg1()
            walletPath=path.join(__dirname, 'wallet/org1');
        }else{
            ccp=buildCCPOrg2()
            walletPath=path.join(__dirname, 'wallet/org2');
        }
        const wallet = await buildWallet(Wallets, walletPath);

        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        let result = await contract.evaluateTransaction('ClientAccountID')

        gateway.disconnect()
        return result.toString()
    }catch (error) {
        console.error(`******** FAILED to run the application: ${error}`)
    }
}



async function Request(clientID, org, tokenID){
    try{
        let ccp;
        let walletPath;
        if (org==='org1'){
            ccp=buildCCPOrg1()
            walletPath=path.join(__dirname, 'wallet/org1');
        }else{
            ccp=buildCCPOrg2()
            walletPath=path.join(__dirname, 'wallet/org2');
        }
        const wallet = await buildWallet(Wallets, walletPath);

        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        let result = await contract.evaluateTransaction('Request',tokenID)

        gateway.disconnect()
        return result
    }catch (error) {
        console.error(`******** FAILED to request token: ${error}`)
    }
}

async function Query(clientID,org,tokenID){
    try{
        let ccp;
        let walletPath;
        if (org==='org1'){
            ccp=buildCCPOrg1()
            walletPath=path.join(__dirname, 'wallet/org1');
        }else{
            ccp=buildCCPOrg2()
            walletPath=path.join(__dirname, 'wallet/org2');
        }
        const wallet = await buildWallet(Wallets, walletPath);

        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        let result = await contract.evaluateTransaction('Query',tokenID)
        gateway.disconnect()
        jsonresult=JSON.parse(result.toString())
        // console.log(jsonresult.Owner)
        return jsonresult
    }catch (error) {
        console.error(`******** FAILED to query token: ${error}`)
    }
}

async function TotalBids(clientID,org){
    try{
        let ccp;
        let walletPath;
        if (org==='org1'){
            ccp=buildCCPOrg1()
            walletPath=path.join(__dirname, 'wallet/org1');
        }else{
            ccp=buildCCPOrg2()
            walletPath=path.join(__dirname, 'wallet/org2');
        }
        const wallet = await buildWallet(Wallets, walletPath);

        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        let result=await contract.evaluateTransaction('TotalBids')
        let i = parseInt(result.toString())
        return i-1
    }catch (error) {
        console.error(`******** FAILED to get bid list: ${error}`)
    }
}

async function GetBidsByIndex(clientID,org, index){
    try{
        let ccp;
        let walletPath;
        if (org==='org1'){
            ccp=buildCCPOrg1()
            walletPath=path.join(__dirname, 'wallet/org1');
        }else{
            ccp=buildCCPOrg2()
            walletPath=path.join(__dirname, 'wallet/org2');
        }
        const wallet = await buildWallet(Wallets, walletPath);

        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        let result = await contract.evaluateTransaction('GetBidByIndex',index)
        return result
    }catch (error) {
        console.error(`******** FAILED to get bid by index: ${error}`)
    }
}

async function AddBid(clientID,org,tokenID, lowPrice,upPrice, time){
    try{
        let ccp;
        let walletPath;
        if (org==='org1'){
            ccp=buildCCPOrg1()
            walletPath=path.join(__dirname, 'wallet/org1');
        }else{
            ccp=buildCCPOrg2()
            walletPath=path.join(__dirname, 'wallet/org2');
        }
        const wallet = await buildWallet(Wallets, walletPath);

        const gateway = new Gateway();
        // act as user1, create asset
        await gateway.connect(ccp, {
            wallet: wallet,
            identity: clientID,
            discovery: { enabled: true, asLocalhost: true } // using asLocalhost as this gateway is using a fabric network deployed locally
        });

        const network = await gateway.getNetwork(channelName)
        const contract = network.getContract(chaincodeName);

        let result = await contract.submitTransaction('AddBid',tokenID,lowPrice,upPrice,time)
        return result
    }catch (error) {
        console.error(`******** FAILED to add bid: ${error}`)
    }
}
async function testView2(){

    let result=await Mint('recipient','org2','1','uploads/t')
    console.log(result.toString())  //expect result: NFT

    result=await AddBid('recipient','org2','1','199','998')
    console.log(result.toString())  //expect result: NFTBid

    result=await GetBidsByIndex('recipient','org2','0')
    console.log(result.toString())  //expect result: NFTBid

    result =await Request('recipient','org2','1')
    console.log(result.toString())  // expect result: file content

}
async function testRequestBids(){
    for(let i=0;i<6;i++){
        await Mint('recipient','org2',i.toString(),'uploads/'+i.toString()+'.png')
        await AddBid('recipient','org2',i.toString(),'1','10','10')
    }
    let result=await TotalBids('recipient','org2')
    console.log(result.toString())
}

async function fullflowtest(){

}

// testRequestBids()

module.exports={Mint,Request,ClientAccountID,Transfer,TotalBids,GetBidsByIndex,Register,Login,GetAccountBalance}
