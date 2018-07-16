package main

import (
	"os"
	"fmt"
	"zhq/bill/Blockchain"
	"zhq/bill/Service"
	"zhq/bill/Web/Controllers"
	"zhq/bill/Web"
)

func main()  {
	setup := Blockchain.FabricSetup{
		//组织内管理员用户
		OrgAdmin:      "Admin",
		//组织ID
		OrgName:       "Org1",
		//通道ID
		ChannelID:     "mychannel",
		//应用配置文件路径
		ConfigFile:    "config.yaml",
		//通道配置文件路径
		ChannelConfig: os.Getenv("GOPATH") + "/src/zhq/bill/fixtures/artifacts/channel.tx",
		//链码相关
		ChaincodeID:     "bill",
		ChaincodeGoPath: os.Getenv("GOPATH"),
		ChaincodePath:   "zhq/bill/ChainCode",
		ChaincodeVersion: "1.0",
		UserName:        "User1",
	}
	err := setup.Initialize()
	if err != nil {
		fmt.Println(err)
	}
	//安装实例化链码
	err = setup.InstallAndInstantiateCC()
	if err != nil {
		fmt.Printf("链码安装实例化发生错误:%s",err.Error())
	}
	serv := new(Service.FabricSetupService)
	serv.Setup = &setup
/*
	//测试数据
	bill := Service.Bill{
		BillInfoID:        "123456",
		BillInfoAmt:       "10",
		BillInfoType:      "liangpiao",
		BillInfoIsseDate:  "20180702",
		BillInfoDueDate:   "20190702",
		DrwrCmID:          "120xxxxxxx",
		DrwrAcct:          "zhq",
		AccptrCmID:        "accp120xxxxx",
		AccptrAcct:        "AccpName",
		PyeeCmID:          "Pyee120xxxxx",
		PyeeAcct:          "PyeeName",
		HoldrCmID:         "120xxxxxxx",
		HoldrAcct:         "zhq",
		WaitEndorseCmID:   "",
		WaitEndorseAcct:   "",
		RejectEndorseCmID: "",
		RejectEndorseAcct: "",
		State:             "",
		History:           nil,
	}
	response, err := serv.IssueBill(bill)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println("交易ID：", response)
	}

	b, err := serv.QueryBill("120xxxxxxx")
	if err != nil {
		fmt.Errorf(err.Error())
	} else {
		var bills = []Service.Bill{}
		json.Unmarshal(b, &bills)
		for _, temp := range bills {
			fmt.Println(temp)
		}
	}

	b, err = serv.QueryBillByNo("123456")
	if err != nil {
		fmt.Errorf(err.Error())
	} else {
		var result Service.Bill
		json.Unmarshal(b, &result)
		fmt.Println(result)
		for _, history := range result.History {

			fmt.Println(history)
		}
	}
	//发起背书
	res, err := serv.Endorse("123456", "pei120xxxxxx", "pei")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(res)
	}
	//
	b, err = serv.QueryMyWaitBills("pei120xxxxxx")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		var bills = []Service.Bill{}
		json.Unmarshal(b, &bills)
		for _, temp := range bills {
			fmt.Println(temp)
		}
	}
	//签收票据
	res, err = serv.Accept("123456", "pei120xxxxxx", "pei")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(res)
	}
	//拒签票据
	res, err = serv.Accept("123456", "pei120xxxxxx", "pei")
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(res)
	}
*/

	app := new(Controllers.Application)
	app.Fabric = serv
	Web.WebStart(app)
	if err != nil {
		fmt.Println(err.Error())
	}
}