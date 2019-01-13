package controllers

import (
	"github.com/astaxie/beego"
	"ttsx/models"
	"github.com/astaxie/beego/orm"
	"strconv"
	"github.com/gomodule/redigo/redis"
	"time"
	"strings"
	"github.com/smartwalle/alipay"
	"fmt"
	"github.com/KenmyZhang/aliyun-communicate"
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
	orderInfo.OrderId = time.Now().Format("20060102150405") + strconv.Itoa(user.Id)
	orderInfo.User = &user
	orderInfo.Receiver = &receiver
	orderInfo.PayMethod = payId
	orderInfo.TotalCount = totalCount
	orderInfo.TotalPrice = totalPrice
	orderInfo.TransitPrice = transPrice

	// 4.插入用户信息表
	o.Insert(&orderInfo)

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
		beego.Info(v)
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
		var orderGoods models.OrderGoods
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

func (this *OrderController) HandlePay() {
	var privateKey = `MIIEpQIBAAKCAQEAukKW0jfG3C0ZL0pDdTtaWi2yhf6yP41QN3kL8HktxdnKsLws
					4PvRh1ddqD4mQEFQDxPjYj4kRbPQkhLjcQJzohNoLOUhpstXH+h0q7jhWbgwWpQL
					nwV1ktYvZ+wrA7pHmkkrvq1wpKWJRHxlG50xZZ2g2M7octL6IKpYRAx/IY3Dxn2M
					vk0PArBw3/7aDEmuSIAhBqmE/+Me6kFIcOTl8B1yF1ey+roPJyVABQmRCKAbP/rh
					h6RRUesK6lax8+G+1RjqULHZL1A+1qgG2CN0VqHxEqQ86K0AnBUZLi75Ex++BWQw
					1v9AfB7py3DmbBOFUmtm3ZLXXzzR+YupkhKdWQIDAQABAoIBADzc23mvvixeFDeu
					taJOFbUX75j3Y/l+TLMDu9IFVt6qzx+3LZcK0im+c50xScB/VxDGN+v3UFTyb/n7
					cBSSb4SLgOQCr19YXIzRoaYnUIPHuw0uCSoaV5P2pyD3PAsIyLLyq/evpvo2GUem
					ukcus2B4BII0AiLbK96Wqyb5SmWE5TU70EaLXBfP5728gE0s4oVKk7kMl1ZBqIDE
					KAyfoLbo1WMgZrt1bjHa0NTalvYh8ZXdZDCU6KODLDz/bp3c5JfPKjw4NuLWObFP
					4tpOQvNvGn+ciETAW+7YchefZcaPsImFokRxaGxSnAqvbakVqrFwV4twzuXM2qED
					9hRWKu0CgYEA7ck/zNf/Gc0jO2qEkD+CezAgHge0jwQapgZseYmELWc1rJc1bOL1
					uYhcAAHmH3bvT1O6TL6MpGsa+YulfEpDz9VpZteIJZJ1KFVKdgh2KQv/Gj6lTYKU
					exXWWQOkgRmzB0c00SqpTaqCo8ZAjw57cjgxjVZihUyez6IUTQazflcCgYEAyIb3
					ObiSbJH4RryvqdsoJu+GuCeoRRMYCuHKiq8BZ9lmDXEM128jdV4Hp6jRs+gPpRX5
					C9+7jawPWDh6fNvFVE1zaDmYyoQ0cIOqbfA9KQjKn6mnT+DJXUizPtUcLJFGFqlD
					Wg3EQ4l6ai2l94KpBDTJsfxug7MO2h4B04iJE88CgYEAuDUZmcUSuJg0XQkNnPm2
					SVxk5R6u/8P8KPX8/sJLhSjZadTR7IJ+PbanHtJZxbJLfbatMlrDdXQLt5o5Huoh
					UlZPiv4ZWJH29MHuJzYy42WJwHkbccpg4GFwZhDuVZzlFhRRlGBqO+KFxf4FcU2U
					0E08BfQP6pgKx2sWMv2n+40CgYEAnqOfnEt3k2rbduK5OfBGUJ83/iJpjdPwNlOw
					j4yp2QV1JfckyJ6E98oe1jXJSMGy9tBuSUWDtC3Fqe5sgLDA6NOpFHBUfwqeDdEs
					GHNxfzAUVMG7uobD5wenvnKMKnn3b+ASh4DSnvd5H9zjKu90VP6J/kQNDiWu/0G0
					AixG/aMCgYEA4Rxm/5MKYWBV3WsTfSLJaIVHz5BZX1tsTXzaFjOuRvUQ/+tS5Ybv
					2rFs71mMvy+tm0Yt9pHvc/i8rB9tc8XWLJkwEJybZbRiNjHFXa2k15zMGZBzhlLO
					6K+HhRKjIq2ZxNNtzlwE/sBOCuoYJj9olC+lPZQ08hP2KIXcnncDtV0=`

	var appId = "2016092400588604"
	var aliPublicKey = `MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAukKW0jfG3C0ZL0pDdTta
						Wi2yhf6yP41QN3kL8HktxdnKsLws4PvRh1ddqD4mQEFQDxPjYj4kRbPQkhLjcQJz
						ohNoLOUhpstXH+h0q7jhWbgwWpQLnwV1ktYvZ+wrA7pHmkkrvq1wpKWJRHxlG50x
						ZZ2g2M7octL6IKpYRAx/IY3Dxn2Mvk0PArBw3/7aDEmuSIAhBqmE/+Me6kFIcOTl
						8B1yF1ey+roPJyVABQmRCKAbP/rhh6RRUesK6lax8+G+1RjqULHZL1A+1qgG2CN0
						VqHxEqQ86K0AnBUZLi75Ex++BWQw1v9AfB7py3DmbBOFUmtm3ZLXXzzR+YupkhKd
						WQIDAQAB`

	var client = alipay.New(appId, aliPublicKey, privateKey, false)

	//alipay.trade.page.pay
	var p = alipay.AliPayTradePagePay{}
	p.NotifyURL = "http://192.168.111.132:8080/user/payOk"
	p.ReturnURL = "http://192.168.111.132:8080/user/payOk"
	p.Subject = "天天生鲜"
	p.OutTradeNo = "1234567811"
	p.TotalAmount = "10000.00"
	p.ProductCode = "FAST_INSTANT_TRADE_PAY"

	var url, err = client.TradePagePay(p)
	if err != nil {
		fmt.Println(err)
	}

	var payURL = url.String()

	this.Redirect(payURL, 302)
}

func (this *OrderController) PayOK() {
	treadNo := this.GetString("trade_no")
	if treadNo != "" {
		/* Receive Message
		http://192.168.111.132:8080/user/payOk?
		charset=utf-8&
		out_trade_no=1234567811&
		method=alipay.trade.page.pay.return&
		total_amount=10000.00&
		sign=Xsczh7hgOJcej6C9WefyamxMqCWLel943jYEx7RWHJDG%2FXUFq0Nub2v60%2FmsWQDCuZJxzbkv1BqcYPyl%2B8Baac%2BXvCM2Cxlbmmlxc7veQAt9SekQXzePnomGDhKTY3h9AQpml6HXZ2DxAGV2aLLqLzM6i0cGTZHxPtoYnnmkMS3MzR76LGltr2eYDT3UvfnfTDtRYHI4BBq%2F09FFAmtnjJEfPs7MIvOBRcEK3rD%2BnznusuX5IH9eq7M1juMZYkWz5JAdZaTACZASXhmIPdgNebXVc9R%2Fp0yGRyU2EhXhkDfD3OZbvBC8wLx6l8FQG2W1kVKkelCw2mEls9jR9vRk3A%3D%3D&
		trade_no=2019011322001470680500694414&
		auth_app_id=2016092400588604&
		version=1.0&
		app_id=2016092400588604&
		sign_type=RSA2&
		seller_id=2088102177119669&
		timestamp=2019-01-13+18%3A41%3A48*/
	}
	this.Redirect("/goods/userCenterOrder", 302)
}

func (this *OrderController) SendMsg() {
	var (
		gatewayUrl      = "http://dysmsapi.aliyuncs.com/"
		accessKeyId     = "LTAIN9gZtWEmkc1e"
		accessKeySecret = "H7wFlnWODmifC7DHgps21wfO5GRn1e"
		phoneNumbers    = "1234567890"  //要发送的电话号码
		signName        = "天天生鲜"     //签名名称
		templateCode    = "SMS_149101793"  //模板号
		code = "iloveyou"
		templateParam   = "{\"code\":\""+code+"\"}"//验证码
	)

	smsClient := aliyunsmsclient.New(gatewayUrl)
	result, err := smsClient.Execute(accessKeyId, accessKeySecret, phoneNumbers, signName, templateCode, templateParam)
	//fmt.Println("Got raw response from server:", string(result.RawResponse))
	if err != nil {
		beego.Info("配置有问题")
	}

	if result.IsSuccessful() {
		this.Ctx.WriteString("发送成功")
		beego.Error("短信成功")
	} else {
		beego.Error("短信失败")
	}

}