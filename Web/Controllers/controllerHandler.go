package Controllers

import (
	"zhq/bill/Service"
	"net/http"
	"fmt"
	"encoding/json"
)

type Application struct {
	Fabric *Service.FabricSetupService

}
var cuser User
func (app *Application)LoginView(w http.ResponseWriter,r *http.Request)  {
	ShowView(w,r,"login.html",nil)
}
func (app *Application)Login(w http.ResponseWriter,r *http.Request){
	userName := r.FormValue("userName")
	password := r.FormValue("password")

	data := &struct {
		CurrentUser User
		Flag    bool
	}{
		Flag:false,
	}

	var flag bool
	for _, user := range Users {
		if user.UserName == userName && user.Password == password {
			cuser = user
			flag = true
			break
		}
	}

	if flag {
		// 登录成功, 根据当前用户查询票据列表
		fmt.Println("当前登录用户信息:", cuser)
		//向表单中插入数据
		//r.Form.Set("holdeCmId", cuser.CmId)
		app.QueryMyBills(w, r)
	} else {
		data.Flag = true
		data.CurrentUser.UserName = userName
		ShowView(w, r, "login.html", data)
	}
}
// 查询我的票据列表
func (app *Application) QueryMyBills(w http.ResponseWriter, r *http.Request)  {
	holdeCmId := cuser.CmId
	result, err := app.Fabric.QueryBill(holdeCmId)
	if err != nil{
		fmt.Println("查询当前用户的票据列表失败: ", err.Error())
	}

	var bills = []Service.Bill{}
	fmt.Println("当前用户Id:", holdeCmId,"bills:",bills)

	json.Unmarshal(result, &bills)
	data := &struct {
		Bills   []Service.Bill
		Cuser   User
	}{
		Bills: bills,
		Cuser: cuser,
	}
	ShowView(w, r, "bills.html", data)
}
//发布票据页
func (app *Application) Issue(w http.ResponseWriter, r *http.Request)  {
	data := &struct {
		Msg   string
		Flag  bool
		Cuser User
	}{
		Msg:   "",
		Flag:  false,
		Cuser: cuser,
	}
	ShowView(w, r, "issue.html", data)
}
//发布票据
func (app *Application) SaveBill(w http.ResponseWriter, r *http.Request)  {
	bill := Service.Bill{
		BillInfoID:       r.FormValue("BillInfoID"),
		BillInfoAmt:      r.FormValue("BillInfoAmt"),
		BillInfoType:     r.FormValue("BillInfoType"),
		BillInfoIsseDate: r.FormValue("BillInfoIsseDate"),
		BillInfoDueDate:  r.FormValue("BillInfoDueDate"),
		DrwrCmID:         r.FormValue("DrwrCmID"),
		DrwrAcct:         r.FormValue("DrwrAcct"),
		AccptrCmID:       r.FormValue("AccptrCmID"),
		AccptrAcct:       r.FormValue("AccptrAcct"),
		PyeeCmID:         r.FormValue("PyeeCmID"),
		PyeeAcct:         r.FormValue("PyeeAcct"),
		HoldrCmID:        r.FormValue("HoldrCmID"),
		HoldrAcct:        r.FormValue("HoldrAcct"),
	}



	transactionID, err := app.Fabric.IssueBill(bill)
	var msg string
	if err != nil {
		msg = "票据发布失败:" + err.Error()
	} else {
		msg = "票据发布成功:" + transactionID
	}
	data := &struct {
		Msg  string
		Flag bool
		Cuser User

	}{
		Msg:  msg,
		Flag: true,
		Cuser: cuser,

	}
	ShowView(w, r, "issue.html", data)
}
//发起背书
func (app *Application) Endorse(w http.ResponseWriter, r *http.Request)  {
	waitEndorseAcct := r.FormValue("waitEndorseAcct")
	waitEndorseCmId := r.FormValue("waitEndorseCmId")
	billNo := r.FormValue("billNo")
	result,err := app.Fabric.Endorse(billNo,waitEndorseCmId,waitEndorseAcct)
	if err != nil{
		fmt.Println(err.Error())
	}

	r.Form.Set("billInfoNo",billNo)
	r.Form.Set("flag","t")
	r.Form.Set("Msg",result)
	app.QueryBillInfo(w,r)
}
//签收票据
func (app *Application) Accetp(w http.ResponseWriter, r *http.Request)  {
	billNo := r.FormValue("billNo")
	cmid := cuser.CmId
	acct := cuser.Acct
	result,err :=app.Fabric.Accept(billNo,cmid,acct)
	if err != nil {
		fmt.Println(err.Error())
	}
	r.Form.Set("billNo",billNo)
	r.Form.Set("flag","t")
	r.Form.Set("Msg",result)
	app.WaitAcceptInfo(w,r)
}
//拒签票据
func (app *Application) Reject(w http.ResponseWriter, r *http.Request)  {
	billNo := r.FormValue("billNo")
	cmid := cuser.CmId
	acct := cuser.Acct
	result,err :=app.Fabric.Reject(billNo,cmid,acct)
	if err != nil {
		fmt.Println(err.Error())
	}
	r.Form.Set("billNo",billNo)
	r.Form.Set("flag","t")
	r.Form.Set("Msg",result)
	app.WaitAcceptInfo(w,r)
}
//查询票据详情
func (app *Application) QueryBillInfo(w http.ResponseWriter, r *http.Request)  {

	billInfoNo := r.FormValue("billNo")
	result, err := app.Fabric.QueryBillByNo(billInfoNo)
	if err != nil {
		fmt.Println(err.Error())
	}
	var bill Service.Bill

	json.Unmarshal(result, &bill)
	data := &struct {
		Cuser User
		Bill  Service.Bill
		Flag bool
		Msg string

	}{
		Bill:  bill,
		Cuser: cuser,
		Flag:false,
		Msg:"",

	}
	flag := r.FormValue("flag")
	if flag=="t" {
		data.Flag = true
		data.Msg = r.FormValue("Msg")
	}
	ShowView(w, r, "billInfo.html", data)
}
//待签收票据列表
func (app *Application) WaitAccepts(w http.ResponseWriter, r *http.Request)  {
	waitEndorseCmId := cuser.CmId
	result,err := app.Fabric.QueryMyWaitBills(waitEndorseCmId)
	if err != nil {
		fmt.Println(err.Error())
	}
	var bills []Service.Bill
	json.Unmarshal(result,&bills)
	data := &struct {
		Bills []Service.Bill
		Cuser User
	}{
		Bills:bills,
		Cuser:cuser,
	}
	ShowView(w,r,"waitAccept.html",data)
}
//待签收票据详情
func (app *Application) WaitAcceptInfo(w http.ResponseWriter, r *http.Request)  {
	billNo := r.FormValue("billNo")
	result,err := app.Fabric.QueryBillByNo(billNo)
	if err != nil {
		fmt.Println(err.Error())
	}
	var bill Service.Bill
	json.Unmarshal(result,&bill)
	data := &struct {
		Bill Service.Bill
		Cuser User
		Flag bool
		Msg string
	}{
		bill,
		cuser,
		false,
		"",
	}
	flag := r.FormValue("flag")
	if flag == "t" {
		data.Flag = true
		data.Msg = r.FormValue("Msg")
	}
	ShowView(w,r,"waitAcceptInfo.html",data)
}
//退出登录
func (app *Application) LoginOut(w http.ResponseWriter, r *http.Request)  {
	cuser = User{}
	app.LoginView(w,r)
}