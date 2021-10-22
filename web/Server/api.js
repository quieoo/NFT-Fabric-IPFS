
var express = require('express')
const {Mint, ClientAccountID, Transfer,Request, TotalBids, GetBidsByIndex,Register,Login, GetAccountBalance} = require("./fabric");
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
        console.log(err)
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
        console.log(`******** FAILED to Login: ${err}`)
        res.status(404).send(err)
    }
})



router.post('/mint', upload.single('file'), async (req, res, next) => {
    // 返回客户端的信息
    console.log(req.body.clientID)
    console.log(req.body.org)
    console.log(req.body.tokenID)
    console.log(req.file)
    // 获取文件
    let file = req.file
    let newname
    if (file) {
        // 获取文件名
        let fileNameArr = req.body.tokenID+'.png'
        // 获取文件后缀名
        var suffix = fileNameArr[fileNameArr.length - 1]
        // 文件重命名
        newname=`./uploads/${fileNameArr}`
        fs.renameSync('./uploads/' + file.filename, newname)
        file['filename'] = `${fileNameArr}`
    }

    let result=await Mint(req.body.clientID, req.body.org, req.body.tokenID, newname)
    res.status(200).send('Successfully Mint NFT with id: ' + req.body.tokenID + ' for: ' + req.body.clientID + ', CID: '+result.toString())
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
    console.log(clientid)
    console.log(org)
    let result=await GetAccountBalance(clientid,org)
    console.log(result)
    res.status(200).send('user: ' + clientid + '\nbalance: '+ result.Balance +'\nencoded accountID: ' + result.Account)
})
router.post('/totalbids',async(req,res)=>{
    console.log(req.url)
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    let result=await TotalBids(clientid,org)
    res.status(200).send(result.toString())
})

router.post('/getBidsByIndex',async(req,res)=>{
    let param = US(req.url)
    let clientid=param.get('clientID')
    let org=param.get('org')
    let index=param.get('index')    //string
    console.log(req.url)
    let result=await GetBidsByIndex(clientid,org,index)
    let parsedresult=JSON.parse(result.toString())
    res.status(200).send(parsedresult)
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
    let result=await Transfer(clientid,org,tokenid,targetid)
    console.log(result)
    res.status(200).send('transfer token '+tokenid+' '+result)
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
