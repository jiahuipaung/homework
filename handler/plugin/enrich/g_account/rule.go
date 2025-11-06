package g_account

const (
	LogKeyDate           = "@timestamp"      // 必须
	LogKeyAccountRef     = "AccountRef"      // 必须
	LogKeyAccountID      = "AccountID"       // 必须
	LogKeyOrderStatus    = "OrderStatus"     // 必须
	LogKeyAgentHostname  = "agent_hostname"  // 必须
	LogKeyAccountIDStr   = "AccountID_str"   // 必须
	LogKeyAccountRefStr  = "AccountRef_str"  // 必须
	LogKeyOrderStatusStr = "OrderStatus_str" // 必须
)

var (
	dictrule = `
	{
		"mode": "json",
		"version": "log_prune",
		"json_settings": {
		  "pre_function": [
			{
			  "mode": "contains",
			  "format": "Account:AccountRef"
			}
		  ],
		  "prefix_match": [
			"message",
			"agent_hostname",
			"timestamp"
		  ],
		  "pre_extract": [
			{
			  "mode": "regexp",
			  "field": "message",
			  "format": "Account:(?P\u003cdata\u003e.*)"
			},
			{
			  "mode": "key_value",
			  "field": "message.data",
			  "format": "{\"field_delimiter\":\" \", \"key_value_delimiter\":\"=\"}"
			}
		  ],
		  "fields": [
			{
			  "rule_type": "submatch",
			  "origin_field": "timestamp",
			  "key": "@timestamp",
			  "value_type": "date",
			  "extention": {
				"preset_filter": "allmatch",
				"default_value": "",
				"date_format": "unix_milli",
				"date_location": "Local"
			  },
			  "required": false
			},
			{
			  "rule_type": "remove",
			  "origin_field": "message",
			  "key": "__root__.x",
			  "value_type": "text",
			  "extention": {
				"preset_filter": "allmatch",
				"default_value": ""
			  },
			  "required": false
			},
			{
			  "rule_type": "graft",
			  "origin_field": "message.data",
			  "key": "__root__",
			  "value_type": "text",
			  "extention": {
				"preset_filter": "allmatch",
				"default_value": ""
			  },
			  "required": false
			}
		  ]
		}
	  }
	`
)

var (
	datarule = `
	{
		"mode": "json",
		"version": "log_prune",
		"json_settings": {
		  "pre_function": [
			{
			  "mode": "contains",
			  "format": "AccountRef"
			}
		  ],
		  "prefix_match": [
			"message",
			"timestamp",
			"agent_hostname"
		  ],
		  "pre_extract": [
			{
			  "mode": "regexp",
			  "field": "message",
			  "format": "[a-zA-Z]+:(?P\u003cdata\u003e.*)"
			},
			{
			  "mode": "key_value",
			  "field": "message.data",
			  "format": "{\"field_delimiter\":\" \", \"key_value_delimiter\":\"=\"}"
			}
		  ],
		  "fields": [
			{
			  "rule_type": "submatch",
			  "origin_field": "timestamp",
			  "key": "@timestamp",
			  "value_type": "date",
			  "extention": {
				"preset_filter": "allmatch",
				"default_value": "",
				"date_format": "unix_milli",
				"date_location": "Local"
			  },
			  "required": false
			},
			{
			  "rule_type": "graft",
			  "origin_field": "message.data",
			  "key": "__root__",
			  "value_type": "text",
			  "extention": {
				"preset_filter": "allmatch",
				"default_value": ""
			  },
			  "required": false
			},
			{
			  "rule_type": "remove",
			  "origin_field": "message",
			  "key": "__root__.x",
			  "value_type": "text",
			  "extention": {
				"preset_filter": "allmatch",
				"default_value": ""
			  },
			  "required": false
			},
			{
			  "rule_type": "submatch",
			  "origin_field": "message.data.ErrorNo",
			  "key": "message.data.ErrorNo",
			  "value_type": "long",
			  "extention": {
				"preset_filter": "allmatch",
				"default_value": ""
			  },
			  "required": false
			},
			{
			  "rule_type": "submatch",
			  "origin_field": "message.data.OrderRef",
			  "key": "message.data.OrderRef",
			  "value_type": "long",
			  "extention": {
				"preset_filter": "allmatch",
				"default_value": ""
			  },
			  "required": false
			},
			{
			  "rule_type": "submatch",
			  "origin_field": "message.data.OrderStatus",
			  "key": "message.data.OrderStatus",
			  "value_type": "long",
			  "extention": {
				"preset_filter": "allmatch",
				"default_value": ""
			  },
			  "required": false
			}
		  ]
		}
	  }
	`
)
