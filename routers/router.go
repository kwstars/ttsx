package routers

import (
	"ttsx/controllers"
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
)

func init() {
	//beego.Router("/", &controllers.MainController{})

	// 路由过滤
	beego.InsertFilter("/goods/*", beego.BeforeExec, filterFunc)

	// 首页 详情页
	beego.Router("/", &controllers.GoodsController{}, "get:ShowIndex")
	beego.Router("/detail", &controllers.GoodsController{}, "get:ShowDetail")
	beego.Router("/list", &controllers.GoodsController{}, "get:ShowList")
	beego.Router("/search", &controllers.GoodsController{}, "post:HandleSearch")

	// 注册 登陆 退出
	beego.Router("/register", &controllers.UserController{}, "get:ShowRegister;post:HandleRegister")
	beego.Router("/login", &controllers.UserController{}, "get:ShowLogin;post:HandleLogin")
	beego.Router("/active", &controllers.UserController{}, "get:ActiveUser")
	beego.Router("/logout", &controllers.UserController{}, "get:UserLogout")

	// 用户中心
	beego.Router("/goods/userCenterInfo", &controllers.UserController{}, "get:ShowUserCenterInfo")
	beego.Router("/goods/userCenterOrder", &controllers.UserController{}, "get:ShowUserCenterOrder")
	beego.Router("/goods/userCenterSite", &controllers.UserController{}, "get:ShowUserCenterSite;post:HeandleShowUserCenterSite")

	// 购物车
	beego.Router("/cart", &controllers.CartController{}, "get:ShowCart;post:HandleAddCart")
	beego.Router("/updateCart", &controllers.CartController{}, "post:UpdateCart")
	beego.Router("/deleteCart", &controllers.CartController{}, "post:DeleteCart")

	// 订单
	beego.Router("/goods/order", &controllers.OrderController{}, "post:ShowOrder")
	beego.Router("/goods/addOrder", &controllers.OrderController{}, "post:AddOrder")

	// 支付宝
	beego.Router("/aliPay", &controllers.OrderController{}, "get:HandlePay")
	beego.Router("/payOk", &controllers.OrderController{}, "get:PayOK")

	// 短信
	beego.Router("/sendMsg", &controllers.OrderController{}, "get:SendMsg")
}

func filterFunc(ctx *context.Context) {
	userName := ctx.Input.Session("userName")
	if userName == nil {
		ctx.Redirect(302, "/login")
		return
	}
}
