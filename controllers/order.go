package controllers

import (
	"github.com/astaxie/beego"
	"ttsx/models"
	"github.com/astaxie/beego/orm"
	"strconv"
	"github.com/gomodule/redigo/redis"
	"time"
	"strings"
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
	conn, _ := redis.Dial("tcp", ":6379")
	for _, value := range goodsIds {
		id, _ := strconv.Atoi(value)
		var temp = make(map[string]interface{})
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)
		temp["goodsSku"] = goodsSku

		count, err := redis.Int(conn.Do("hget", "cart_"+userName, id))
		if err != nil {
			beego.Error(err)
			return
		}
		price := goodsSku.Price * count
		temp["price"] = price
		temp["count"] = count
		totalPrice += price
		totalCount += count
		goods = append(goods, temp)
	}

	transferPrice := 10
	payPrice := totalPrice + transferPrice

	// order提交需要
	this.Data["goodsIds"] = goodsIds
	this.Data["receiver"] = receiver

	// 显示页面
	this.Data["totalPrice"] = totalPrice
	this.Data["totalCount"] = totalCount
	this.Data["transferPrice"] = transferPrice
	this.Data["payPrice"] = payPrice

	this.Data["goods"] = goods
	this.TplName = "place_order.html"
}

func (this *OrderController) AddOrder() {
	addrId, err1 := this.GetInt("addrId")
	payId, err2 := this.GetInt("payId")
	goodsIds := this.GetString("goodsIds")
	totalCount, err3 := this.GetInt("totalCount")
	totalPrice, err4 := this.GetInt("totalPrice")
	transPrice, err5 := this.GetInt("transPrice")
	totalPay, err6 := this.GetInt("totalPay")

	resp := make(map[string]interface{})
	defer RespFun(&this.Controller, resp)
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || err5 != nil || err6 != nil {
		beego.Error(err1, err2, err3, err4, err5, err6)
		resp["errno"] = 1
		resp["errmsg"] = "传输数据不完整"
		return
	}
	beego.Info("addrId", addrId, "payId", payId, "goodsIds", goodsIds, "totalCount", totalCount, "totalPrice\n",
		totalPrice, "transPrice", transPrice, "totalPay", totalPay)

	o := orm.NewOrm()
	// 1.获取地址
	var orderInfo models.OrderInfo
	var receiver models.Receiver
	receiver.Id = addrId
	o.Read(&receiver)

	// 2.获取用户
	userName := this.GetSession("userName")
	if userName == nil {
		resp["errno"] = 2
		resp["errmsg"] = "用户没有登陆"
		return
	}

	var user models.User
	user.UserName = userName.(string)
	o.Read(&user, "UserName")

	// 3.支付编号 用户 地址  付款方式 商品数量 商品总价 运费
	orderInfo.OrderId = time.Now().Format("20061002150405") + strconv.Itoa(user.Id)
	orderInfo.User = &user
	orderInfo.Receiver = &receiver
	orderInfo.PayMethod = payId
	orderInfo.TotalCount = totalCount
	orderInfo.TotalPrice = totalPrice
	orderInfo.TransitPrice = transPrice

	// 4.插入用户信息表
	o.Insert(&orderInfo)

	var orderGoods models.OrderGoods
	beego.Info(goodsIds)
	// 1.处理goodsIds的字符串 [1 7 23]
	ids := strings.Split(goodsIds[1:len(goodsIds)-1], " ")

	// 2.连接redis
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		beego.Error(err)
		resp["errno"] = 2
		resp["errmsg"] = "redis connect failed"
		return
	}

	for _, v := range ids {
		// 3.循环获取商品
		id, err := strconv.Atoi(v)
		if err != nil {
			beego.Error(err)
			return
		}
		var goodsSku models.GoodsSKU
		goodsSku.Id = id
		o.Read(&goodsSku)

		count, err := redis.Int(conn.Do("hget", "cart_"+userName.(string), id))
		if err != nil {
			beego.Error(err)
			resp["errno"] = 3
			resp["errmsg"] = "Failed to obtain redis data"
			return
		}

		// 4.订单 商品 商品数量 商品价格
		orderGoods.OrderInfo = &orderInfo
		orderGoods.GoodsSKU = &goodsSku
		orderGoods.Count = count
		orderGoods.Price = count * goodsSku.Price

		// 5. 插入商品信息表
		o.Insert(&orderGoods)
	}

	resp["errno"] = 5
	resp["errmsg"] = "ok"
}
