package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"fmt"
	"github.com/hyperledger/fabric/protos/peer"
)

type BillChainCode struct {

}
func (t *BillChainCode) Init(stub shim.ChaincodeStubInterface) peer.Response {
	return shim.Success(nil)
}

func (t *BillChainCode) Invoke(stub shim.ChaincodeStubInterface) peer.Response {
	function, args := stub.GetFunctionAndParameters()
	//票据操作的七个方法
	if function == "issue" {
		//发布票据
		return t.Issue(stub, args)
	}else if function == "queryMyBills" {
		//查看我的票据列表
		return t.QueryMyBills(stub, args)
	}else if function == "queryBillByNo" {
		//票据号查询票据
		return t.QueryBillByNo(stub, args)
	}else  if function == "queryMyWaitBills" {
		//查询我的待背书票据列表
		return t.QueryMyWaitBills(stub, args)
	}else if function == "endorse" {
		//发起背书
		return t.endorse(stub, args)
	}else if function == "accept" {
		//签名
		return t.Accept(stub, args)
	}else if function == "reject" {
		//拒签
		return t.Reject(stub, args)
	}

	return shim.Error("指定的函数名称错误")
}
func main()  {
	chaincode := new(BillChainCode)
	err := shim.Start(chaincode)
	if err != nil {
		fmt.Println("启动链码错误: ", err)
	}
}