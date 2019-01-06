package controllers

import "github.com/astaxie/beego"

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
	this.TplName = "index.html"
}