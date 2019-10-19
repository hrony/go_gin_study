package lib

import (
	"bytes"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	dlog "go_gin_study/lesson/Gin入门实战/common/log"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var TimeLocation *time.Location
var (
	TimeFormat = "2006-01-02 15:04:05"
	DateFormat = "2006-01-02"
	LocalIP    = net.ParseIP("127.0.0.1")
)

func Init(configPath string) error {
	return InitModule(configPath, []string{
		"base",
		"mysql",
		"redis",
	})
}

func InitModule(configPath string, modules []string) error {
	var conf *string
	if len(configPath) > 0 {
		conf = &configPath
	} else {
		conf = flag.String("config", "", "input config file like ./conf/dev/")
		flag.Parse()
	}
	if *conf == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.Println("------------------------------------------------------------------------")
	log.Printf("[INFO]  config=%s\n", *conf)
	log.Printf("[INFO] %s\n", " start loading resources.")

	// 设置ip信息，优先设置便于日志打印
	ips := GetLocalIPs()
	if len(ips) > 0 {
		LocalIP = ips[0]
	}

	// 解析配置文件目录
	if err := ParseConfPath(*conf); err != nil {
		return err
	}

	// 初始化配置文件
	if err := InitViperConf(); err != nil {
		return err
	}

	// 加载base配置
	if InArrayString("base", modules) {
		if err := InitBaseConf(GetConfPath("base")); err != nil {
			fmt.Printf("[ERROR] %s%s\n", time.Now().Format(TimeFormat), " InitBaseConf:"+err.Error())
		}
	}

	// 加载redis配置
	if InArrayString("redis", modules) {
		if err := InitRedisConf(GetConfPath("redis")); err != nil {
			fmt.Printf("[ERROR] %s%s\n", time.Now().Format(TimeFormat), " InitRedisConf:"+err.Error())
		}
	}

	// 加载mysql配置并初始化实例
	if InArrayString("mysql", modules) {
		if err := InitDBPool(GetConfPath("mysql")); err != nil {
			fmt.Printf("[ERROR] %s%s\n", time.Now().Format(TimeFormat), " InitDBPool:"+err.Error())
		}
	}

	// 设置时区
	if location, err := time.LoadLocation(ConfBase.TimeLocation); err != nil {
		return err
	} else {
		TimeLocation = location
	}

	log.Printf("[INFO] %s\n", " success loading resources.")
	log.Println("------------------------------------------------------------------------")
	return nil
}

// 公共销毁函数
func Destroy() {
	log.Println("------------------------------------------------------------------------")
	log.Printf("[INFO] %s\n", " start destroy resources.")
	CloseDB()
	dlog.Close()
	log.Printf("[INFO] %s\n", " success destroy resources.")
}

func HTTPGET(trace *TraceContext, urlString string, urlParams url.Values, msTimeout int, header http.Header) (*http.Response, []byte, error) {
	startTime := time.Now().UnixNano()
	client := http.Client{
		Timeout: time.Duration(msTimeout) * time.Millisecond,
	}
	urlString = AddGetDataToURL(urlString, urlParams)
	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		Log.TagWarn(trace, DLTagHTTPFailed, map[string]interface{}{
			"url":       urlString,
			"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
			"method":    "GET",
			"args":      urlParams,
			"err":       err.Error(),
		})
		return nil, nil, err
	}
	if len(header) > 0 {
		req.Header = header
	}
	req = addTrace2Header(req, trace)
	resp, err := client.Do(req)
	if err != nil {
		Log.TagWarn(trace, DLTagHTTPFailed, map[string]interface{}{
			"url":       urlString,
			"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
			"method":    "GET",
			"args":      urlParams,
			"err":       err.Error(),
		})
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.TagWarn(trace, DLTagHTTPFailed, map[string]interface{}{
			"url":       urlString,
			"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
			"method":    "GET",
			"args":      urlParams,
			"result":    string(body),
			"err":       err.Error(),
		})
		return nil, nil, err
	}
	Log.TagInfo(trace, DLTagHTTPSuccess, map[string]interface{}{
		"url":       urlString,
		"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
		"method":    "GET",
		"args":      urlParams,
		"result":    string(body),
	})
	return resp, body, nil
}

func HTTPPOST(trace *TraceContext, urlString string, urlParams url.Values, msTimeout int, header http.Header, contextType string) (*http.Response, []byte, error) {
	startTime := time.Now().UnixNano()
	client := http.Client{
		Timeout: time.Duration(msTimeout) * time.Millisecond,
	}
	if contextType == "" {
		contextType = "application/x-www-form-urlencoded"
	}
	req, err := http.NewRequest("POST", urlString, strings.NewReader(urlParams.Encode()))
	if len(header) > 0 {
		req.Header = header
	}
	req = addTrace2Header(req, trace)
	req.Header.Set("Content-Type", contextType)
	resp, err := client.Do(req)
	if err != nil {
		Log.TagWarn(trace, DLTagHTTPFailed, map[string]interface{}{
			"url":       urlString,
			"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
			"method":    "POST",
			"args":      urlParams,
			"err":       err.Error(),
		})
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.TagWarn(trace, DLTagHTTPFailed, map[string]interface{}{
			"url":       urlString,
			"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
			"method":    "POST",
			"args":      urlParams,
			"result":    string(body),
			"err":       err.Error(),
		})
		return nil, nil, err
	}
	Log.TagInfo(trace, DLTagHTTPSuccess, map[string]interface{}{
		"url":       urlString,
		"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
		"method":    "POST",
		"args":      urlParams,
		"result":    string(body),
	})
	return resp, body, nil
}

