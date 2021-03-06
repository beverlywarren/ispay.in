package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"rowei/mvip/common/utils"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	XForwardedFor = "X-Forwarded-For"
	XRealIP       = "X-Real-IP"
	Userid        = "10004"                            //10004
	SignKey       = "c775a071726677eac6cd88e9aad182d2" //商户签名
)

type businessModel struct {
	Name string
}

func main() {
	r := gin.Default()

	r.LoadHTMLGlob("templates/*")
	r.Static("/js", "./js")
	r.Static("/qrcode", "./qrcode")
	r.Static("/css", "./css")
	r.Static("/img", "./img")

	r.GET("/", func(c *gin.Context) {
		// go func() {
		// 	i := 1
		// 	for {
		// 		if i > 4 {
		// 			break
		// 		}
		// 		t := time.NewTimer(time.Second * 5)
		// 		<-t.C
		// 		fmt.Println(i)
		// 		i++
		// 	}
		// 	fmt.Println("stop")
		// }()

		c.HTML(http.StatusOK, "index.html", gin.H{})
	})

	r.GET("/alipay", func(c *gin.Context) {
		c.HTML(http.StatusOK, "alipay.html", gin.H{})
	})

	r.GET("/alipay2", func(c *gin.Context) {
		//alipays://platformapi/startapp?appId=10000011&url=http%3a%2f%2fjstash.best%2fali.html
		//c.HTML(http.StatusOK, "alipay2.html", gin.H{})
		c.Redirect(301, "alipays://platformapi/startapp?appId=10000011&url=http%3a%2f%2fjstash.best%2fali.html")
	})

	r.GET("/902", func(c *gin.Context) {
		c.HTML(http.StatusOK, "902.html", gin.H{})
	})

	r.GET("/loading", func(c *gin.Context) {
		c.HTML(http.StatusOK, "loading.html", gin.H{})
	})

	//提交订单
	r.POST("/gateway", formdata)

	//二维码受限错误
	r.POST("/order/error", func(c *gin.Context) {
		order_id := c.PostForm("order_id")
		//moblie := c.PostForm("moblie")
		//content := c.PostForm("content")

		var r http.Request
		r.ParseForm()
		r.Form.Add("userid", "10004")
		r.Form.Add("order_id", order_id)
		r.Form.Add("moblie", "13989898989")
		r.Form.Add("content", "二维码无法支付")
		r.Form.Add("ip", c.ClientIP())

		bodystr := strings.TrimSpace(r.Form.Encode())
		body2, err := Post("https://pay.ispay.in/order/error", bodystr)

		if err == nil {
			c.Header("Content-Type", "application/json")
			c.String(http.StatusOK, string(body2))
			//fmt.Println(body2)
		} else {
			c.JSON(http.StatusOK, gin.H{"success": false, "errorCode": 0, "errorMsg": "提交错误失败，请联系客服"})
		}

	})

	//回调执行
	r.GET("/url/return", func(c *gin.Context) {
		order_no := c.Query("order_no")
		c.String(http.StatusOK, fmt.Sprintf("订单已经支付成功：%v", order_no))

	})

	// 通知
	r.GET("/url/notify", func(c *gin.Context) {

		order_no := c.Query("order_no")
		sign := c.Query("sign")

		return_sign := utils.MD5("10004" + order_no + SignKey)

		// 对比签名是否等于
		if sign == return_sign {
			c.String(http.StatusOK, "success")

		} else {
			c.String(http.StatusOK, "fail")
		}

	})

	//查询订单状态
	r.GET("/query/:trade_no", func(c *gin.Context) {

		// 支付状态说明
		// {"1", "订单提交，待出码"},
		// {"2", "已出码"},
		// {"3", "用户取消"},
		// {"4", "超时未接单系统取消"},
		// {"5", "等待打款"},
		// {"6", "超时未打款取消"},
		// {"7", "支付受限,支付关闭"},
		// {"8", "发单确认打款"},
		// {"9", "订单已完成"},

		trade_no := c.Param("trade_no")

		//fmt.Println(trade_no)
		if body, err := Get("https://pay.ispay.in/query/" + trade_no); err == nil {
			// 再设置个json
			c.Header("Content-Type", "application/json")
			c.String(http.StatusOK, string(body))
		} else {
			c.JSON(http.StatusOK, gin.H{"success": false, "errorCode": 0, "errorMsg": err})
			//fmt.Println(err)
		}
	})

	r.Run(fmt.Sprintf(":%d", 7005))
}

