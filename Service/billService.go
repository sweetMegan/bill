package Service


import (
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
)

func (t *FabricSetupService) IssueBill(bill Bill) (string, error) {
	//序列化票据
	b, err := json.Marshal(bill)
	if err != nil {
		return "", fmt.Errorf("指定的票据对象序列化错误:%s",err.Error())
	}
	//指定调用链码时的请求参数
	req := chclient.Request{
		ChaincodeID: t.Setup.ChaincodeID,
		Fcn:         "issue",
		Args:        [][]byte{b},
	}
	//使用实例化链码时创建的客户端对象执行链码
	//发布票据会产生交易，会被记录到区块上，所以使用"Execute"方法
	response,err := t.Setup.Client.Execute(req)
	if err != nil {
		return "",fmt.Errorf("保存票据信息失败：%s",err.Error())
	}
	//返回交易ID和错误信息
	return response.TransactionID.ID,nil
}
func (t *FabricSetupService)QueryBill(holderCmId string)([]byte,error)  {
	var args []string
	args = append(args,"queryMyBills")
	args = append(args,holderCmId)
	req := chclient.Request{
		ChaincodeID:t.Setup.ChaincodeID,
		Fcn:args[0],
		Args:[][]byte{[]byte(holderCmId)},
	}
	response,err := t.Setup.Client.Query(req)
	if err != nil {
		return nil,fmt.Errorf("根据持票人证件号查询票据失败：%s",err.Error())
	}
	b := response.Payload
	return b[:],nil
}
func (t *FabricSetupService)QueryBillByNo(billNo string)([]byte,error)  {
	var args []string
	args = append(args,"queryBillByNo")
	args = append(args,billNo)
	req := chclient.Request{
		ChaincodeID:t.Setup.ChaincodeID,
		Fcn:args[0],
		Args:[][]byte{[]byte(billNo)},
	}
	response,err := t.Setup.Client.Query(req)
	if err != nil {
		return nil,fmt.Errorf("根据票据号查询票据失败：%s",err.Error())
	}
	b := response.Payload
	return b[:],nil
}
//根据待背书人的证件号码查询待背书票据
func (t *FabricSetupService)QueryMyWaitBills(waitEndorseCmID string)([]byte,error)  {
	var args []string
	args = append(args,"queryMyWaitBills")
	args = append(args,waitEndorseCmID)
	req := chclient.Request{
		ChaincodeID:t.Setup.ChaincodeID,
		Fcn:args[0],
		Args:[][]byte{[]byte(args[1])},
	}
	response,err := t.Setup.Client.Query(req)
	if err != nil {
		return nil,fmt.Errorf(err.Error())
	}
	return response.Payload,nil
}
//billNo,waitEndorseCmID,waitEndorseAcct
func (t *FabricSetupService)Endorse(billNo string,waitEndorseCmID string,waitEndorseAcct string)(string,error)  {
	var args []string
	args = append(args,"endorse")
	args = append(args,billNo)
	args = append(args,waitEndorseCmID)
	args = append(args,waitEndorseAcct)
	req := chclient.Request{
		ChaincodeID:t.Setup.ChaincodeID,
		Fcn:args[0],
		Args:[][]byte{[]byte(args[1]),[]byte(args[2]),[]byte(args[3])},
	}
	response,err := t.Setup.Client.Execute(req)
	if err != nil {
		return "",fmt.Errorf("背书失败:%s",err.Error())
	}
	return "发起背书成功"+string(response.Payload),nil
}
func (t *FabricSetupService)Accept(billNo string,waitEndorseCmID string,waitEndorseAcct string)(string,error) {
	var args []string
	args = append(args,"accept")
	args = append(args,billNo)
	args = append(args,waitEndorseCmID)
	args = append(args,waitEndorseAcct)
	req := chclient.Request{
		ChaincodeID:t.Setup.ChaincodeID,
		Fcn:args[0],
		Args:[][]byte{[]byte(args[1]),[]byte(args[2]),[]byte(args[3])},
	}
	response,err := t.Setup.Client.Execute(req)
	if err != nil {
		return "签收失败",fmt.Errorf("签收失败:%s",err.Error())
	}
	return "签收成功"+string(response.Payload),nil
}
func (t *FabricSetupService)Reject(billNo string,waitEndorseCmID string,waitEndorseAcct string)(string,error) {
	var args []string
	args = append(args,"reject")
	args = append(args,billNo)
	args = append(args,waitEndorseCmID)
	args = append(args,waitEndorseAcct)
	req := chclient.Request{
		ChaincodeID:t.Setup.ChaincodeID,
		Fcn:args[0],
		Args:[][]byte{[]byte(args[1]),[]byte(args[2]),[]byte(args[3])},
	}
	response,err := t.Setup.Client.Execute(req)
	if err != nil {
		return "",fmt.Errorf("拒签失败:%s",err.Error())
	}
	return string(response.Payload),nil
}