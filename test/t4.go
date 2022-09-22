package main

import (
	"context"
	"fmt"
	"github.com/PeterYangs/siteCopy2"
	"time"
)

func main() {

	c := siteCopy2.NewCopy(context.Background())

	c.Delay(300 * time.Millisecond)

	c.Url("https://www.522gg.com/", "首页.html")

	err := c.Zip("aaa.zip")

	if err != nil {

		fmt.Println(err)
	}

}
