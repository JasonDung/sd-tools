package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sd-tools/config"
	"sort"
	"strconv"
	"strings"
	"time"
)

type Node struct {
	No    int
	Price string
	Url   string
}

type TaskNode []Node

func (p TaskNode) Less(i, j int) bool {
	sortType := config.GetValue("sortType")
	if sortType == "2" {
		return p[i].No < p[j].No
	} else {
		return p[i].Price > p[j].Price
	}
}
func (p TaskNode) Len() int {
	return len(p)
}
func (p TaskNode) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

func getTask(cookieValue string) {

	indexUrl := config.GetValue("indexUrl")
	// 主页html
	fmt.Println("进入砖石会员页面，获取任务URL")
	resp := sendGet(cookieValue, indexUrl)
	defer resp.Body.Close()
	html, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Errorf("%s 请求错误", "首页页面")
	}
	//dir, _ := os.Getwd()
	//htmlPath := filepath.Join(dir, "test.html")
	//file, _ := os.Open(htmlPath)
	fmt.Println("解析任务页面第一页HTML构造")
	var nodes = TaskNode{}
	sortType := config.GetValue("sortType")
	if sortType == "2" {
		nodes = noSort(html, cookieValue)
	} else {
		nodes = priceSort(html, cookieValue)
	}

	fmt.Println("开始领取任务")
	submitTaskUrl := config.GetValue("submitTaskUrl")

	for taskIndex := range nodes {
		time.Sleep(700 * time.Millisecond)
		node := nodes[taskIndex]
		id := strings.Replace(node.Url, "https://cc.157nk.cn/index/index/show.html?id=", "", 1)
		fmt.Println(fmt.Sprintf("--第%d个领取任务成功--", node.No))
		submitIdStr, _ := reqTask(cookieValue, submitTaskUrl, id)
		if submitIdStr == "" {
			fmt.Println(fmt.Sprintf("--第%d个提交任务已领取，不需要再次领取--", node.No))
			continue
		}
	}

	fmt.Println("所有任务领取完毕")

}

func priceSort(html []byte, cookieValue string) TaskNode {
	distinctMap := make(map[string]string)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		log.Fatal(err)
	}

	find := doc.Find("ul.index-renw-ul").Find("li")

	find.Each(func(i int, selection *goquery.Selection) {

		herf, _ := selection.Find("a").Attr("href")
		text := selection.Find("div>p").Eq(1).Text()
		rune := []rune(text)
		key := fmt.Sprintf("https://cc.157nk.cn%s", herf)
		if _, ok := distinctMap[key]; !ok {
			distinctMap[key] = string(rune[1 : len(rune)-1])
		}
	})

	fmt.Println("解析任务页面第二页HTML构造")
	taskPageUrl := config.GetValue("taskPageUrl")
	taskList := reqTaskList(cookieValue, taskPageUrl, 2)

	for _, sigleTask := range taskList {
		m := sigleTask.(map[string]interface{})
		key := fmt.Sprintf("https://cc.157nk.cn/index/index/show.html?id=%v", m["id"])
		if _, ok := distinctMap[key]; !ok {
			distinctMap[key] = m["price"].(string)
		}
	}

	fmt.Println("解析任务页面第三页HTML构造")

	taskList = reqTaskList(cookieValue, taskPageUrl, 3)

	for _, sigleTask := range taskList {
		m := sigleTask.(map[string]interface{})
		key := fmt.Sprintf("https://cc.157nk.cn/index/index/show.html?id=%v", m["id"])
		//rune := []rune(m["title"].(string))
		if _, ok := distinctMap[key]; !ok {
			distinctMap[key] = m["price"].(string)
		}
	}

	fmt.Println("将解析到的任务进行去重，按编号从小到大排序")
	nodes := make(TaskNode, 0)

	for key, value := range distinctMap {
		nodes = append(nodes, Node{
			Price: value,
			Url:   key,
		})
	}
	sort.Sort(nodes)

	return nodes

}

