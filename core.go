package siteCopy2

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"github.com/PeterYangs/request/v2"
	"github.com/PeterYangs/tools"
	"github.com/PeterYangs/tools/link"
	"github.com/PuerkitoBio/goquery"
	uuid "github.com/satori/go.uuid"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type SiteCopy struct {
	client      *request.Client
	workChan    chan workItem
	fileMap     sync.Map
	pathList    map[string]bool
	lock        sync.Mutex
	SiteUrlList []*SiteUrl
	zipWriter   *zip.Writer
	cxt         context.Context
	cancel      context.CancelFunc
	delay       time.Duration
	wait        sync.WaitGroup
}

type workItem struct {
	link     string
	filename string
	tagName  string
}

type SiteUrl struct {
	SiteCopy *SiteCopy
	u        string //原链接
	host     string
	scheme   string
	name     string
}

//var f *os.File

func NewCopy(cxt context.Context) *SiteCopy {

	//f, _ = os.OpenFile("path.txt", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)

	c, cancel := context.WithCancel(cxt)

	return &SiteCopy{
		workChan: make(chan workItem, 10),
		fileMap:  sync.Map{},
		lock:     sync.Mutex{},
		cxt:      c,
		cancel:   cancel,
		wait:     sync.WaitGroup{},
		pathList: make(map[string]bool),
		//waitSon:  sync.WaitGroup{},
	}
}

//初始化客户端
func (sc *SiteCopy) initClient() *SiteCopy {

	client := request.NewClient()

	client.Header(map[string]string{
		"Accept":             "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3;q=0.9",
		"Accept-Encoding":    "gzip, deflate, br",
		"Accept-Language":    "zh-CN,zh;q=0.9,en;q=0.8,en-GB;q=0.7,en-US;q=0.6",
		"sec-ch-ua-platform": "\"Windows\"",
		"User-Agent":         "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/102.0.5005.63 Safari/537.36 Edg/102.0.1245.33",
	})

	sc.client = client

	return sc

}

func (sc *SiteCopy) Url(u string, name string) {

	up, _ := url.Parse(u)

	sl := &SiteUrl{
		u:        u,
		SiteCopy: sc,
		host:     up.Host,
		scheme:   up.Scheme,
		name:     name,
	}

	sc.SiteUrlList = append(sc.SiteUrlList, sl)

}

func (sc *SiteCopy) Proxy(url string) *SiteCopy {

	sc.client.Proxy(url)

	return sc
}

// Delay 设置请求延迟
func (sc *SiteCopy) Delay(delay time.Duration) *SiteCopy {

	sc.delay = delay

	return sc

}

func (sc *SiteCopy) work() {

	defer sc.wait.Done()

	for item := range sc.workChan {

		if sc.delay != 0 {

			time.Sleep(sc.delay)

		}

		ct, err := sc.client.R().GetToContent(item.link)

		if err != nil {

			fmt.Println(err)

			continue
		}

		data := ct.ToString()

		//下载css文件中图片
		if item.tagName == "link" {

			s := regexp.MustCompile(`url\((.*?)\)`).FindAllStringSubmatch(data, -1)

			if len(s) > 0 {

				mm := make(map[string]bool)

				for _, i2 := range s {

					if i2[1] == "" {

						continue
					}

					mm[i2[1]] = true

				}

				for s2, _ := range mm {

					path := strings.Replace(strings.Replace(s2, `"`, "", -1), `'`, "", -1)

					realLink, cssErr := link.GetCompleteLink(item.link, path)

					if cssErr != nil {

						fmt.Println(cssErr)

						continue
					}

					localName, ok := sc.dealLinkToLocalName(realLink, "img")

					if !ok {

						//sc.push(realLink, localName, "img")

						ct1, err1 := sc.client.R().GetToContent(realLink)

						if err1 != nil {

							fmt.Println(err1)

							continue
						}

						data1 := ct1.ToString()

						wErr1 := sc.WriteZip(localName, []byte(data1))

						if wErr1 != nil {

							fmt.Println(wErr1)

							continue
						}

					}

					data = strings.Replace(data, s2, "../"+localName, -1)

				}

			}

		}

		wErr := sc.WriteZip(item.filename, []byte(data))

		if wErr != nil {

			fmt.Println(wErr)

			continue
		}

	}

}

