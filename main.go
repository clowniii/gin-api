package main

import (
	_ "github.com/jinzhu/gorm/dialects/mysql"

	"gin-app/cmd"
)

func main() {
	cmd.Run()
}
