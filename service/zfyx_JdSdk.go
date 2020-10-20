package service

import (
	"fmt"
	"time"
	"strings"
	"reflect"
	"sort"
	"net/url"
	"github.com/qiusnay/3dorderquery/util"
	"github.com/qiusnay/3dorderquery/model"
	"encoding/json"
	"github.com/google/logger"
	// "github.com/jinzhu/gorm"
)

type OrderParam struct {
	OrderReq OrderReq `json:"orderReq"`
}

type OrderReq struct {
	PageNo       int    `json:"pageNo"`                 //页码，返回第几页结果
	PageSize     int    `json:"pageSize"`               //每页包含条数，上限为500
	Type         int    `json:"type"`                   //订单时间查询类型(1：下单时间，2：完成时间，3：更新时间)
	StartTime         string `json:"startTime"`         //查询时间，建议使用分钟级查询，格式：yyyyMMddHH、yyyyMMddHHmm或yyyyMMddHHmmss，如201811031212 的查询范围从12:12:00--12:12:59
	EndTime         string `json:"endTime"`             //查询时间，建议使用分钟级查询，格式：yyyyMMddHH、yyyyMMddHHmm或yyyyMMddHHmmss，如201811031212 的查询范围从12:12:00--12:12:59
	ChildUnionId int64  `json:"childUnionId,omitempty"` //子站长ID（需要联系运营开通PID账户权限才能拿到数据），childUnionId和key不能同时传入
	Key          string `json:"key,omitempty"`          //其他推客的授权key，查询工具商订单需要填写此项，childUnionid和key不能同时传入
}

type JdUnionOpenOrderRowQueryResponse struct {
	JdUnionOpenOrderRowQueryResponse struct {
		Result string `json:"result"`
		Code   string `json:"code"`
	} `json:"jd_union_open_order_row_query_response"`
}

type SysParam struct {
	Method       string `json:"method"`
	App_key      string `json:"app_key"`
	Access_token string `json:"access_token"`
	Timestamp    string `json:"timestamp"`
	Format       string `json:"format"`
	V            string `json:"v"`
	Sign_method  string `json:"sign_method"`
	Param_json   string `json:"param_json"`
}

type Jdsdk struct {
	RequestParam SysParam
	SignAndUri   string
	RequestRul   string
}

func (J *Jdsdk) GetParams(start string, end string) string {
	ParamStruct := OrderParam{}
	ParamStruct.OrderReq.StartTime = start
	ParamStruct.OrderReq.EndTime = end
	ParamStruct.OrderReq.PageNo = 1
	ParamStruct.OrderReq.PageSize = 10
	ParamStruct.OrderReq.Type = 1
	bytes, _ := json.Marshal(ParamStruct)
	return string(bytes)
}

//获取订单
func (J *Jdsdk) GetOrders(start string, end string) OrderResult {
	Param := J.GetParams(start, end)
	J.SetSignJointUrlParam(Param)
	var urls strings.Builder
	urls.WriteString(util.JD_HOST)
	urls.WriteString(J.SignAndUri)
	body, _ := util.HttpGet(urls.String())
	response := &JdUnionOpenOrderRowQueryResponse{}
	e := json.Unmarshal([]byte(body), &response)
	
	if e != nil {
		panic(e)
	}
	logger.Info(fmt.Sprintf("get jd order %+v", string(response.JdUnionOpenOrderRowQueryResponse.Result)))
	result := OrderResult{}
	e = json.Unmarshal([]byte(response.JdUnionOpenOrderRowQueryResponse.Result), &result)
	if e != nil {
		panic(e)
	}
	for _, order := range result.Data {
		model.DB.Table("tb_jd_original_order").Create(&order)
	}
	return result
}

type Apiconfig struct {
	Appkey   string
	APPsecret   string
	Method    string
}
type config struct {
	Jd Apiconfig   `toml:"jd"`
}
var conf config
//生成请求参数和签名
func (J *Jdsdk) SetSignJointUrlParam(paramjson string) *Jdsdk {
	util.Config().Bind("conf", "thirdpartysdk", &conf)
	// logger.Info(fmt.Sprintf("ddddddd %+v", conf.Jd.Method))

	J.RequestParam.App_key = conf.Jd.Appkey
	J.RequestParam.Format  = "json"
	J.RequestParam.V       = "1.0"
	J.RequestParam.Method = conf.Jd.Method

	J.RequestParam.Sign_method = "md5"
	J.RequestParam.Timestamp = time.Now().Format("2006-01-02 15:04:05")
	J.RequestParam.Param_json = paramjson
	
	values := reflect.ValueOf(J.RequestParam)
	keys := reflect.TypeOf(J.RequestParam)
	count := values.NumField()
	SortSlice := util.Items{}
	for i := 0; i < count; i++ {
		value := values.Field(i)
		key := keys.Field(i)
		switch value.Kind() {
		case reflect.String:
			SortSlice = append(SortSlice, util.Onestruct{strings.ToLower(key.Name), value.String()})
		case reflect.Int:
			SortSlice = append(SortSlice, util.Onestruct{strings.ToLower(key.Name), value.String()})
		}
	}
	sort.Sort(SortSlice)
	
	var builder strings.Builder
	u := url.Values{}
	builder.WriteString(conf.Jd.APPsecret)
	for _, person := range SortSlice {
		if person.Value == "" {
			continue
		}
		builder.WriteString(strings.ToLower(person.Key) + person.Value)
		u.Add(strings.ToLower(person.Key), person.Value)
	}
	builder.WriteString(conf.Jd.APPsecret)

	//生成签名
	u.Add("sign", strings.ToUpper(util.Md5(builder.String())))
	//拼接参数
	J.SignAndUri = u.Encode()
	return J
}

