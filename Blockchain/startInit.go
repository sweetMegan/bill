package Blockchain

import (
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/resmgmtclient"
	"fmt"
	"errors"

	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/hyperledger/fabric-sdk-go/pkg/config"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chmgmtclient"
	"github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient"
	"time"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabric-client/ccpackager/gopackager"
	"github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"
)

type FabricSetup struct {
	//应用配置文件路径
	ConfigFile string
	//通道ID
	ChannelID string
	//sdk是否已初始化过，若已初始化，不再做初始化操作
	Initialized bool
	//通道配置文件路径
	ChannelConfig string
	//组织管理员账户名
	OrgAdmin string
	//组织名
	OrgName string
	//ResourceMgmtClient 使用'github.com/hyperledger/fabric-sdk-go/api/apitxn/resmgmtclient'包下的结构体，不要搞错
	Admin resmgmtclient.ResourceMgmtClient
	sdk   *fabsdk.FabricSDK
	//链码所需参数
	ChaincodeID     string //链码名称
	ChaincodeGoPath string //系统GOPATH路径
	ChaincodePath   string //链码所在路径
	ChaincodeVersion string //链码版本
	//执行链码的用户名
	UserName string
	//chclient.ChannelClient 使用的是 "github.com/hyperledger/fabric-sdk-go/api/apitxn/chclient" 包下的结构体
	Client  chclient.ChannelClient
}

func (setup *FabricSetup) Initialize() error {
	fmt.Println("开始初始化。。。")
	if setup.Initialized {
		return errors.New("sdk已经初始化")
	}
	//使用指定的配置文件创建SDK
	sdk, err := fabsdk.New(config.FromFile(setup.ConfigFile))
	if err != nil {
		return fmt.Errorf("创建SDK失败:%s", err.Error())
	}
	setup.sdk = sdk
	//根据指定的具有特权的用户（admin）创建用于管理通道的客户端API
	chMgmtClient, err := setup.sdk.NewClient(fabsdk.WithUser(setup.OrgAdmin), fabsdk.WithOrg(setup.OrgName)).ChannelMgmt()

	//chMgmtClient, err := setup.sdk.NewClient(fabsdk.WithUser(setup.OrgAdmin), fabsdk.WithOrg(setup.OrgName)).ChannelMgmt()
	if err != nil {
		return fmt.Errorf("SDK添加管理用户失败:%s", err.Error())
	}

	//获取客户端的会话用户
	session, err := setup.sdk.NewClient(fabsdk.WithUser(setup.OrgAdmin), fabsdk.WithOrg(setup.OrgName)).Session()
	if err != nil {
		return fmt.Errorf("获取会话用户失败：%s,%s:%s\n", setup.OrgName, setup.OrgAdmin, err.Error())
	}
	orgAdminUser := session

	//指定用于创建或更新通道的参数
	req := chmgmtclient.SaveChannelRequest{
		ChannelID:setup.ChannelID,
		ChannelConfig:setup.ChannelConfig,
		SigningIdentity:orgAdminUser,
	}
	//使用指定参数创建或更新通道
	err = chMgmtClient.SaveChannel(req)
	if err != nil {
		return fmt.Errorf("创建通道失败:%s\n",err.Error())
	}
	//创建或更新通过会有延迟,主线程等5秒
	time.Sleep(time.Second * 5)
	//创建一个用于管理系统资源的饿客户端API
	setup.Admin,err = setup.sdk.NewClient(fabsdk.WithUser(setup.OrgAdmin)).ResourceMgmt()
	if err != nil {
		return fmt.Errorf("创建资源管理客户端失败:%s\n",err.Error())
	}

	//将peer加入通道
	if err = setup.Admin.JoinChannel(setup.ChannelID); err != nil {
		return fmt.Errorf("peer加入通道失败:%s\n",err.Error())
	}
	fmt.Println("初始化成功")
	setup.Initialized = true
	return nil

}
func (t *FabricSetup) InstallAndInstantiateCC()error {
	fmt.Println("开始安装链码...")
	//对链码进行打包
	ccPkg,err := gopackager.NewCCPackage(t.ChaincodePath,t.ChaincodeGoPath)
	if err != nil {
		return fmt.Errorf("创建指定的链码包失败%s",err.Error())
	}
	//指定安装链码时的所需参数
	installCCRequest := resmgmtclient.InstallCCRequest{
		Name:t.ChaincodeID,
		Path:t.ChaincodePath,
		Version:t.ChaincodeVersion,
		Package:ccPkg,
	}
	//安装链码
	_,err = t.Admin.InstallCC(installCCRequest)
	if err != nil {
		return fmt.Errorf("安装链码失败%s",err.Error())
	}
	fmt.Println("安装链码成功")
	fmt.Println("开始实例化链码...")
	//指定链码策略

	//cauthdsl.SignedByAnyMember 使用 "github.com/hyperledger/fabric-sdk-go/third_party/github.com/hyperledger/fabric/common/cauthdsl"包下的
	ccPolicy := cauthdsl.SignedByAnyMember([]string{"Org1MSP"})
	//指定实例化链码时的所需参数
	instantiateCCReq := resmgmtclient.InstantiateCCRequest{
		Name:t.ChaincodeID,
		Path:t.ChaincodePath,
		Version:t.ChaincodeVersion,
		Args:[][]byte{[]byte("init")},
		Policy:ccPolicy,
	}

	err = t.Admin.InstantiateCC(t.ChannelID,instantiateCCReq)
	if err != nil {
		return fmt.Errorf("实例化链码失败%s",err.Error())
	}
	fmt.Println("实例化链码成功")
	//创建客户端对象，能够通过该对象执行链码查询及事务执行
	t.Client,err = t.sdk.NewClient(fabsdk.WithUser(t.UserName)).Channel(t.ChannelID)
	if err != nil {
		return fmt.Errorf("创建新的通道客户端失败:%s",err.Error())
	}
	fmt.Println("链码安装实例化完成，且成功创建客户端对象")
	return nil
}
