package main

import (
	"context"
	"fmt"
	"github.com/PeterYangs/siteCopy2"
	"time"
)

func main() {

	c := siteCopy2.NewCopy(context.Background())

	c.Delay(100 * time.Millisecond)

	c.Url("http://www.jinpengjiuping.com/", "首页.html")
	//c.Url("http://www.jinpengjiuping.com/page/453.html", "关于我们.html")
	//c.Url("http://www.jinpengjiuping.com/list/26.html", "酒瓶展示.html")
	//c.Url("http://www.jinpengjiuping.com/article/26/501.html", "酒瓶详情.html")
	//c.Url("http://www.jinpengjiuping.com/list/29.html", "新闻动态.html")
	//c.Url("http://www.jinpengjiuping.com/article/29/658.html", "新闻详情.html")
	//c.Url("http://www.jinpengjiuping.com/page/499.html", "生产车间.html")
	//c.Url("http://www.jinpengjiuping.com/page/455.html", "在线留言.html")
	//c.Url("http://www.jinpengjiuping.com/page/3.html", "联系我们.html")

	err := c.Zip("酒.zip")

	if err != nil {

		fmt.Println(err)
	}

}