func noSort(html []byte, cookieValue string) TaskNode {
	distinctMap := make(map[string]int)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(html)))
	if err != nil {
		log.Fatal(err)
	}

	find := doc.Find("ul.index-renw-ul").Find("li")

	find.Each(func(i int, selection *goquery.Selection) {

		herf, _ := selection.Find("a").Attr("href")
		textStr := selection.Find("div>p").Eq(0).Text()
		//text := selection.Find("div>p").Eq(1).Text()
		text := strings.Replace(textStr, "【点赞关注】", "", 1)
		value, _ := strconv.Atoi(text)
		if value == 0 {
			text = strings.Replace(textStr, "【点赞关注_】", "", 1)
			value, _ = strconv.Atoi(text)
		}
		//rune := []rune(text)
		key := fmt.Sprintf("https://cc.157nk.cn%s", herf)
		if _, ok := distinctMap[key]; !ok {
			distinctMap[key] = value
		}
	})

	fmt.Println("解析任务页面第二页HTML构造")
	taskPageUrl := config.GetValue("taskPageUrl")
	taskList := reqTaskList(cookieValue, taskPageUrl, 2)

	for _, sigleTask := range taskList {
		m := sigleTask.(map[string]interface{})
		key := fmt.Sprintf("https://cc.157nk.cn/index/index/show.html?id=%v", m["id"])
		text := strings.Replace(m["title"].(string), "【点赞关注】", "", 1)
		value, _ := strconv.Atoi(text)
		if value == 0 {
			text = strings.Replace(m["title"].(string), "【点赞关注_】", "", 1)
			value, _ = strconv.Atoi(text)
		}
		if _, ok := distinctMap[key]; !ok {
			distinctMap[key] = value
		}
	}

	fmt.Println("解析任务页面第三页HTML构造")

	taskList = reqTaskList(cookieValue, taskPageUrl, 3)

	for _, sigleTask := range taskList {
		m := sigleTask.(map[string]interface{})
		key := fmt.Sprintf("https://cc.157nk.cn/index/index/show.html?id=%v", m["id"])
		text := strings.Replace(m["title"].(string), "【点赞关注】", "", 1)
		value, _ := strconv.Atoi(text)
		if value == 0 {
			text = strings.Replace(m["title"].(string), "【点赞关注_】", "", 1)
			value, _ = strconv.Atoi(text)
		}

		//rune := []rune(m["title"].(string))
		if _, ok := distinctMap[key]; !ok {
			distinctMap[key] = value
		}
	}

	fmt.Println("将解析到的任务进行去重，按编号从小到大排序")
	nodes := make(TaskNode, 0)

	for key, value := range distinctMap {
		nodes = append(nodes, Node{
			No:  value,
			Url: key,
		})
	}
	sort.Sort(nodes)

	return nodes
}

func main() {

	account := config.GetValue("account")

	accounts := strings.Split(account, ",")

	for index := range accounts {
		fmt.Println(fmt.Sprintf("第%d账号", index+1))
		cookieValue := loginSuccess(accounts[index])

		//getTask(cookieValue)

		submitOkTaskUrl := config.GetValue("submitOkTaskUrl")
		time.Sleep(3 * time.Second)
		ids := readFileId()
		for taskIdIndex := range ids {
			taskIdStr := ids[taskIdIndex]
			time.Sleep(2 * time.Second)
			if taskIdStr == "" {
				fmt.Println("空任务不用管！")
				continue
			}
			submitTask(cookieValue, submitOkTaskUrl, taskIdStr)
			fmt.Println(fmt.Sprintf("--id: %s 第%d个提交任务成功--", taskIdStr, taskIdIndex))
		}
		clearFile()
	}

	fmt.Println("所有账号领取任务成功")

	//TODO 自动任务提交

}

func loginSuccess(account string) string {
	fmt.Println(fmt.Sprintf("--账号%s：登录网红社APP--", account))
	pwd := config.GetValue("password")
	loginUrl := config.GetValue("loginUrl")
	data := url.Values{}
	data.Set("username", account)
	data.Set("password", pwd)

	//登陆页html
	cookieValue, err := reqLogin(loginUrl, data)
	if err != nil {
		panic(fmt.Sprintf("%s 请求错误", "账号登录"))
	}

	fmt.Printf("拿到登录cookie：%s\n", cookieValue)

	return cookieValue
}

// 发送GET请求
// url:请求地址
// response:请求返回的内容
func Get(url string) string {
	client := http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	var buffer [512]byte
	result := bytes.NewBuffer(nil)
	for {
		n, err := resp.Body.Read(buffer[0:])
		result.Write(buffer[0:n])
		if err != nil && err == io.EOF {
			break
		} else if err != nil {
			panic(err)
		}
	}

	return result.String()
}