func HTTPJSON(trace *TraceContext, urlString string, jsonContent string, msTimeout int, header http.Header) (*http.Response, []byte, error) {
	startTime := time.Now().UnixNano()
	client := http.Client{
		Timeout: time.Duration(msTimeout) * time.Millisecond,
	}
	req, err := http.NewRequest("POST", urlString, strings.NewReader(jsonContent))
	if len(header) > 0 {
		req.Header = header
	}
	req = addTrace2Header(req, trace)
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		Log.TagWarn(trace, DLTagHTTPFailed, map[string]interface{}{
			"url":       urlString,
			"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
			"method":    "POST",
			"args":      jsonContent,
			"err":       err.Error(),
		})
		return nil, nil, err
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		Log.TagWarn(trace, DLTagHTTPFailed, map[string]interface{}{
			"url":       urlString,
			"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
			"method":    "POST",
			"args":      jsonContent,
			"result":    string(body),
			"err":       err.Error(),
		})
		return nil, nil, err
	}
	Log.TagInfo(trace, DLTagHTTPSuccess, map[string]interface{}{
		"url":       urlString,
		"proc_time": float32(time.Now().UnixNano()-startTime) / 1.0e9,
		"method":    "POST",
		"args":      jsonContent,
		"result":    string(body),
	})
	return resp, body, nil
}

func AddGetDataToURL(urlString string, data url.Values) string {
	if strings.Contains(urlString, "?") {
		urlString = urlString + "&"
	} else {
		urlString = urlString + "?"
	}
	return fmt.Sprintf("%s%s", urlString, data.Encode())
}

func addTrace2Header(request *http.Request, trace *TraceContext) *http.Request {
	traceID := trace.TraceID
	cSpanID := NewSpanID()
	if traceID != "" {
		request.Header.Set("header-rid", traceID)
	}
	if cSpanID != "" {
		request.Header.Set("header-spanid", cSpanID)
	}
	trace.CSpanID = cSpanID
	return request
}

func GetMd5Hash(text string) string {
	hasher := md5.New()
	hasher.Write([]byte(text))
	return hex.EncodeToString(hasher.Sum(nil))
}

func Encode(data string) (string, error) {
	h := md5.New()
	_, err := h.Write([]byte(data))
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ParseServerAddr(serverAddr string) (host, port string) {
	serverInfo := strings.Split(serverAddr, ":")
	if len(serverInfo) == 2 {
		host = serverInfo[0]
		port = serverInfo[1]
	} else {
		host = serverAddr
		port = ""
	}
	return host, port
}

func NewTrace() *TraceContext {
	trace := &TraceContext{}
	trace.TraceID = GetTraceID()
	trace.SpanID = NewSpanID()
	return trace
}

func NewSpanID() string {
	timestamp := uint32(time.Now().Unix())
	ipToLong := binary.BigEndian.Uint32(LocalIP.To4())
	b := bytes.Buffer{}
	b.WriteString(fmt.Sprintf("%08x", ipToLong^timestamp))
	b.WriteString(fmt.Sprintf("%08x", rand.Int31()))
	return b.String()
}

func GetTraceID() (traceID string) {
	return calcTraceID(LocalIP.String())
}

func calcTraceID(ip string) (traceID string) {
	now := time.Now()
	timestamp := uint32(now.Unix())
	timeNano := now.UnixNano()
	pid := os.Getpid()

	b := bytes.Buffer{}
	netIP := net.ParseIP(ip)
	if netIP == nil {
		b.WriteString("00000000")
	} else {
		b.WriteString(hex.EncodeToString(netIP.To4()))
	}
	b.WriteString(fmt.Sprintf("%08x", timestamp&0xffffffff))
	b.WriteString(fmt.Sprintf("%04x", timeNano&0xffff))
	b.WriteString(fmt.Sprintf("%04x", pid&0xffff))
	b.WriteString(fmt.Sprintf("%06x", rand.Int31n(1<<24)))
	b.WriteString("b0") // 末两位标记来源,b0为go

	return b.String()
}

func GetLocalIPs() (ips []net.IP) {
	interfaceAddr, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}
	for _, address := range interfaceAddr {
		ipNet, isValidIPNet := address.(*net.IPNet)
		if isValidIPNet && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				ips = append(ips, ipNet.IP)
			}
		}
	}
	return ips
}

func InArrayString(s string, arr []string) bool {
	for _, i := range arr {
		if i == s {
			return true
		}
	}
	return false
}
