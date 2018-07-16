package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"encoding/json"
)

//根据订单号查询订单是否存在
func (t *BillChainCode) getBill(stub shim.ChaincodeStubInterface, billNo string) (Bill, bool) {
	var bill Bill
	b, err := stub.GetState(billNo)
	if err != nil {
		return bill, false
	}
	//判断查询到的结果是否为空
	err = json.Unmarshal(b, &bill)
	if err != nil {
		return bill, false
	}
	return bill, true
}

//保存票据
func (t *BillChainCode) putBill(stub shim.ChaincodeStubInterface, bill Bill) ([]byte, bool) {
	b, err := json.Marshal(bill)
	if err != nil {
		return nil, false
	}
	err = stub.PutState(bill.BillInfoID, b)
	if err != nil {
		return nil, false
	}
	return b, true

}

//args 为票据的json串
func (t *BillChainCode) Issue(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	//1、检查请求参数是否合法
	if len(args) != 1 {
		return shim.Error("发布票据失败，指定的票据内容错误")
	}

	////测试数据
	//bill := Bill{
	//	BillInfoID:        "123456",
	//	BillInfoAmt:       "10",
	//	BillInfoType:      "liangpiao",
	//	BillInfoIsseDate:  "20180702",
	//	BillInfoDueDate:   "20190702",
	//	DrwrCmID:          "120xxxxxxx",
	//	DrwrAcct:          "zhq",
	//	AccptrCmID:        "accp120xxxxx",
	//	AccptrAcct:        "AccpName",
	//	PyeeCmID:          "Pyee120xxxxx",
	//	PyeeAcct:          "PyeeName",
	//	HoldrCmID:         "120xxxxxxx",
	//	HoldrAcct:         "zhq",
	//	WaitEndorseCmID:   "pei120xxxxxx",
	//	WaitEndorseAcct:   "pei",
	//	RejectEndorseCmID: "",
	//	RejectEndorseAcct: "",
	//	State:             "",
	//	History:           nil,
	//}

	var bill Bill
	err := json.Unmarshal([]byte(args[0]), &bill)
	if err != nil {
		return shim.Error("反序列化票据对象时发生错误")
	}
	//2、查重
	//票据具有唯一性，不允许重复发布票据
	//根据票据号查询票据，如果票据已存在，不允许重复发布
	_, bl := t.getBill(stub, bill.BillInfoID)
	if bl {
		return shim.Error("发布的票据已存在")
	}
	//3、将票据状态保存为发布状态
	bill.State = BillInfo_State_NewPublish
	//4、将票据保存至账本
	_, bl = t.putBill(stub, bill)
	if !bl {
		return shim.Error("保存票据信息时发生错误")
	}
	//5、根据当前持票人ID与票据号码，定义复合key，方便后期批量查询
	holderCmIDBillIInfoIDIndexKey, err := stub.CreateCompositeKey(IndexName, []string{bill.HoldrCmID, bill.BillInfoID})
	if err != nil {
		return shim.Error("创建复合键时发生错误")
	}
	err = stub.PutState(holderCmIDBillIInfoIDIndexKey, []byte{0x00})
	if err != nil {
		return shim.Error("保存复合键时发生错误")
	}
	return shim.Success([]byte("指定的票据发布成功"))

}

//根据票据持有人ID，查询这个人的所有票据
func (t *BillChainCode) QueryMyBills(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("查询票据错误,非法的持票人号码")
	}
	iterator, err := stub.GetStateByPartialCompositeKey(IndexName, args)
	if err != nil {
		return shim.Error("根据指定的持票人证件号码查询信息时发生错误")
	}
	defer iterator.Close()
	//迭代处理
	var bills []Bill
	for iterator.HasNext() {
		kv, _ := iterator.Next()
		//kv
		//k:bill.HoldrCmID+bill.BillInfoID
		//v:[]byte{0x00}
		//所以要找到订单号要拆分k就够了
		//分割查询到的复合键
		_, compositeKey, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			return shim.Error("分割指定复合键时发生错误")
		}
		//从复合键中获取到的票据号码
		bill, bl := t.getBill(stub, compositeKey[1])
		if !bl {
			return shim.Error("根据指定的票据号码查询票据信息时发生错误")
		}
		//将查询到的订单，添加到查询结果数组中
		bills = append(bills, bill)
	}
	bs, err := json.Marshal(bills)
	if err != nil {
		return shim.Error("序列化票据时发生错误")
	}
	return shim.Success(bs)
}
func (t *BillChainCode) QueryBillByNo(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("必须且只能指定要查询的票据号码")
	}
	bill, bl := t.getBill(stub, args[0])
	if !bl {
		return shim.Error("根据指定的票据号码查询对应信息时失败")
	}
	iterator, err := stub.GetHistoryForKey(bill.BillInfoID)
	if err != nil {
		return shim.Error("根据指定的票据查询历史流转信息时失败")
	}
	defer iterator.Close()
	var bills []HistoryItem
	var historyBill Bill
	for iterator.HasNext() {
		hisData, err := iterator.Next()
		if err != nil {
			return shim.Error("获取历史流转信息时发生错误")
		}
		var historyItem HistoryItem
		historyItem.TxId = hisData.TxId
		json.Unmarshal(hisData.Value, &historyBill)
		if hisData.Value == nil {
			var empty Bill
			historyItem.Bill = empty
		} else {
			historyItem.Bill = historyBill
		}
		bills = append(bills, historyItem)
	}
	bill.History = bills
	b, err := json.Marshal(bill)
	if err != nil {
		return shim.Error("序列化票据室发生错误")
	}
	return shim.Success(b)
}
func (t *BillChainCode) QueryMyWaitBills(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 1 {
		return shim.Error("必须且只能指定待背书人证件号码")
	}
	iterator, err := stub.GetStateByPartialCompositeKey(IndexName, args)
	if err != nil {
		return shim.Error("根据指定待背书人证件号码查询复合键时发生错误")
	}

	defer iterator.Close()
	var bills []Bill
	for iterator.HasNext() {
		kv, _ := iterator.Next()
		//对复合key进行分割
		_, composite, err := stub.SplitCompositeKey(kv.Key)
		if err != nil {
			return shim.Error("分割复合key时发生错误")
		}
		bill, bl := t.getBill(stub, composite[1])
		if !bl {
			return shim.Error("根据指定的票据号码查询票据信息时发生错误")
		}
		if bill.State == BillInfo_State_EndorseWaitSign && bill.WaitEndorseCmID == args[0] {
			bills = append(bills, bill)
		}
	}
	b, err := json.Marshal(bills)
	if err != nil {
		return shim.Error("序列表待背书人票据时发生错误")
	}
	return shim.Success(b)
}