type refjson struct {
	Success   bool   `json:"success"`
	ErrorCode int    `json:"errorCode"`
	ErrorMsg  string `json:"errorMsg"`
	Orderid   uint   `json:"order_id"`
}

// 提交订单信息
func formdata(c *gin.Context) {

	order_type := c.PostForm("order_type")
	subject := c.PostForm("subject")
	return_url := c.PostForm("return_url")
	notify_url := c.PostForm("notify_url")
	ip := c.ClientIP()

	//userided, err := base64.URLEncoding.DecodeString(userid)

	// 测试的签名地址

	urls := "http://127.0.0.1:7003/gateway"

	amount, err := strconv.ParseInt(c.PostForm("amount"), 10, 64)
	if err != nil {
		c.JSON(200, gin.H{
			"success":  false,
			"errorMsg": "金额错误!",
		})
		return
	}

	order_no := time.Now().Format("20060102150405")
	order_time := time.Now()

	var r http.Request
	r.ParseForm()
	r.Form.Add("order_no", order_no)
	r.Form.Add("userid", Userid)
	r.Form.Add("subject", subject)
	r.Form.Add("order_type", order_type)
	r.Form.Add("amount", FormatPrice(amount))
	r.Form.Add("return_url", return_url)
	r.Form.Add("notify_url", notify_url)
	r.Form.Add("order_time", fmt.Sprintf("%d", order_time.Unix()))
	r.Form.Add("ip", ip)
	r.Form.Add("sign", MD5(Userid+order_no+order_type+FormatPrice(amount)+return_url+SignKey))

	// fmt.Println(Userid + order_no + order_type + FormatPrice(amount) + return_url + SignKey)
	// fmt.Println(MD5(Userid + order_no + order_type + FormatPrice(amount) + return_url + SignKey))

	bodystr := strings.TrimSpace(r.Form.Encode())

	body2, err := Post(urls, bodystr)

	if err != nil {
		c.JSON(200, gin.H{
			"success":   false,
			"errorCode": 1010,
			"errorMsg":  fmt.Sprint("Error:", err.Error()),
		})
		return
	}

	fmt.Println(string(body2))
	//返回JSON实例化对象
	var ref refjson
	if err := json.Unmarshal(body2, &ref); err == nil {

		if ref.ErrorCode != 0 {
			c.JSON(200, gin.H{
				"success":   false,
				"errorCode": ref.ErrorCode,
				"errorMsg":  ref.ErrorMsg,
			})
		} else {
			c.JSON(200, gin.H{
				"success":   true,
				"errorCode": ref.ErrorCode,
				"errorMsg":  ref.ErrorMsg,
				"order_id":  ref.Orderid,
			})

		}

	} else {
		c.JSON(200, gin.H{
			"success":   false,
			"errorCode": "0000",
			"errorMsg":  "解析Json数据失败:" + err.Error(),
		})
	}

}

// 获取IP
func GetRemoteIP(req *http.Request) string {
	remoteAddr := req.RemoteAddr
	if ip := req.Header.Get(XRealIP); ip != "" {
		remoteAddr = ip
	} else if ip = req.Header.Get(XForwardedFor); ip != "" {
		remoteAddr = ip
	} else {
		remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
	}

	if remoteAddr == "::1" {
		remoteAddr = "127.0.0.1"
	}

	return remoteAddr
}

// 标准化价格￥0.00
func FormatPrice(price interface{}) string {
	switch price.(type) {
	case float32, float64:
		return fmt.Sprintf("%0.2f", price)
	case int, uint, int32, int64, uint32, uint64:
		return fmt.Sprintf("%d.00", price)
	}
	return ""
}

// MD5加密哈希
func MD5(str string) string {
	h := md5.New()
	h.Write([]byte(str))
	return hex.EncodeToString(h.Sum(nil))
}

func Post(urls string, post_data string) ([]byte, error) {

	request, err := http.NewRequest("POST", urls, strings.NewReader(post_data))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	request.Header.Set("Pragma", "no-cache")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.181 Mobile Safari/537.36")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")

	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	// 保证I/O正常关闭
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

//获取流媒体
func Get(urls string) ([]byte, error) {

	request, err := http.NewRequest("GET", urls, nil)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
	request.Header.Set("Host", "pay.ispay.in")
	request.Header.Set("Pragma", "no-cache")
	request.Header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/65.0.3325.181 Mobile Safari/537.36")
	request.Header.Set("X-Requested-With", "XMLHttpRequest")

	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	// 保证I/O正常关闭
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}
