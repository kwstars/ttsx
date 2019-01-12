package main

import (
	_ "ttsx/routers"
	"github.com/astaxie/beego"
	_ "ttsx/models"
)

func main() {
	beego.AddFuncMap("add", OrderAddOne)
	beego.Run()
}

func OrderAddOne(in int) (out int)  {
	out = in + 1
	return
}