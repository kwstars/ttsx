package controllers

import (
	"github.com/astaxie/beego"
	"regexp"
	"github.com/astaxie/beego/orm"
	"ttsx/models"
)

type UserController struct {
	beego.Controller
}

func (this *UserController) ShowRegister() {
	this.TplName = "register.html"
}

func (this *UserController) HandleRegister() {
	userName := this.GetString("user_name")
	pwd := this.GetString("pwd")
	cpwd := this.GetString("cpwd")
	email := this.GetString("email")
	if userName == "" || pwd == "" || cpwd == "" || email == "" {
		this.Data["errmsg"] = "输入数据不完整,请重新输入"
		this.TplName = "register.html"
		return
	}

	reg, err := regexp.Compile(`.*`)
	if err != nil {
		this.Data["errmsg"] = "正则创建失败"
		this.TplName = "register.html"
		return
	}

	res := reg.MatchString(email)
	if res == false {
		this.Data["errmsg"] = "邮箱格式不正确，请重新校验"
		this.TplName = "register.html"
		return
	}

	if pwd != pwd {
		this.Data["errmsg"] = "两次输入密码不一致"
		this.TplName = "register.html"
		return
	}

	o := orm.NewOrm()
	var user models.User
	user.UserName = userName
	user.Pwd = pwd
	user.Email = email
	id, err := o.Insert(&user)
	if err != nil {
		this.Data["errmsg"] = "用户名重复，请重新输入"
		this.TplName = "register.html"
		return
	}
	beego.Info(id)

	this.Redirect("/login", 302)
}

func (this *UserController) ShowLogin() {
	this.TplName = "login.html"
}

func (this *UserController) HandleLogin() {

}
