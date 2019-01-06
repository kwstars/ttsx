package controllers

import (
	"github.com/astaxie/beego"
	"regexp"
	"github.com/astaxie/beego/orm"
	"ttsx/models"
	"github.com/astaxie/beego/utils"
	"strconv"
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

	beego.Info("校验邮箱")
	reg, err := regexp.Compile("^[A-Za-z0-9\u4e00-\u9fa5]+@[a-zA-Z0-9_-]+(\\.[a-zA-Z0-9_-]+)+$")
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

	beego.Info("插入用户信息")
	o := orm.NewOrm()
	var user models.User
	user.UserName = userName
	user.Pwd = pwd
	user.Email = email
	beego.Info("用户邮箱", user.Email)
	id, err := o.Insert(&user)
	if err != nil {
		this.Data["errmsg"] = "用户名重复，请重新输入"
		this.TplName = "register.html"
		return
	}
	beego.Info("用户信息插入成功id=", id)

	config := `{"username":"kwstars@163.com","password":"jejNzdTHZaTFSWH8","host":"smtp.163.com","port":25}`
	email163 := utils.NewEMail(config)
	email163.From = "kwstars@163.com"
	email163.To = []string{user.Email}
	email163.Subject = "Active Account for DailyFresh"
	//email163.HTML = `<a href="http://192.168.111.132:8080/active?userId="` + strconv.Itoa(user.Id) + `>点击激活</a>`
	email163.Text = "http://192.168.111.132:8080/active?id=" + strconv.Itoa(user.Id)
	beego.Info("user.id=", user.Id)
	email163.Send()
	beego.Info("邮件已发送")
	//this.Redirect("/login", 302)
	this.Ctx.WriteString("注册成功，请激活账号")
}

func (this *UserController) ShowLogin() {
	this.TplName = "login.html"
}

func (this *UserController) HandleLogin() {

}

func (this *UserController) ActiveUser() {
	id, err := this.GetInt("id")
	if err != nil {
		beego.Info("userId=", id, err)
		this.Data["errmsg"] = "GetInt,激活失败"
		this.TplName = "register.html"
		return
	}
	o := orm.NewOrm()
	var user models.User
	user.Id = id
	err = o.Read(&user)
	if err != nil {
		this.Data["errmsg"] = "激活失败，用户不存在"
		this.TplName = "register.html"
		return
	}

	user.Active = 1
	num, err := o.Update(&user)
	if err != nil {
		this.Data["errmsg"] = "激活失败，更新用户出问题了"
		this.TplName = "register.html"
		return
	}
	beego.Info(num)

	this.Redirect("login", 302)
}
