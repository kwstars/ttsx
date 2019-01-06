package models

import (
	"github.com/astaxie/beego/orm"
	_ "github.com/go-sql-driver/mysql"
)

type User struct {
	Id        int
	UserName  string      `orm:"unique;size(100)"`
	Pwd       string      `orm:"size(100)"`
	Email     string
	Power     int         `orm:"default(0)"` //0 普通   1 管理员
	Active    int         `orm:"default(0)"` //0 未激活 1 激活
	Receivers []*Receiver `orm:"reverse(many)"`
}

type Receiver struct {
	Id        int
	Name      string
	ZipCode   string
	Addr      string
	Phone     string
	IsDefault bool  `orm:"default(false)"`
	User      *User `orm:"rel(fk)"`
}

func init() {
	orm.RegisterDataBase("default", "mysql", "root:123456@tcp(127.0.0.1:3306)/dailyfresh?charset=utf8")
	orm.RegisterModel(new(User), new(Receiver))
	orm.RunSyncdb("default", false, true)
}
