package controllers

import (
	"github.com/astaxie/beego"
	"ttsx/models"
	"github.com/astaxie/beego/orm"
	"strconv"
	"github.com/gomodule/redigo/redis"
)

type OrderController struct {
	beego.Controller
}

func (this *OrderController) ShowOrder() {
	goodsIds := this.GetStrings("goodsId")
	userName := this.GetSession("userName").(string)
	//beego.Info(goodsIds)

	if len(goodsIds) == 0 {
		beego.Error("传输数据为空")
		this.Redirect("/cart", 302)
		return
	}

	o := orm.NewOrm()
	var receiver []models.Receiver
	o.QueryTable("Receiver").RelatedSel("User").Filter("User__UserName", userName).All(&receiver)
	this.Data["receiver"] = receiver


	var goods []map[string]interface{}
	totalPrice := 0
	totalCount := 0
	conn,_ := redis.Dial("tcp",":6379")
	for _, value := range goodsIds {
		id, _ := strconv.Atoi(value)
		var temp = make(map[string]interface{})
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)
		temp["goodsSku"] = goodsSku

		count, _ := redis.Int(conn.Do("hget","cart_"+userName,id))
		price := goodsSku.Price * count
		temp["price"] = price
		temp["count"] = count
		totalPrice += price
		totalCount += count
		goods = append(goods,temp)
	}

	transferPrice := 10
	payPrice := totalPrice + transferPrice

	this.Data["totalPrice"] = totalPrice
	this.Data["totalCount"] = totalCount
	this.Data["transferPrice"] = transferPrice
	this.Data["payPrice"] = payPrice

	this.Data["goods"] = goods
	this.TplName = "place_order.html"
}
