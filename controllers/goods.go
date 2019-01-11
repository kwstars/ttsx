package controllers

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/orm"
	"ttsx/models"
	"github.com/gomodule/redigo/redis"
	"math"
	"strconv"
)

type GoodsController struct {
	beego.Controller
}

// 查询商品类型
func showGoodsTypes(this *GoodsController) (goodsTypes []models.GoodsType) {
	o := orm.NewOrm()
	o.QueryTable("GoodsType").All(&goodsTypes)
	this.Data["goodsTypes"] = goodsTypes
	return goodsTypes
}

// 详情页分页
func pageCal(pageCount, pageIndex int) []int {
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

// 显示主页
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

	this.Layout = "layout_goods.html"
	this.Data["goods"] = goods
	this.TplName = "index.html"
}

// 显示详情页
func (this *GoodsController) ShowDetail() {
	GetGoodsUser(this)
	goodsId, err := this.GetInt("goodsId")
	if err != nil {
		beego.Error("请求连接错误")
		this.Redirect("/", 302)
		return
	}

	o := orm.NewOrm()
	// 商品的SKU
	var goodsSku models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("Goods", "GoodsType").Filter("Id", goodsId).One(&goodsSku)

	// 查询所有商品类别
	showGoodsTypes(this)

	// 新品推荐
	var newGoods []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType", goodsSku).
		OrderBy("Time").Limit(2, 0).All(&newGoods)

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

	this.Data["goodsSku"] = goodsSku
	this.Data["newGoods"] = newGoods
	this.Data["goodsId"] = goodsId
	this.TplName = "detail.html"
}

// 显示列表页
func (this *GoodsController) ShowList() {
	GetGoodsUser(this)

	typeId := this.GetString("typeId")
	o := orm.NewOrm()

	// 获取全部商品类型
	var goodsTypes []models.GoodsType
	o.QueryTable("GoodsType").All(&goodsTypes)

	// 设置页面的大小
	var pageSize = 2

	// 获取当前所在的页面 index
	pageIndexS := this.GetString("pageIndex")
	var pageIndex int
	var err error
	if pageIndexS == "" {
		pageIndex = 1
	} else {
		pageIndex, err = strconv.Atoi(pageIndexS)
		if err != nil {
			beego.Error("atoi失败", err)
			pageIndex = 1
		}
	}

	// 获取页面的开始位置
	var start = (pageIndex - 1) * pageSize
	var goods []models.GoodsSKU
	sort := this.GetString("sort")
	var qs orm.QuerySeter
	var goodsCount int64
	var navigationBarGoodsType models.GoodsSKU
	if typeId == "" {
		qs = o.QueryTable("GoodsSKU")
		goodsCount, _ = qs.Count()
	} else {
		id, err := strconv.Atoi(typeId)
		if err != nil {
			beego.Error("atoi错误", err)
			this.Redirect("/", 302)
			return
		}
		qs = o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", id)
		goodsCount, _ = qs.Count()

		// 导航栏 类目显示
		qs.One(&navigationBarGoodsType)
		this.Data["navigationBarGoodsType"] = navigationBarGoodsType
	}

	if sort == "sales" {
		qs.OrderBy("Sales").Limit(pageSize, start).All(&goods)
	} else if sort == "price" {
		qs.OrderBy("Price").Limit(pageSize, start).All(&goods)
	} else {
		qs.Limit(pageSize, start).All(&goods)
	}

	// 获取页面的数量
	pageCount := math.Ceil(float64(goodsCount) / float64(pageSize))

	// 对页面进行分页
	var pages []int
	pages = pageCal(int(pageCount), pageIndex)

	// 下一页, 上一页
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
	beego.Info(preIndex, pageIndex, nextIndex, pageCount)

	// 新品推荐
	var newGoods []models.GoodsSKU
	o.QueryTable("GoodsSKU").RelatedSel("GoodsType").Filter("GoodsType__Id", typeId).
		OrderBy("Time").Limit(2, 0).All(&newGoods)

	// 新品推荐
	this.Data["newGoods"] = newGoods

	// 排序
	this.Data["sort"] = sort

	// 分页
	this.Data["typeId"] = typeId
	this.Data["pageIndex"] = pageIndex
	this.Data["preIndex"] = preIndex
	this.Data["nextIndex"] = nextIndex
	this.Data["pages"] = pages

	this.Data["goods"] = goods
	this.Data["goodsTypes"] = goodsTypes

	this.Layout = "layout_goods.html"
	this.TplName = "list.html"
}

// 显示搜索页
func (this *GoodsController) HandleSearch() {
	GetGoodsUser(this)
	searchName := this.GetString("searchName")
	if searchName == "" {
		this.Redirect("/", 302)
		return
	}

	o := orm.NewOrm()
	var goods []models.GoodsSKU
	o.QueryTable("GoodsSKU").Filter("Name__contains", searchName).All(&goods)
	this.Data["goods"] = goods

	this.Layout = "layout_goods.html"
	this.TplName = "search.html"
}