func sendGet(cookieStr string, url string) *http.Response {
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1")
	req.Header.Add("Host", "www.13jsz30.cn")
	req.Header.Add("Connection", "keep-alive")
	req.Header.Add("Upgrade-Insecure-Requests", "1")
	req.Header.Add("Referer", url)
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")

	if cookieStr != "" {
		req.Header.Add("Cookie", cookieStr)
	}

	if err != nil {
		panic(err)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}

func reqLogin(url string, data url.Values) (string, error) {

	client := &http.Client{}
	req, _ := http.NewRequest("POST", url, strings.NewReader(data.Encode())) // URL-encoded payload
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1")
	req.Header.Add("Referer", "https://cc.157nk.cn/index/login/index.html")
	req.Header.Add("Origin", "https://cc.157nk.cn")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	fmt.Println("post send success")

	fmt.Println("--登录网红社APP成功，开始解析cookie--")
	cookieStr := resp.Header.Get("Set-Cookie")
	cookieValue := strings.Split(cookieStr, ";")[0]
	println(cookieValue)

	return cookieValue, nil

}

// 请求任务首页
func reqIndex(url string, cookieStr string) *http.Response {
	fmt.Println("开始解析砖石任务首页")
	req, err := http.NewRequest("GET", url, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1")
	req.Header.Add(":authority", "cc.157nk.cn")
	req.Header.Add(":path", "/index/index/level_list.html?level_id=3")
	req.Header.Add(":scheme", "https")
	req.Header.Add("Referer", "https://cc.157nk.cn/index/member/vip.html")
	req.Header.Add("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,image/apng,*/*;q=0.8,application/signed-exchange;v=b3")

	if cookieStr != "" {
		req.Header.Add("Cookie", cookieStr)
	}

	if err != nil {
		panic(err)
	}
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	return resp
}

func reqTaskList(cookieStr string, link string, page int) []interface{} {

	data := url.Values{}
	data.Set("page", fmt.Sprintf("%d", page))
	data.Set("level_id", "3")

	client := &http.Client{}
	req, _ := http.NewRequest("POST", link, strings.NewReader(data.Encode())) // URL-encoded payload
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1")
	req.Header.Add("Referer", "https://cc.157nk.cn/index/index/level_list.html?level_id=3")
	req.Header.Add("Origin", "https://cc.157nk.cn")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	if cookieStr != "" {
		req.Header.Add("Cookie", cookieStr)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return nil
	}
	fmt.Println("post send success")

	s, _ := ioutil.ReadAll(resp.Body)

	var result map[string]interface{}
	json.Unmarshal(s, &result)
	dataMap := result["data"].(map[string]interface{})
	taskList := dataMap["list"].([]interface{})

	return taskList

}

func reqTask(cookieStr string, link string, id string) (string, error) {

	data := url.Values{}
	data.Set("id", id)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", link, strings.NewReader(data.Encode())) // URL-encoded payload
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1")
	req.Header.Add("Referer", fmt.Sprintf("https://cc.157nk.cn/index/index/show.html?id=%s", id))
	req.Header.Add("Origin", "https://cc.157nk.cn")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	if cookieStr != "" {
		req.Header.Add("Cookie", cookieStr)
	}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}
	s, _ := ioutil.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(s, &result)
	code := fmt.Sprintf("%v", result["code"])
	if code == "0" {
		return "", err
	}

	dataMap := result["data"].(map[string]interface{})
	idStr := dataMap["id"].(string)
	writeIdSave(idStr)
	return idStr, nil

}

func writeIdSave(idStr string) {
	dir, _ := os.Getwd()
	idSavePath := filepath.Join(dir, "id.txt")
	file, _ := os.OpenFile(idSavePath, os.O_APPEND|os.O_RDWR, 0755)
	io.WriteString(file, fmt.Sprintf("%s,", idStr))
}

func clearFile() {
	dir, _ := os.Getwd()
	idSavePath := filepath.Join(dir, "id.txt")
	file, _ := os.OpenFile(idSavePath, os.O_TRUNC|os.O_RDWR, 0755)
	io.WriteString(file, "")
}

func readFileId() []string {
	dir, _ := os.Getwd()
	idSavePath := filepath.Join(dir, "id.txt")
	file, _ := os.OpenFile(idSavePath, os.O_RDWR, 0755)
	allId, e := ioutil.ReadAll(file)
	if e != nil {
		panic("读取ID文件错误")
	}
	if len(allId) == 0 {
		fmt.Println("暂时没有文件数据读取")
	}
	idStr := string(allId)
	return strings.Split(idStr, ",")
}

func submitTask(cookieStr string, link string, id string) (string, error) {

	data := url.Values{}
	data.Set("id", id)

	client := &http.Client{}
	req, _ := http.NewRequest("POST", link, strings.NewReader(data.Encode())) // URL-encoded payload
	req.Header.Add("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 11_0 like Mac OS X) AppleWebKit/604.1.38 (KHTML, like Gecko) Version/11.0 Mobile/15A372 Safari/604.1")
	req.Header.Add("Referer", fmt.Sprintf("https://cc.157nk.cn/index/index/submittask.html?id=%s", id))
	req.Header.Add("Origin", "https://cc.157nk.cn")
	req.Header.Add("Sec-Fetch-Mode", "cors")
	req.Header.Add("sec-fetch-site", "same-origin")
	//req.Header.Add(":authority", "cc.157nk.cn")
	req.Header.Add("content-length", "9")
	//req.Header.Add(":scheme", "https")
	//req.Header.Add(":path", "/index/index/submittask.html")
	req.Header.Add("X-Requested-With", "XMLHttpRequest")
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	if cookieStr != "" {
		req.Header.Add("Cookie", cookieStr)
	}
	_, err := client.Do(req)
	if err != nil {
		fmt.Println(err.Error())
		return "", err
	}

	return "", nil

}
