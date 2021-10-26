
var express = require('express')
const {Mint, ClientAccountID, Transfer,Request, TotalBids, GetBidsByIndex,Register,Login, GetAccountBalance, TotalNFTs,
    GetNFTByIndex, IsOnSale, AddBid, Offer, IsNFTExist
} = require("./fabric");
var router = express.Router()

const multer = require('multer')
const fs = require('fs')
// 设置上传文件存储地址
const upload = multer({ dest: 'uploads/' })
router.use(express.static('uploads'))


function US (str) {
    var map = new Map()
    var num = str.indexOf('?')
    str = str.substr(num + 1)
    var arr = str.split('&')
    for (var i = 0; i < arr.length; i++) {
        num = arr[i].indexOf('=')
        if (num > 0) {
            map.set(arr[i].substring(0, num), arr[i].substr(num + 1))
        }
    }
    return map
}

router.post('/register',async(req,res)=>{
    let param = US(req.url)
    let clientID = param.get('clientID')
    try{
        let result = await Register(clientID)
        res.status(200).send(result)
    }
    catch(err){
        res.status(404).send(err)
    }
})

router.post('/login',async(req,res)=>{
    let param = US(req.url)
    let clientid=param.get('clientID')
    try{
        let result=await Login(clientid)
        res.status(200).send('Login Successfully')
    }
    catch(err){
        res.status(404).send(err)
    }
})



router.post('/mint', upload.single('file'), async (req, res, next) => {
    // 返回客户端的信息
    console.log(req.body.clientID)
    console.log(req.body.org)
    console.log(req.body.tokenID)
    console.log(req.file)
    const clientID=req.body.clientID
    const org=req.body.org
    const tokenID=req.body.tokenID
    // 获取文件
    try{
            const exist = await IsNFTExist(clientID,org,tokenID)
            if( exist.toString()==='true'){
                res.status(200).send('tokenID: '+ tokenID + ' already exists')
            }else if (exist.toString()==='false'){
                let file = req.file
                let newname
                if (file) {
                    // 获取文件名
                    let fileNameArr = tokenID+'.png'
                    // 获取文件后缀名
                    var suffix = fileNameArr[fileNameArr.length - 1]
                    // 文件重命名
                    newname=`./uploads/${fileNameArr}`
                    fs.renameSync('./uploads/' + file.filename, newname)
                    file['filename'] = `${fileNameArr}`
                }

                let result=await Mint(clientID, org, tokenID, newname)
                const jrsult=JSON.parse(result.toString())
                res.status(200).send('成功铸造货币！\n' + '数字资产被保存到IPFS上，可使用唯一标识符: ' + jrsult.CID + ' 访问')
            }else{
                res.status(404)
            }
    }catch(err){
        res.status(404).send(err)
    }
})


router.get('/uploads/*', (req, res) => {
    console.log(res.url)
    // eslint-disable-next-line no-path-concat
    res.sendFile(__dirname + '/' + req.url)
    console.log('Request for ' + req.url + ' received.')
})


router.post('/account',async (req,res)=>{
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    console.log(req.url)
    try{
        let result=await GetAccountBalance(clientid,org)
        res.status(200).send('用户名: ' + clientid + '\n余额: '+ result.Balance +'\n身份令牌: ' + result.Account)
    }catch(err){
        res.status(404).send(err)
    }

})
router.post('/totalnfts',async(req,res)=>{
    console.log(req.url)
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    try{
        let result=await TotalNFTs(clientid,org)
        res.status(200).send(result.toString())
    }
    catch (err) {
        console.log(err)
        res.status(404)
    }
})
router.post('/totalbids',async(req,res)=>{
    console.log(req.url)
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    try{
        let result=await TotalBids(clientid,org)
        console.log(result)
        res.status(200).send(result.toString())
    }
    catch (err) {
        console.log(err)
        res.status(404)
    }
})

router.post('/getBidsByIndex',async(req,res)=>{
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    let index=param.get('index')    //string
    console.log(req.url)
    try{
        let result=await GetBidsByIndex(clientid,org,index)
        let parsedresult=JSON.parse(result.toString())
        //console.log(parsedresult)
        res.status(200).send(parsedresult)
    }
    catch(err){
        res.status(404)
    }

})

router.post('/getNFTByIndex',async(req,res)=>{
    console.log(req.url)
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    let index=param.get('index')    //string
    try{
        let result=await GetNFTByIndex(clientid,org,index)
        let parsedresult=JSON.parse(result.toString())

        let onSale = await IsOnSale(clientid,org,parsedresult.ID)
        let jsonResult= {'tokenID':parsedresult.ID, 'CID':parsedresult.CID, 'Status':onSale.toString()}
        res.status(200).send(jsonResult)
    }
    catch(err){
        res.status(404)
    }

})
router.post('/addbid',async(req,res)=>{
    console.log(req.url)
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    let tid=param.get('tokenID')
    let lp=param.get('lowerPrice')
    let hp=param.get('higherPrice')
    let lt=param.get('lifeTime')
    try{
        let result=await AddBid(clientid,org,tid,lp,hp,lt)
        let parsedresult=JSON.parse(result.toString())
        res.status(200).send(parsedresult)
    }
    catch(err){
        res.status(404).send(err)
    }
})

router.post('/offer',async(req,res)=>{
    console.log(req.url)
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    let tid=param.get('tokenID')
    let price=param.get('price')
    try{
        await Offer(clientid,org,tid,price)
        res.status(200).send('出价成功')
    }
    catch(err){
        res.status(404).send(err)
    }
})


router.post('/transfer',async (req,res)=>{
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    let tokenid=param.get('tokenID')
    let targetid=param.get('targetID')
    console.log(clientid)
    console.log(org)
    console.log(tokenid)
    console.log(targetid)
    try{
        let result=await Transfer(clientid,org,tokenid,targetid)
        console.log(result)
        res.status(200).send('transfer token '+tokenid+' '+result)
    }catch(err){
        res.status(404).send(err)
    }
})

router.post('/user', function (req, res) {
    console.log(req.params)
    res.send('successfully')
    // eslint-disable-next-line node/no-deprecated-api
})


/*
router.post('/mint',async function (req, res) {
    console.log(req.url)
    let param = US(req.url)
    let clientID=param.get('clientID')
    let tokenID=param.get('tokenID')
    let org=param.get('org')
    let fileURI=param.get('fileURI')
    console.log(clientID+" "+tokenID+" "+org+" "+fileURI)
    await Mint(clientID,org,tokenID,fileURI)

    res.status(200).send('Successfully Mint NFT with id: '+ tokenID + ' for: '+clientID)
})
*/


module.exports = router
