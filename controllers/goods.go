package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"ttsx/models"
	"github.com/gomodule/redigo/redis"
)

type GoodsController struct {
	beego.Controller
}

func (this *GoodsController) ShowIndex() {
	currentLoginUser := this.GetSession("userName")
	if currentLoginUser == nil {
		this.Data["userName"] = ""
	} else {
		this.Data["userName"] = currentLoginUser.(string)
	}

	o := orm.NewOrm()
	var goodsTypes []models.GoodsType
	//查询所有的商品类型
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["goodsTypes"] = goodsTypes

	//获取轮播图
	var goodsLunbo []models.IndexGoodsBanner
	o.QueryTable("IndexGoodsBanner").OrderBy("Index").All(&goodsLunbo)
	this.Data["goodsLunbo"] = goodsLunbo

	//获取促销商品
	var goodsPro []models.IndexPromotionBanner
	o.QueryTable("IndexPromotionBanner").OrderBy("Index").All(&goodsPro)
	this.Data["goodsPro"] = goodsPro

	//获取分类商品展示
	var goods []map[string]interface{}
	for _, v := range goodsTypes {
		temp := make(map[string]interface{})
		temp["goodsType"] = v
		goods = append(goods, temp)
	}

	for _, v := range goods {
		qs := o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU", "GoodsType").Filter("GoodsType", v["goodsType"])

		var goodsText []models.IndexTypeGoodsBanner
		qs.Filter("DisplayType", 0).OrderBy("Index").All(&goodsText)

		var goodsImage []models.IndexTypeGoodsBanner
		qs.Filter("DisplayType", 1).OrderBy("Index").All(&goodsImage)

		v["goodsText"] = goodsText
		v["goodsImage"] = goodsImage
	}

	this.Data["goods"] = goods
	this.TplName = "index.html"
}

func (this *GoodsController) ShowDetail() {
	goodsId, err := this.GetInt("goodsId")
	if err != nil {
		beego.Error("请求连接错误")
		this.Redirect("/", 302)
		return
	}

	o := orm.NewOrm()
	var goodsSku models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("Goods", "GoodsType").Filter("Id", goodsId).One(&goodsSku)

	var goodsTypes []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsTypes)

	var newGoods []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType", goodsSku).OrderBy("Time").Limit(2, 0).All(&newGoods)

	this.Data["goodsSku"] = goodsSku
	this.Data["goodsTypes"] = goodsTypes
	this.Data["newGoods"] = newGoods

	// 最近商品游览记录保存到redis
	userName := this.GetSession("userName")
	if userName != nil {
		conn, err := redis.Dial("tcp", ":6379")
		if err != nil {
			beego.Error("redis连接失败")
			return
		}
		defer conn.Close()
		conn.Do("lrem", "history_"+userName.(string), 0, goodsId)
		conn.Do("lpush", "history_"+userName.(string), goodsId)
	}

	this.TplName = "detail.html"
}