func (sc *SiteCopy) push(link string, filename string, tagName string) {

	sc.workChan <- workItem{link: link, filename: filename, tagName: tagName}

}

func (sc *SiteCopy) removeQuery(link string) string {

	re1 := regexp.MustCompile(`(\?.*?)$`).FindStringSubmatch(link)

	if len(re1) > 0 {

		return strings.Replace(link, re1[1], "", 1)
	}

	return link
}

//网络链接文件转本地路径
func (sc *SiteCopy) dealLinkToLocalName(link string, tagName string) (string, bool) {

	sc.lock.Lock()

	defer sc.lock.Unlock()

	localFilename, ok := sc.fileMap.Load(link)

	if ok {

		return localFilename.(string), true
	}

	filename := sc.removeQuery(filepath.Base(link))

	path := ""

	ext := filepath.Ext(filename)

	if tagName == "link" {

		switch ext {

		case ".css":

			path = "css/" + filename

			break

		default:

			path = "css/" + filename

			break

		}

	}

	if tagName == "script" {

		path = "js/" + filename

	}

	if tagName == "img" {

		switch ext {

		case ".png":

			path = "image/" + filename

			break

		case ".gif":

			path = "image/" + filename

			break

		case ".jpg":

			path = "image/" + filename

			break

		case ".woff2":

			path = "font/" + filename

			break

		case ".ttf":

			path = "font/" + filename

			break

		case ".eot":

			path = "font/" + filename

			break

		case ".svg":

			path = "font/" + filename

			break

		default:

			path = "image/" + uuid.NewV4().String() + ".png"

			break

		}

	}

	path = sc.dealPath(path)

	sc.fileMap.Store(link, path)

	sc.pathList[path] = false

	return path, false

}

//同名文件重命名
func (sc *SiteCopy) dealPath(path string) string {

	index := 0

	for {

		_, ok := sc.pathList[path]

		if ok {

			name := strings.Replace(path, filepath.Ext(path), "", 1)

			index++

			rename := name + strconv.Itoa(index)

			path = strings.Replace(path, name, rename, 1)

		} else {

			return path
		}

	}

}

// WriteZip 往压缩包写入一个文件
func (sc *SiteCopy) WriteZip(name string, content []byte) error {

	sc.lock.Lock()

	defer sc.lock.Unlock()

	w, err := sc.zipWriter.Create(name)

	if err != nil {

		return err
	}

	_, err = io.Copy(w, bytes.NewReader(content))

	if err != nil {

		return err
	}

	return nil

}

func (sc *SiteCopy) Zip(name string) error {

	//初始化客户端
	sc.initClient()

	for i := 0; i < 10; i++ {

		sc.wait.Add(1)

		go sc.work()
	}

	archive, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)

	if err != nil {

		return err

	}

	zipWriter := zip.NewWriter(archive)

	sc.zipWriter = zipWriter

	defer zipWriter.Close()

	for _, siteUrl := range sc.SiteUrlList {

		ct, httpErr := sc.client.R().GetToContent(siteUrl.u)

		if httpErr != nil {

			fmt.Println(httpErr)

			continue
		}

		html := ct.ToString()

		html, dErr := siteUrl.dealCoding(html, ct.Header())

		if dErr != nil {

			fmt.Println(dErr)

			continue
		}

		doc, gErr := goquery.NewDocumentFromReader(strings.NewReader(html))

		if gErr != nil {

			fmt.Println(gErr)

			continue
		}

		//处理样式文件
		doc.Find("link").Each(func(i int, selection *goquery.Selection) {

			v, ok := selection.Attr("href")

			if ok {

				webLink, _ := link.GetCompleteLink(siteUrl.u, v)

				localName, okk := sc.dealLinkToLocalName(webLink, "link")

				if !okk {

					sc.push(webLink, localName, "link")
				}

				selection.SetAttr("href", localName)

			}

		})

		//处理图片
		doc.Find("img").Each(func(i int, selection *goquery.Selection) {

			v, ok := selection.Attr("src")

			if ok {

				webLink, _ := link.GetCompleteLink(siteUrl.u, v)

				localName, okk := sc.dealLinkToLocalName(webLink, "img")

				if !okk {

					sc.push(webLink, localName, "img")
				}

				selection.SetAttr("src", localName)

			}

		})

		//处理js脚本
		doc.Find("script").Each(func(i int, selection *goquery.Selection) {

			v, ok := selection.Attr("src")

			if ok {

				webLink, _ := link.GetCompleteLink(siteUrl.u, v)

				localName, okk := sc.dealLinkToLocalName(webLink, "script")

				if !okk {

					sc.push(webLink, localName, "script")
				}

				selection.SetAttr("src", localName)

			}

		})

		html, hErr := doc.Html()

		if hErr != nil {

			fmt.Println(hErr)

			continue
		}

		sc.WriteZip(siteUrl.name, []byte(html))

	}

	sc.closeChan()

	sc.wait.Wait()

	return nil

}

