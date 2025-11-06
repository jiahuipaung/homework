package hundsun

import "fmt"

/* 提取结果样例, date字段是time.Time类型, 其他字段都是string
{
    "agent_hostname": "tt-fc-es01.nj",
    "data": "fund_account|futu_entrust_price|entrust_bs|futu_code|entrust_amount|entrust_time|entrust_no|subcribe_type|trade_server|function_id|\n10000666|87660.000|2|cu2210|1|10:57:23|       76979|112|CTP|112|",
    "date": "2023-03-29T10:14:48.28+08:00",
    "func_no": "112",
    "fund_account": "10000666",
    "req_no": "1661987632",
    "tag": "13:8|4:111|5:112|6:2006041512|11:0|3:1|12:0|7:0|69:4#UTF-8markid#dabd06ed4d894a9388b12e1d2d442bad|10:[]|connection_no:1034944523|",
    "type": "response",
    "error_no": "0", // 可能不存在
	"error_info": "xxx", // 可能不存在
}
*/

const (
	LogKeyDate          = "date"           // 必须
	LogKeyAgentHostname = "agent_hostname" // 必须
	LogKeyReqNo         = "req_no"         // 必须
	LogKeyFuncNo        = "func_no"        // 必须
	LogKeyFundAccount   = "fund_account"   // 非必须
	LogKeyType          = "type"           // 必须
	LogKeyTag           = "tag"            // 非必须
	LogKeyData          = "data"           // 非必须
	LogKeyErrorNo       = "error_no"       // 非必须
	LogKeyErrorInfo     = "error_info"     // 非必须
	LogKeyRequestTime   = "request_time"   // 请求耗时, 单位毫秒
)

const (
	LogTypeRequest  = "request"
	LogTypeResponse = "response"
)

const (
	LogKeyAccountContent = "account_content" // 28017的fund_account 的替换值
)

var (
	ErrorExtractFieldMissingFn = func(field string) error {
		return fmt.Errorf("field[%v] missing", field)
	}
	ErrorExtractFieldTypeNotMatchFn = func(field string) error {
		return fmt.Errorf("field[%v] type not match", field)
	}
)

var (
	musts = []string{LogKeyDate, LogKeyAgentHostname,
		LogKeyReqNo, LogKeyFuncNo, LogKeyType, LogKeyTag, LogKeyData}
	shoulds = []string{LogKeyFundAccount, LogKeyErrorNo, LogKeyErrorInfo}
)

