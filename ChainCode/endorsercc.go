package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"github.com/hyperledger/fabric/protos/peer"
	"encoding/json"
)

//args:billNo,WaitEndorseCmID,WaitEndorseAcct
func (t *BillChainCode) endorse(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	if len(args) != 3 {
		return shim.Error("必须指定票据号码，待背书人证件号码及待背书人名称")
	}
	bill, bl := t.getBill(stub, args[0])
	if !bl {
		return shim.Error("根据指定的票据号码查询信息时发生错误")
	}
	//检查当前待背书的票据是否为与持有人是同一个人
	if bill.HoldrCmID == args[1] {
		return shim.Error("被背书人不能是当前持票人")
	}
	//当前待背书的不能是票据流转历史中的持有人
	iterator, err := stub.GetHistoryForKey(bill.BillInfoID)
	if err != nil {
		return shim.Error("获取票据流转历史信息时发生错误")
	}
	defer iterator.Close()
	var hisBill Bill
	if iterator.HasNext() {
		hisData, err := iterator.Next()
		if err != nil {
			return shim.Error("获取历史数据时发生错误")
		}
		json.Unmarshal(hisData.Value, &hisBill)
		if hisData.Value == nil {
			var empty Bill
			hisBill = empty
		}
		if bill.HoldrCmID == args[1] {
			return shim.Error("被背书人不能是该票据的历史持有人")
		}
	}
	//更改票据状态，待背书人信息及拒绝背书人信息
	bill.State = BillInfo_State_EndorseWaitSign
	bill.WaitEndorseCmID = args[1]
	bill.WaitEndorseAcct = args[2]
	bill.RejectEndorseAcct = ""
	bill.RejectEndorseCmID = ""
	//保存票据信息
	_, bl = t.putBill(stub, bill)
	if !bl {
		return shim.Error("票据背书请求失败,保存票据信息时发生错误")
	}
	//根据待背书人的证件号码及票据号码创建复合键，以方便批量查询
	waitEndorSerCmIDBillInfoID, err := stub.CreateCompositeKey(IndexName, []string{bill.WaitEndorseCmID, bill.BillInfoID})
	if err != nil {
		return shim.Error("根据待背书人的证件号码及票据号码创建复合键失败")
	}
	stub.PutState(waitEndorSerCmIDBillInfoID, []byte{0x00})
	return shim.Success([]byte("发起背书请求成功，此票据待背书人处理"))
}
// 票据背书签收
// args: 0 - Bill_No;  1 - endorseCmId(待背书人ID); 2 - endorseAcct(待背书人名称)
func (t *BillChainCode) Accept(stub shim.ChaincodeStubInterface, args []string) peer.Response  {
	// 1. 检查参数长度是否为3(票据号码, 待背书人ID, 待背书人名称)
	if len(args) < 3 {
		res := GetRetString(1, "票据背书签收失败, 参数不能少于3个")
		return shim.Error(res)
	}

	// 2. 根据票据号码获取票据状态
	bill, bl := t.getBill(stub, args[0])
	if !bl {
		res := GetRetString(1, "票据背书签收失败, 根据票据号码查询对应票据状态时发生错误")
		return shim.Error(res)
	}

	// 3. 以前手持票人ID与票据号码构造复合键, 删除该key, 以便前手持票人无法再查到该票据
	holderNameBillNoIndexKey, err := stub.CreateCompositeKey(IndexName, []string{bill.HoldrCmID, bill.BillInfoID})
	if err != nil{
		res := GetRetString(1, "票据背书签收失败, 创建持票人ID与票据号码复合键时发生错误")
		return shim.Error(res)
	}
	stub.DelState(holderNameBillNoIndexKey)

	// 4. 更改票据信息与状态: 将当前持票人更改为待背书人(证件与名称), 票据状态更改为背书签收, 重置待背书人
	bill.HoldrCmID = args[1]
	bill.HoldrAcct = args[2]
	bill.State = BillInfo_State_EndorseSigned
	bill.WaitEndorseCmID = ""
	bill.WaitEndorseAcct = ""

	// 5. 保存票据
	_, bl = t.putBill(stub, bill)
	if !bl {
		res := GetRetString(1, "票据背书签收失败, 保存票据状态时发生错误")
		return shim.Error(res)
	}

	// 6. 返回
	res := GetRetByte(0, "票据背书签收成功")
	return shim.Success(res)
}

// 票据背书拒签(拒绝背书)
// args: 0 - bill_NO;   1 - endorseCmId(待背书人ID);    2 - endorseAcct(待背书人名称)
func (t *BillChainCode) Reject(stub shim.ChaincodeStubInterface, args []string) peer.Response {
	// 1. 检查参数长度是否为3(票据号码, 待背书人ID, 待背书人名称)
	if len(args) < 3 {
		res := GetRetString(1, "票据背书拒签失败, 参数不能少于3个")
		return shim.Error(res)
	}

	// 2. 根据票据号码查询对应的票据状态
	bill, bl := t.getBill(stub, args[0])
	if !bl {
		res := GetRetString(1, "票据背书拒签失败, 根据票据号码查询对应的票据状态时发生错误")
		return shim.Error(res)
	}

	// 3. 以待背书人ID及票据号码构造复合键, 从search中删除该key, 以便当前被背书人无法再次查询到该票据
	holderNameBillNoIndexKey, err := stub.CreateCompositeKey(IndexName, []string{args[1], bill.BillInfoID})
	if err != nil {
		res := GetRetString(1, "票据背书拒签失败, 以待背书人ID及票据号码构造复合键时发生错误")
		return shim.Error(res)
	}
	stub.DelState(holderNameBillNoIndexKey)

	// 4. 更改票据信息与状态: 将拒绝背书人更改为当前待背书人(证件号码与名称), 票据状态更改为背书拒绝, 重置待背书人
	bill.RejectEndorseCmID = args[1]
	bill.RejectEndorseAcct = args[2]
	bill.State = BillInfo_State_EndorseReject
	bill.WaitEndorseCmID = ""
	bill.WaitEndorseAcct = ""

	// 5. 保存票据状态
	_, bl = t.putBill(stub, bill)
	if !bl {
		res := GetRetString(1, "票据背书拒签失败, 保存票据状态时发生错误")
		return shim.Error(res)
	}

	// 6. 返回
	res := GetRetByte(0, "票据背书拒签成功")
	return shim.Success(res)
}

