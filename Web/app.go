package Web

import (
	"zhq/bill/Web/Controllers"
	"fmt"
	"net/http"
)

func WebStart(app *Controllers.Application)error  {
	// 指定文件服务器
	//如果不指定文件服务器，css和js将不起作用
	fs := http.FileServer(http.Dir("Web/Static"))
	http.Handle("/Static/", http.StripPrefix("/Static/", fs))

	fmt.Println("启动应用程序，监听端口号:8888")
	http.HandleFunc("/",app.LoginView)
	http.HandleFunc("/login.html",app.LoginView)
	//登录按钮响应
	http.HandleFunc("/login",app.Login)
	//发布票据页
	http.HandleFunc("/issue.html",app.Issue)
	//发布票据
	http.HandleFunc("/issue",app.SaveBill)
	//查询我的票据列表
	http.HandleFunc("/bills.html",app.QueryMyBills)
	//发起背书
	http.HandleFunc("/endorse",app.Endorse)
	//查看票据详情
	http.HandleFunc("/billinfo",app.QueryBillInfo)
	//查看所有待签收票据
	http.HandleFunc("/waitAccept.html",app.WaitAccepts)
	//查看代签收票据详情
	http.HandleFunc("/waitAcceptInfo.html",app.WaitAcceptInfo)
	//退出登录
	http.HandleFunc("/loginout",app.LoginOut)
	//签收票据
	http.HandleFunc("/accept",app.Accetp)
	//拒签票据
	http.HandleFunc("/reject",app.Reject)
	err := http.ListenAndServe(":8888",nil)
	if err != nil {
		return fmt.Errorf("启动web服务失败:%s",err.Error())
	}
	return nil
}