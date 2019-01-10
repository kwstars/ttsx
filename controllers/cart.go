package controllers

import (
	"github.com/astaxie/beego"
	"github.com/gomodule/redigo/redis"
)

type AddCartController struct {
	beego.Controller
}

func (this *AddCartController) HandleAddCart() {
	goodsId, err1 := this.GetInt("goodsId")
	goodsCount, err2 := this.GetInt("goodsCount")

	if err1 != nil || err2 != nil {
		beego.Error("ajax传递数据失败")
		return
	}

	resp := make(map[string]interface{})
	//defer this.ServeJSON()

	userName := this.GetSession("userName")
	if userName == nil {
		resp["errno"] = 1
		resp["errmsg"] = "用户未登陆"
		this.Data["json"] = resp
		this.ServeJSON()
	}

	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		resp["errno"] = 2
		resp["errmsg"] = "用户连接redis失败"
		this.Data["json"] = resp
		this.ServeJSON()
	}
	defer conn.Close()

	conn.Do("hset","cart_"+userName.(string),goodsId, goodsCount)
	resp["errno"] = 5
	resp["errmsg"] = "Ok"
	this.Data["json"] = resp
	this.ServeJSON()
}
