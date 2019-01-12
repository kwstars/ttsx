package controllers

import (
	"github.com/astaxie/beego"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"ttsx/models"
	"github.com/astaxie/beego/orm"
)

type CartController struct {
	beego.Controller
}

func (this *CartController) HandleAddCart() {
	goodsId, err1 := this.GetInt("goodsId")
	goodsCount, err2 := this.GetInt("goodsCount")

	if err1 != nil || err2 != nil {
		beego.Error("ajax传递数据失败")
		return
	}

	resp := make(map[string]interface{})
	defer this.ServeJSON()

	userName := this.GetSession("userName")
	if userName == nil {
		resp["errno"] = 1
		resp["errmsg"] = "用户未登陆"
		this.Data["json"] = resp
	}

	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		resp["errno"] = 2
		resp["errmsg"] = "用户连接redis失败"
		this.Data["json"] = resp
	}
	defer conn.Close()

	conn.Do("hset", "cart_"+userName.(string), goodsId, goodsCount)
	resp["errno"] = 5
	resp["errmsg"] = "Ok"
	this.Data["json"] = resp
}

func (this *CartController) ShowCart() {
	userName := this.GetSession("userName")
	var cartMap map[string]int
	if userName == nil {
		this.Data["userName"] = ""
		this.Redirect("/login", 302)
	} else {
		this.Data["userName"] = userName.(string)
		conn, err := redis.Dial("tcp", ":6379")
		if err != nil {
			beego.Error("redis连接失败", err)
			this.Redirect("/", 302)
			return
		}
		defer conn.Close()

		cartMap, err = redis.IntMap(conn.Do("hgetall", "cart_"+userName.(string)))
		if err != nil {
			beego.Error("获取数据失败", err)
			this.Redirect("/", 302)
			return
		}
	}

	o := orm.NewOrm()
	var goods []map[string]interface{}
	var totalPrice, totalGoodsCount, totalGoodsTypeCount int
	for goodsId, value := range cartMap {
		temp := make(map[string]interface{})
		id, _ := strconv.Atoi(goodsId)
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)
		temp["goodsSku"] = goodsSku
		temp["count"] = value

		price := goodsSku.Price * value
		temp["price"] = price
		totalPrice += price
		totalGoodsTypeCount += 1
		totalGoodsCount += value

		goods = append(goods, temp)
	}

	this.Data["totalPrice"] = totalPrice
	this.Data["totalGoodsCount"] = totalGoodsCount
	this.Data["totalGoodsTypeCount"] = totalGoodsTypeCount
	this.Data["goods"] = goods

	this.Layout = "layout_user.html"
	this.TplName = "cart.html"
}

func (this *CartController) UpdateCart() {
	resp := make(map[string]interface{})
	userName := this.GetSession("userName")
	if userName == "" {
		resp["errno"] = 9
		resp["errmsg"] = "请先登陆"
		this.Data["json"] = resp
		this.ServeJSON()
		this.Redirect("/login", 302)
	}
	goodsId, err1 := this.GetInt("goodsId")
	count, err2 := this.GetInt("count")
	defer this.ServeJSON()

	if err1 != nil || err2 != nil {
		beego.Info("err1", err1, "err2", err2)
		resp["errno"] = 1
		resp["errmsg"] = "传输数据格式错误"
		this.Data["json"] = resp
		return
	}

	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		beego.Error(err)
		resp["errno"] = 2
		resp["errmsg"] = "redis connent err"
		this.Data["json"] = resp
		this.ServeJSON()
		return
	}
	defer conn.Close()

	_, err = conn.Do("hset", "cart_"+userName.(string), goodsId, count)
	if err != nil {
		beego.Error(err)
		resp["errno"] = 3
		resp["errmsg"] = "redis写入数据失败"
		this.Data["json"] = resp
		return
	} else {
		resp["errno"] = 5
		resp["errmsg"] = "redis写入成功"
		this.Data["json"] = resp
	}
}