func (sc *SiteCopy) closeChan() {

	for {

		select {
		case <-time.After(200 * time.Millisecond):

			if len(sc.workChan) <= 0 {

				close(sc.workChan)

				return
			}

		}

	}

}

// GetLink 获取完整链接
func (sl *SiteUrl) getLink(href string) string {

	case1, _ := regexp.MatchString("^/[a-zA-Z0-9_]+.*", href)

	case2, _ := regexp.MatchString("^//[a-zA-Z0-9_]+.*", href)

	case3, _ := regexp.MatchString("^(http|https).*", href)

	switch true {

	case case1:

		href = sl.scheme + "://" + sl.host + href

		break

	case case2:

		//获取当前网址的协议
		//res := regexp.MustCompile("^(https|http).*").FindStringSubmatch(sl.host)

		href = sl.scheme + "://" + sl.host + href

		break

	case case3:

		break

	default:

		href = sl.scheme + "://" + sl.host + "/" + href
	}

	return href

}

// DealCoding 解决编码问题
func (sl *SiteUrl) dealCoding(html string, header http.Header) (string, error) {

	//return html, nil

	headerContentType_ := header["Content-Type"]

	if len(headerContentType_) > 0 {

		headerContentType := headerContentType_[0]

		charset := sl.getCharsetByContentType(headerContentType)

		charset = strings.ToLower(charset)

		switch charset {

		case "gbk":

			return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

		case "gb2312":

			return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

		case "utf-8":

			return html, nil

		case "utf8":

			return html, nil

		case "euc-jp":

			return string(tools.ConvertToByte(html, "euc-jp", "utf8")), nil

		case "":

			break

		default:
			return string(tools.ConvertToByte(html, charset, "utf8")), nil

		}

	}

	code, err := goquery.NewDocumentFromReader(strings.NewReader(html))

	if err != nil {

		return html, err
	}

	contentType, _ := code.Find("meta[charset]").Attr("charset")

	//转小写
	contentType = strings.TrimSpace(strings.ToLower(contentType))

	switch contentType {

	case "gbk":

		return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

	case "gb2312":

		return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

	case "utf-8":

		return html, nil

	case "utf8":

		return html, nil

	case "euc-jp":

		return string(tools.ConvertToByte(html, "euc-jp", "utf8")), nil

	case "":

		break
	default:
		return string(tools.ConvertToByte(html, contentType, "utf8")), nil

	}

	contentType, _ = code.Find("meta[http-equiv=\"Content-Type\"]").Attr("content")

	charset := sl.getCharsetByContentType(contentType)

	switch charset {

	case "utf-8":

		return html, nil

	case "utf8":

		return html, nil

	case "gbk":

		return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

	case "gb2312":

		return string(tools.ConvertToByte(html, "gbk", "utf8")), nil

	case "euc-jp":

		return string(tools.ConvertToByte(html, "euc-jp", "utf8")), nil

	case "":

		break

	default:
		return string(tools.ConvertToByte(html, charset, "utf8")), nil

	}

	return html, nil
}

// GetCharsetByContentType 从contentType中获取编码
func (sl *SiteUrl) getCharsetByContentType(contentType string) string {

	contentType = strings.TrimSpace(strings.ToLower(contentType))

	//捕获编码
	r, _ := regexp.Compile(`charset=([^;]+)`)

	re := r.FindAllStringSubmatch(contentType, 1)

	if len(re) > 0 {

		c := re[0][1]

		return c

	}

	return ""
}
