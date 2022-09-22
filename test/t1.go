package main

import (
	"context"
	"fmt"
	"github.com/PeterYangs/siteCopy2"
)

func main() {

	c := siteCopy2.NewCopy(context.Background())

	c.Url("https://www.diyiyou.com/", "index.html")
	c.Url("https://www.diyiyou.com/newgame/", "news.html")

	err := c.Zip("aa.zip")

	if err != nil {

		fmt.Println(err)

		return
	}
}