// categraf采集多行日志时, 回车字符'\n' 被变成了 '\\\\n', 需要做适配处理
// categraf新版已经去掉了对'\n'的转义
var (
	settings = `
	{
		"pre_function": [
			{
				"mode": "contains",
				"format": "agent_hostname"
			},
			{
				"mode": "contains",
				"format": "CLIENT_REQ"
			}
		],
		"prefix_match": [
			"message",
			"agent_hostname"
		],
		"pre_extract": [
			{
				"mode": "split",
				"field": "message",
				"format": "{\"delimiter\":\"\\n\",\"num_of_substring\":7}"
			},
			{
				"mode": "regexp",
				"field": "message.0",
				"format": "^(?P\u003cdate\u003e[\\d\\-:, ]+) .* \\| (?P\u003ctype\u003e.*)"
			},
			{
				"mode": "key_value",
				"field": "message.6",
				"format": "{\"mode\":\"table\",\"table_row_delimiter\":\"\\n\"}"
			}
		],
		"fields": [
			{
				"rule_type": "graft",
				"origin_field": "agent_hostname",
				"key": "agent_hostname",
				"required": true
			},
			{
				"rule_type": "submatch",
				"origin_field": "message.0.date",
				"key": "date",
				"value_type": "date",
				"extention": {
					"preset_filter": "allmatch",
					"default_value": "",
					"date_format": "2006-01-02 15:04:05,999",
					"date_location": "Local"
				},
				"required": true
			},
			{
				"rule_type": "submatch",
				"origin_field": "message.4",
				"key": "func_no",
				"value_type": "text",
				"extention": {
					"regexp": "\\|5:(\\d+)\\|"
				},
				"required": true
			},
			{
				"rule_type": "submatch",
				"origin_field": "message.6.fund_account",
				"key": "fund_account",
				"value_type": "text",
				"extention": {
					"preset_filter": "allmatch"
				},
				"required": false
			},
			{
				"rule_type": "submatch",
				"origin_field": "message.0.type",
				"key": "type",
				"value_type": "text",
				"extention": {
					"preset_filter": "allmatch"
				},
				"required": true
			},
			{
				"rule_type": "graft",
				"origin_field": "message.2",
				"key": "req_no",
				"required": true
			},
			{
				"rule_type": "graft",
				"origin_field": "message.4",
				"key": "tag",
				"required": true
			},
			{
				"rule_type": "graft",
				"origin_field": "message.6.__origin__",
				"key": "data",
				"required": true
			},
			{
				"rule_type": "graft",
				"origin_field": "message.6.error_no",
				"key": "error_no",
				"required": false
			},
			{
				"rule_type": "graft",
				"origin_field": "message.6.error_info",
				"key": "error_info",
				"required": false
			},
			{
				"rule_type": "remove",
				"origin_field": "message"
			}
		]
	}
	`
	settings2 = `
	{
		"pre_function": [
			{
				"mode": "contains",
				"format": "agent_hostname"
			},
			{
				"mode": "contains",
				"format": "CLIENT_REQ"
			}
		],
		"prefix_match": [
			"message",
			"agent_hostname"
		],
		"pre_extract": [
			{
				"mode": "split",
				"field": "message",
				"format": "{\"delimiter\":\"\\\\n\",\"num_of_substring\":7}"
			},
			{
				"mode": "regexp",
				"field": "message.0",
				"format": "^(?P\u003cdate\u003e[\\d\\-:, ]+) .* \\| (?P\u003ctype\u003e.*)"
			},
			{
				"mode": "key_value",
				"field": "message.6",
				"format": "{\"mode\":\"table\",\"table_row_delimiter\":\"\\\\n\"}"
			}
		],
		"fields": [
			{
				"rule_type": "graft",
				"origin_field": "agent_hostname",
				"key": "agent_hostname",
				"required": true
			},
			{
				"rule_type": "submatch",
				"origin_field": "message.0.date",
				"key": "date",
				"value_type": "date",
				"extention": {
					"preset_filter": "allmatch",
					"default_value": "",
					"date_format": "2006-01-02 15:04:05,999",
					"date_location": "Local"
				},
				"required": true
			},
			{
				"rule_type": "submatch",
				"origin_field": "message.4",
				"key": "func_no",
				"value_type": "text",
				"extention": {
					"regexp": "\\|5:(\\d+)\\|"
				},
				"required": true
			},
			{
				"rule_type": "submatch",
				"origin_field": "message.6.fund_account",
				"key": "fund_account",
				"value_type": "text",
				"extention": {
					"preset_filter": "allmatch"
				},
				"required": false
			},
			{
				"rule_type": "submatch",
				"origin_field": "message.0.type",
				"key": "type",
				"value_type": "text",
				"extention": {
					"preset_filter": "allmatch"
				},
				"required": true
			},
			{
				"rule_type": "graft",
				"origin_field": "message.2",
				"key": "req_no",
				"required": true
			},
			{
				"rule_type": "graft",
				"origin_field": "message.4",
				"key": "tag",
				"required": true
			},
			{
				"rule_type": "graft",
				"origin_field": "message.6.__origin__",
				"key": "data",
				"required": true
			},
			{
				"rule_type": "graft",
				"origin_field": "message.6.error_no",
				"key": "error_no",
				"required": false
			},
			{
				"rule_type": "graft",
				"origin_field": "message.6.error_info",
				"key": "error_info",
				"required": false
			},
			{
				"rule_type": "remove",
				"origin_field": "message"
			}
		]
	}	
	`
)
