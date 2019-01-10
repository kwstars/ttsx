package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"ttsx/models"
	"github.com/gomodule/redigo/redis"
	"math"
)

type GoodsController struct {
	beego.Controller
}

// 查询商品类型
func showGoodsTypes(this *GoodsController) (goodsTypes []models.GoodsType) {
	o := orm.NewOrm()
	o.QueryTable("GoodsType").All(&goodsTypes)
	return goodsTypes
}

// 详情页分页
func pageEditor(pageCount, pageIndex int) []int {
	var pages []int
	if pageCount < 5 {
		pages = make([]int, pageCount)
		for i := 1; i <= pageCount; i++ {
			pages[i-1] = i
		}
	} else if pageIndex <= 3 {
		pages = make([]int, 5)
		for i := 1; i <= 5; i++ {
			pages[i-1] = i
		}
	} else if pageIndex >= pageCount-2 {
		pages = make([]int, 5)
		for i := 1; i <= 5; i++ {
			pages[i-1] = pageCount - 5 + i
		}
	} else {
		pages = make([]int, 5)
		for i := 1; i <= 5; i++ {
			pages[i-1] = pageIndex - 3 + i
		}
	}
	return pages
}

// 获取当前登陆的用户
func GetGoodsUser(this *GoodsController) (userName interface{}) {
	userName = this.GetSession("userName")
	if userName == nil {
		this.Data["userName"] = ""
	} else {
		this.Data["userName"] = userName.(string)
	}
	return
}

func (this *GoodsController) ShowIndex() {
	GetGoodsUser(this)
	o := orm.NewOrm()
	// 查询商品类型
	goodsTypes := showGoodsTypes(this)

	// 查询商品轮播图
	var goodsBanner []models.IndexGoodsBanner
	o.QueryTable("IndexGoodsBanner").OrderBy("Index").All(&goodsBanner)
	this.Data["goodsBanner"] = goodsBanner

	// 查询促销商品
	var promotionBanner []models.IndexPromotionBanner
	o.QueryTable("IndexPromotionBanner").OrderBy("Index").All(&promotionBanner)
	this.Data["promotionBanner"] = promotionBanner

	//获取分类商品展示
	var goods []map[string]interface{}
	for _, goodsType := range goodsTypes {
		temp := make(map[string]interface{})
		temp["goodsType"] = goodsType
		goods = append(goods, temp)
	}

	var goodsText []models.IndexTypeGoodsBanner
	var goodsImage []models.IndexTypeGoodsBanner

	for _, goodsMap := range goods {
		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU", "GoodsType").
			Filter("GoodsType", goodsMap["goodsType"]).Filter("DisplayType", 0).OrderBy("Index").All(&goodsText)

		o.QueryTable("IndexTypeGoodsBanner").RelatedSel("GoodsSKU", "GoodsType").
			Filter("GoodsType", goodsMap["goodsType"]).Filter("DisplayType", 1).OrderBy("Index").All(&goodsImage)

		goodsMap["goodsText"] = goodsText
		goodsMap["goodsImage"] = goodsImage
	}

	this.Layout = "layout.html"
	this.Data["goodsTypes"] = goodsTypes
	this.Data["goods"] = goods
	this.TplName = "index.html"
}

func (this *GoodsController) ShowDetail() {
	GetGoodsUser(this)
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
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType", goodsSku).
		OrderBy("Time").Limit(2, 0).All(&newGoods)

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
		conn.Do("ltrim", "history_"+userName.(string), 0, 4)
	}

	//this.Data["loginUser"] = loginUser
	this.Layout = "layout.html"
	this.Data["goodsId"] = goodsId
	this.TplName = "detail.html"
}

func (this *GoodsController) ShowList() {
	GetGoodsUser(this)
	typeId, err := this.GetInt("typeId")
	if err != nil {
		beego.Error("获取类型ID错误")
		this.Redirect("/", 302)
		return
	}

	goodsTypes := showGoodsTypes(this)

	// 获取当前类型的商品
	o := orm.NewOrm()
	var goodsSkus []models.GoodsSKU
	qs := o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId)
	count, err := qs.Count()
	if err != nil {
		beego.Error("请求连接错误")
		this.Redirect("/", 302)
		return
	}

	pageSize := 2
	pageCount := math.Ceil(float64(count) / float64(pageSize))
	pageIndex, err := this.GetInt("pageIndex")
	if err != nil {
		pageIndex = 1
	}
	pages := pageEditor(int(pageCount), pageIndex)

	start := (pageIndex - 1) * pageSize

	sort := this.GetString("sort")
	if sort == "" {
		qs.Limit(pageSize, start).All(&goodsSkus)
	} else if sort == "price" {
		qs.OrderBy("Price").Limit(pageSize, start).All(&goodsSkus)
	} else {
		qs.OrderBy("Sales").Limit(pageSize, start).All(&goodsSkus)
	}

	var preIndex, nextIndex int
	if pageIndex == 1 {
		preIndex = 1
	} else {
		preIndex = pageIndex - 1
	}

	if pageIndex == int(pageCount) {
		nextIndex = int(pageCount)
	} else {
		nextIndex = pageIndex + 1
	}

	// 新品推荐
	var newGoods []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).
		OrderBy("Time").Limit(2, 0).All(&newGoods)

	this.Data["goodsTypes"] = goodsTypes
	this.Data["newGoods"] = newGoods
	this.Data["sort"] = sort
	this.Data["typeId"] = typeId
	this.Data["preIndex"] = preIndex
	this.Data["nextIndex"] = nextIndex
	this.Data["pageIndex"] = pageIndex

	this.Data["pages"] = pages
	this.Data["goodsSkus"] = goodsSkus

	this.Layout = "layout.html"
	this.TplName = "list.html"
}
