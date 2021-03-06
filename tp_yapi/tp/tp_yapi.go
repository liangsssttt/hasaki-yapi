package tp

import (
	"fmt"
	"github.com/liangsssttt/hasaki-util/string_util"
	"github.com/liangsssttt/hasaki-yapi/yapi"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

/**
 * 获取门店下的活动列表
 * @path /home/Activity/listActivity.json
 * @throws \think\db\exception\DataNotFoundException
 * @throws \think\db\exception\ModelNotFoundException
 * @throws \think\exception\DbException
 */

/**
 * @api 前台的活动的模块
 * Class Activity
 * @package app\home\controller
 */

// 仅对自动路由匹配
// 扫描规则
// 控制器类的方法声明上@api 表示属于某个分类
// 方法上的注释表示接口路径 @path
// 方法上的@param表示参数 @param
// 目录结构
/**

-- app
	-- home
		-- controller
			-- Index		/home/Index/xxx方法
*/

type TpYapi struct {
	ScanDir    string `comment:"扫描的目录"`
	PathSuffix string `comment:"路径后缀比如.json .html"`
	FileSuffix string `comment:"扫描的文件后缀 .class.php  .php"`
	Controller string `comment:"扫描的文件目录"`
}

type Config struct {
	Tp   *TpYapi
	Yapi *yapi.Yapi
}

var (
	c     *Config
	tp    *TpYapi
	once  sync.Once
	tpApi *TpApiCollect
)

func InitTpYapi() *Config {
	once.Do(func() {
		c = readConfig()
		tp = c.Tp
	})
	return c
}

func readConfig() *Config {
	args := os.Args[1:]
	var configPath string
	if len(args) >= 2 && (args[0] == "-c" || args[0] == "-C") {
		configPath = args[1]
		if filepath.Ext(configPath) != "yaml" {
			log.Fatal("配置文件必须是yaml格式！")
		}
	}
	if configPath == "" {
		executePath := os.Args[0]
		configPath = string_util.SystemAppend(false, filepath.Dir(executePath), "yapi.yaml")
	}
	yamlFile, err := ioutil.ReadFile(configPath)
	if err != nil {
		log.Fatalf("配置文件：%s不存在！", configPath)
	}
	c := &Config{}
	err = yaml.Unmarshal(yamlFile, c)
	if c.Yapi == nil || c.Tp == nil {

		log.Fatal(`配置文件不准确:例子如下
tp:
  scandir: F:\OneDrive\huishi-shop\application
  pathsuffix: .json
  filesuffix: .class.php
  controller: controller
yapi:
  token: 5af10be82269b7f3dfae11ad76f086fd0909223962ebbda6101
  projectid: 23
  host: http://api.example.net
`)
	}
	if err != nil {
		log.Fatalf("配置文件解析失败：%s", err)
	}
	return c
}

type TpApiCollect struct {
	TpApi        map[string]map[string]*yapi.Api `json:"tp_api" comment:"搜集的某个分类下的api接口"`
	FailureTpApi map[string]map[string]*yapi.Api `json:"failure_tp_api" comment:"成功的"`
	SuccessTpApi map[string]map[string]*yapi.Api `json:"success_tp_api" comment:"失败的"`
}

func (t *TpYapi) Scan() *TpApiCollect {
	tpApi = &TpApiCollect{
		TpApi:        make(map[string]map[string]*yapi.Api),
		FailureTpApi: make(map[string]map[string]*yapi.Api),
		SuccessTpApi: make(map[string]map[string]*yapi.Api),
	}
	_ = filepath.Walk(t.ScanDir, t.walk)

	return tpApi
}

func (t *TpYapi) walk(filename string, fi os.FileInfo, err error) error {
	if strings.Contains(filepath.Dir(filename), t.Controller) {
		// 获取目标文件的方法
		b, _ := ioutil.ReadFile(filename)
		// 获取@api
		re := regexp.MustCompile("@api[^\r\n]+")
		cateName := string(re.Find(b))
		if cateName == "" {
			return nil
		}
		cateName = string_util.Replace(cateName, "", "@api", " ")
		modules := string_util.SystemSplit(strings.Replace(filename, string_util.SystemAppend(true, t.Controller, fi.Name()), "", -1))
		module := modules[len(modules)-1]
		control := string_util.Replace(fi.Name(), "", t.FileSuffix, ".php", ".class.php", "controller", "Controller")
		dealFunction([]byte(strings.Split(string(b), cateName)[1]), cateName, "/"+module+"/"+control, t.PathSuffix)
	}
	return nil
}

func dealFunction(b []byte, cateName, path, suffix string) {
	re := regexp.MustCompile("\\s+\\/\\*\\*[^`]*?\\)")
	bs := re.FindAll(b, -1)

	for _, v := range bs {
		if string_util.Contains(string(v), false, "namespace", "class", "@api") || (!string_util.Contains(string(v), true, "public", "function")) {
			continue
		}
		reName := regexp.MustCompile(`function\s*?([^-~\s]*?)\(`)
		name := string_util.Replace(string(reName.Find(v)), "", "function", " ", "(")
		reTitle := regexp.MustCompile("[^/\\*\n(\\s+)]+[\\s]*?[^/*\n(\\s+)]+")
		title := string(reTitle.Find(v))
		fields := dealFiled(v)
		tempApi := &yapi.Api{
			Title:               title,
			Path:                path + "/" + name + suffix,
			CatId:               0,
			Status:              yapi.Done,
			Method:              yapi.Post,
			ReqBodyIsJsonSchema: false,
			ResBodyIsJsonSchema: false,
			Desc:                string(v),
			Markdown:            string(v),
			ReqBodyOther:        "",
			ReqBodyType:         "",
			ReqQuery:            nil,
			ReqBodyForm:         fields,
			ReqHeaders: yapi.GetHeaders(map[string]yapi.HeaderType{
				"Content-Type": yapi.XWwwFormUrlencoded,
			}),
			ResBody:     "",
			ResBodyType: "",
		}
		if len(fields) > 0 {
			tempApi.ReqBodyType = yapi.ReqForm
		}

		if _, ex := tpApi.TpApi[cateName]; !ex {
			tpApi.TpApi[cateName] = make(map[string]*yapi.Api)
			if _, exx := tpApi.TpApi[cateName][path+"/"+name+suffix]; !exx {
				tpApi.TpApi[cateName][path+"/"+name+suffix] = tempApi
			}
		} else {
			tpApi.TpApi[cateName][path+"/"+name+suffix] = tempApi
		}

	}
}

// 获取字段信息
func dealFiled(b []byte) []*yapi.Field {
	fs := make([]*yapi.Field, 0)
	// 获取字段
	reFiled, _ := regexp.Compile("\\(([^`]*?)\\)")
	fieldsStr := reFiled.Find(b)
	if !string_util.Contains(string(fieldsStr), true, "$") {
		return nil
	}
	fields := strings.Split(string_util.Replace(string(fieldsStr), "", "int ", " int", " int ", " string", "string ", " string ", "array ", " array", " array ", "(", ")", "\r\n", " "), ",")
	for _, v := range fields {
		if !string_util.Contains(v, true, "$") {
			continue
		}
		f := &yapi.Field{}
		tempS := strings.Split(v, "=")
		if len(tempS) == 1 {
			f.Name = string_util.Replace(tempS[0], "", "$")
			f.Example = ""
			f.Required = 1
		} else if len(tempS) == 2 {
			f.Name = string_util.Replace(tempS[0], "", "$")
			f.Example = tempS[1]
		}
		if f.Name == "" {
			continue
		}
		f.Type = yapi.Text
		// 获取中文注释
		reDesc, _ := regexp.Compile(fmt.Sprintf("@param[^(\r\n)]+\\$%s[^(\r\n)]+", f.Name))
		f.Desc = ""
		if string(reDesc.Find(b)) != "" {
			descs := strings.Split(string(reDesc.Find(b)), "$"+f.Name)
			if len(descs) == 2 {
				f.Desc = descs[1]
			}
		}
		fs = append(fs, f)
	}
	return fs
}
