package routers

import (
	"ttsx/controllers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

func init() {
	// 路由过滤
	beego.InsertFilter("/goods/*", beego.BeforeExec, filterFunc)

	//beego.Router("/", &controllers.MainController{})
	beego.Router("/", &controllers.GoodsController{}, "get:ShowIndex")

	// 注册 登陆 退出
	beego.Router("/register", &controllers.UserController{}, "get:ShowRegister;post:HandleRegister")
	beego.Router("/login", &controllers.UserController{}, "get:ShowLogin;post:HandleLogin")
	beego.Router("/active", &controllers.UserController{}, "get:ActiveUser")
	beego.Router("/logout", &controllers.UserController{}, "get:UserLogout")

	// 用户中心
	beego.Router("/goods/userCenterInfo", &controllers.UserController{}, "get:ShowUserCenterInfo")
	beego.Router("/goods/userCenterOrder", &controllers.UserController{}, "get:ShowUserCenterOrder")
	beego.Router("/goods/userCenterSite", &controllers.UserController{}, "get:ShowUserCenterSite;post:HeandleShowUserCenterSite")
}

func filterFunc(ctx *context.Context) {
	userName := ctx.Input.Session("userName")
	if userName == nil {
		ctx.Redirect(302, "/login")
		return
	}
}
