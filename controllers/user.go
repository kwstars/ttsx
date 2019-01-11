package controllers

import (
	"github.com/astaxie/beego"
	"regexp"
	"github.com/astaxie/beego/orm"
	"ttsx/models"
	"github.com/astaxie/beego/utils"
	"strconv"
	"github.com/gomodule/redigo/redis"
)

type UserController struct {
	beego.Controller
}

// 用户登陆注册退出
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

func (this *UserController) ShowLogin() {
	userName := this.Ctx.GetCookie("userName")
	beego.Info(userName)
	if userName != "" {
		this.Data["userName"] = userName
		this.Data["checked"] = "checked"
	} else {
		this.Data["userName"] = ""
		this.Data["checked"] = ""
	}
	this.TplName = "login.html"
}

func (this *UserController) HandleLogin() {
	userName := this.GetString("username")
	pwd := this.GetString("pwd")
	remember := this.GetString("remember")

	if userName == "" || pwd == "" {
		beego.Error("输入数据不完整，请重新输入")
		this.TplName = "login.html"
		return
	}

	o := orm.NewOrm()
	var user models.User
	user.UserName = userName
	err := o.Read(&user, "userName")
	if err != nil {
		beego.Error("用户名不存在")
		this.TplName = "login.html"
		return
	}

	if user.Pwd != pwd {
		beego.Error("用户密码不正确")
		this.TplName = "login.html"
		return
	}

	if user.Active == 0 {
		beego.Error("用户未激活，请先激活")
		this.TplName = "login.html"
		return
	}
	beego.Info("用户登陆校验完成")
	beego.Info(userName)
	beego.Info(pwd)

	//remember := this.GetString("remember")
	beego.Info(remember)

	if remember == "on" {
		this.Ctx.SetCookie("userName", userName, 3600)
	} else {
		this.Ctx.SetCookie("userName", userName, -1)
	}

	beego.Info(this.Ctx.GetCookie("userName"))

	this.SetSession("userName", userName)
	this.Redirect("/", 302)
}

func (this *UserController) UserLogout() {
	this.DelSession("userName")
	this.Redirect("/", 302)
}

// 获取当前登陆的用户
func GetUser(this *UserController) (userName interface{}) {
	userName = this.GetSession("userName")
	if userName == nil {
		this.Data["userName"] = ""
	} else {
		this.Data["userName"] = userName.(string)
	}
	return
}

func (this *UserController) ShowUserCenterInfo() {
	currentLoginUser := GetUser(this)

	o := orm.NewOrm()
	var receiver models.Receiver
	qs := o.QueryTable("Receiver").RelatedSel("User").Filter("User__UserName", currentLoginUser.(string))
	qs.Filter("IsDefault", true).One(&receiver)

	// 获取最近游览记录
	conn, err := redis.Dial("tcp", ":6379")
	if err != nil {
		beego.Error("redis连接失败")
		return
	}
	defer conn.Close()
	res, err := redis.Ints(conn.Do("lrange", "history_"+currentLoginUser.(string), 0, 4))

	var goods []models.GoodsSKU

	for _, goodsId := range res {
		var goodsSku models.GoodsSKU
		goodsSku.Id = goodsId
		o.Read(&goodsSku)
		goods = append(goods, goodsSku)
	}

	this.Data["goods"] = goods
	this.Data["receiver"] = receiver

	this.Layout = "layout_user.html"
	this.TplName = "user_center_info.html"
}

func (this *UserController) ShowUserCenterOrder() {
	GetUser(this)
	this.Layout = "layout_user.html"
	this.TplName = "user_center_order.html"
}

func (this *UserController) ShowUserCenterSite() {
	GetUser(this)
	userName := this.GetSession("userName")

	// 显示地址
	o := orm.NewOrm()
	var receiver models.Receiver
	qs := o.QueryTable("Receiver").RelatedSel("User").Filter("User__UserName", userName.(string))
	qs.Filter("IsDefault", true).One(&receiver)

	this.Data["receiver"] = receiver

	this.Layout = "layout_user.html"
	this.TplName = "user_center_site.html"
}

func (this *UserController) HeandleShowUserCenterSite() {
	currentLoginUser := this.GetSession("userName")
	name := this.GetString("name")
	zipCode := this.GetString("zipcode")
	addr := this.GetString("addr")
	phone := this.GetString("phone")

	// 校验数据不能为空
	if name == "" || addr == "" || zipCode == "" || phone == "" {
		this.Data["errmsg"] = "添加地址数据不能为空"
		this.TplName = "user_center_site.html"
		return
	}

	// 校验邮箱地址

	// 校验手机号码

	o := orm.NewOrm()
	var receiver models.Receiver
	receiver.Name = name
	receiver.Phone = phone
	receiver.ZipCode = zipCode
	receiver.Addr = addr
	receiver.IsDefault = true

	var user models.User
	user.UserName = currentLoginUser.(string)
	o.Read(&user, "UserName")
	receiver.User = &user

	o.Begin()
	num, err := o.QueryTable("Receiver").RelatedSel("User").Filter("User__Id", user.Id).Update(orm.Params{"IsDefault": false})
	//count, err := qs.Filter("IsDefault", true).
	beego.Info("update记录数", num)
	if err != nil {
		beego.Error("update数据错误 ", err)
		o.Rollback()
	}

	insertId, err := o.Insert(&receiver)
	beego.Info("primary key", insertId)
	if err != nil {
		beego.Error("insert数据错误 ", err)
		o.Rollback()
	}
	err = o.Commit()
	if err != nil {
		beego.Error("用户地址数据更新错误", err)
	}

	this.Redirect("/goods/userCenterSite", 302)
}
