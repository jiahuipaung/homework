package extract

import (
	"context"
	"testing"

	"github.com/flashcatcloud/fc-stash/types"
	"github.com/flashcatcloud/fc-stash/utils"
)

// Note:
// 测试时go test -count=10跑多次

func TestJsonPruneV2_Handle(t *testing.T) {
	type args struct {
		ctx   context.Context
		input string
	}
	tests := []struct {
		name    string
		prune   *JsonPruneV2
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "prune_A",
			prune: &JsonPruneV2{
				Fields: []*FieldExtract{
					{
						RuleType:    "sub_match",
						Key:         "log",
						OriginField: "log",
						Extract: ExtractRule{
							PresetFilter: "allmatch",
						},
					},
				},
				PreExtract: []*JsonPreSourceExtract{
					{
						Field: "log",
						Mode:  "regexp",
						// 执行转义
						Format: `(?P<ext_time>\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}\.\d{3})\s+\[(?P<ext_pod>(.*?))\]\s+\[(?P<ext_thread>(.*?))\]\s+(?P<logLevel>.*)\s+(?P<ext_className>.*)\s+\-\s+\[(?P<ext_appd_trace_id>.*)\]\-\[(?P<ext_request_id>.*)\]\-\[(?P<ext_traceId>.*)\]\s+\[(?P<ext_sysId>.*)\]\s+\[(?P<ext_LogType>.*)\]\s+\[(?P<ext_ReqClassName>.*)\]\s+\[(?P<ext_ReqMethodName>.*)\]\s+\[(?P<ext_OutSysId>.*)\]\s+\[(?P<ext_OutBizFrom>.*)\]\s+\[(?P<ext_OutBizScenario>.*)\]\s+\[(?P<ext_InvokeResult>.*)\]\s+\[(?P<ext_RespCode>.*)\]\s+\[(?P<ext_RespMsg>.*)\]\s+\[(?P<ext_Taken>.*)\]\s+\[(?P<aggrfield>.*)\]\s+(?P<ext_msg>[\s\S]*)`,
					},
				},
				OverwriteOrigin: false,
			},
			args: args{
				ctx:   context.Background(),
				input: input0,
			},
			want:    `{"@timestamp_fluentbit":1715053811.758151,"aggrfield":"{\"EndTime\":\"2024-05-07 11:50:11.758\",\"StartTime\":\"2024-05-07 11:50:11.757\",\"monitorFlag\":\"true\",\"UrlFunctionDesc\":\"定时任务-订单上报oms重试\"}","env":"cn-rke-ptozonesit","ext_InvokeResult":"true","ext_LogType":"controller","ext_OutBizFrom":"","ext_OutBizScenario":"","ext_OutSysId":"","ext_ReqClassName":"com.cn.rdc.core.soc.controller.TaskController","ext_ReqMethodName":"uploadRetry","ext_RespCode":"","ext_RespMsg":"","ext_Taken":"1","ext_appd_trace_id":"","ext_className":"c.m.c.r.l.s.l.i.ControllerLogHandleStrategy","ext_msg":"url=http://pos-aioserv.pos:8080/soc-coreserv/v1/task/upload-retry, request=null, response=\"ok\"","ext_pod":"pos-aio-container","ext_request_id":"","ext_sysId":"POS-AIO","ext_thread":"XNIO-1 task-94","ext_time":"2024-05-07 11:50:11.758","ext_traceId":"a4d5fac156fe4663940f0894a14d0d61","filepath":"/var/log/app/pos/pos-aioserv/pos-aio-container.log","log":"2024-05-07 11:50:11.758 [pos-aio-container] [XNIO-1 task-94] INFO  c.m.c.r.l.s.l.i.ControllerLogHandleStrategy - []-[]-[a4d5fac156fe4663940f0894a14d0d61] [POS-AIO] [controller] [com.cn.rdc.core.soc.controller.TaskController] [uploadRetry] [] [] [] [true] [] [] [1] [{\"EndTime\":\"2024-05-07 11:50:11.758\",\"StartTime\":\"2024-05-07 11:50:11.757\",\"monitorFlag\":\"true\",\"UrlFunctionDesc\":\"定时任务-订单上报oms重试\"}] url=http://pos-aioserv.pos:8080/soc-coreserv/v1/task/upload-retry, request=null, response=\"ok\"","logLevel":"INFO "}`,
			wantErr: false,
		},
		{
			name: "prune_B",
			prune: &JsonPruneV2{
				PreExtract: []*JsonPreSourceExtract{
					{
						Field:  "message",
						Mode:   "regexp",
						Format: `#\s+Time:\s+(?P<Time>.+)\n#\s+User@Host:\s+(?P<user>\w+)\[\w+\]\s+@\s+(?P<proxyHost>(\w|\[|\]|\.|\s)+)\s+Id:\s+(?P<threadID>\d+)\n# Schema: (?P<Killed>\w+)  Last_errno: (?P<Last_errno>\d+)  Killed: (?P<Killed>\d+)\n# Query_time:\s+(?P<queryTime>\w+\.?\w+)\s+Lock_time:\s+(?P<lockTime>\w+\.?\w+)\s+Rows_sent:\s+(?P<rowsSent>\d+)\s+Rows_examined:\s+(?P<rowsExamined>\d+)\s+Rows_affected:\s+(?P<Rows_affected>\d+)\s+Bytes_sent:\s+(?P<Bytes_sent>\d+)\n(use\s+(?P<use>.*);\n)?SET\s+timestamp=(?P<sqlTimestamp>\d+);\n(?P<sqlText>(.|\n)*)`,
					},
					{
						Field: "fctags",
						Mode:  "json",
					},
				},
				OverwriteOrigin: false,
			},
			args: args{
				ctx:   context.Background(),
				input: input1,
			},
			want:    `{"":";","Bytes_sent":"13391","Killed":"0","Last_errno":"0","Rows_affected":"0","Time":"2024-11-06T08:40:05.519318Z","agent_hostname":"cn-r1990564-edge01.store","fcservice":"xxxx","fcsource":"127.0.0.1","fctags":{"dirname":"/var/lib/rancher/k3s/storage/pvc-12b8953a-2ebb-459a-b548-26f0cd866a19_pos-system_data-bitpoke-cluster-mysql-0","env":"prod","filename":"bitpoke-cluster-mysql-0-slow.log","frontend":"store","store":"r1990564","storeNameCn":"餐厅"},"lockTime":"0.000001","message":"# Time: 2024-11-06T08:40:05.519318Z\n# User@Host: soc_rw[soc_rw] @  [172.21.1.251]  Id: 9558478\n# Schema: soc  Last_errno: 0  Killed: 0\n# Query_time: 0.105158  Lock_time: 0.000001  Rows_sent: 33  Rows_examined: 132734  Rows_affected: 0  Bytes_sent: 13391\nuse soc;\nSET timestamp=1730882405;\nSELECT  \n id,snow_id,order_item_id,order_id,store_code,product_code,product_name,qty,unit_price,product_level,parent_product_id,day_part_code,order_type,qty_promo,real_qty,total_price,cost_include,avg_discount,avg_discount_amount,real_amount,tax_rate,tax_sub_type,tax_id,family_group,sub_family_group,category,department,department_class,department_sub_class,avg_discount_id,stld_transaction_kind,kvs_name_cn,kvs_name_en,product_type,action,round,compose_qty,promo_amount,tax_amount,discount_tax_amount,promo_tax_amount,bd_tax_amount,bp_tax_amount,create_time,update_time,create_by,update_by,deleted \n  FROM order_product      \n\n WHERE  deleted=0\n\nAND (store_code = '' AND order_item_id IN (12344));","msg_key":"cn-r1990564-edge01.store/","proxyHost":"[172.21.1.251] ","queryTime":"0.105158","rowsExamined":"132734","rowsSent":"33","sqlText":"SELECT  \n id,snow_id,order_item_id,order_id,store_code,product_code,product_name,qty,unit_price,product_level,parent_product_id,day_part_code,order_type,qty_promo,real_qty,total_price,cost_include,avg_discount,avg_discount_amount,real_amount,tax_rate,tax_sub_type,tax_id,family_group,sub_family_group,category,department,department_class,department_sub_class,avg_discount_id,stld_transaction_kind,kvs_name_cn,kvs_name_en,product_type,action,round,compose_qty,promo_amount,tax_amount,discount_tax_amount,promo_tax_amount,bd_tax_amount,bp_tax_amount,create_time,update_time,create_by,update_by,deleted \n  FROM order_product      \n\n WHERE  deleted=0\n\nAND (store_code = '' AND order_item_id IN (12344));","sqlTimestamp":"1730882405","status":"info","threadID":"9558478","timestamp":1730882406987,"topic":"mysql-gzip","use":"soc","user":"soc_rw"}`,
			wantErr: false,
		},
		{
			name: "prune_C",
			prune: &JsonPruneV2{
				PreExtract: []*JsonPreSourceExtract{
					{
						Field: "log",
						Mode:  "json",
					},
					{
						Field:  "log.log",
						Mode:   "gonx",
						Format: `$remote_addr - $remote_user [$time_local] "$request" $status $body_bytes_sent "$http_referer" "$http_user_agent" $request_length $request_time [$proxy_upstream_name] [$proxy_alternative_upstream_name] $upstream_addr $upstream_response_length $upstream_response_time $upstream_status $req_id`,
					},
				},
				OverwriteOrigin: false,
			},
			args: args{
				ctx:   context.Background(),
				input: input2,
			},
			want:    `{"@timestamp_fluentbit":1744727180.085154,"env":"cn-rke-ptozone09","filepath":"/var/log/pods/kube-system_ingress-nginx-controller-595c45c584-8k7ks_bef96167-ff36-4659-91fb-7d8bdc5eeb3b/controller/0.log","log":{"body_bytes_sent":"85","http_referer":"-","http_user_agent":"okhttp/3.14.8","log":"127.0.0.1 - - [15/Apr/2025:22:26:20 +0800] \"GET /soc-coreserv/v1/order/rank/1030701600000542978167290385 HTTP/2.0\" 200 85 \"-\" \"okhttp/3.14.8\" 166 0.002 [pos-pos-aioserv-http] [] 172.23.86.254:8080 102 0.001 200 1023202\n","proxy_alternative_upstream_name":"","proxy_upstream_name":"pos-pos-aioserv-http","remote_addr":"127.0.0.1","remote_user":"-","req_id":"1023202\n","request":"GET /soc-coreserv/v1/order/rank/1030701600000542978167290385 HTTP/2.0","request_length":"166","request_time":"0.002","status":"200","stream":"stdout","time":"2025-04-15T14:26:20.085122099Z","time_local":"15/Apr/2025:22:26:20 +0800","upstream_addr":"172.23.86.254:8080","upstream_response_length":"102","upstream_response_time":"0.001","upstream_status":"200"},"marketCityNameCn":"上海"}`,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.prune.Compile(); err != nil {
				t.Fatal(err)
			}
			got, err := tt.prune.Handle(tt.args.ctx, &types.LogEvent{
				Message: tt.args.input,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("Handle() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			/*if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Handle() got = %v, want %v", got, tt.want)
			}*/

			if utils.MustToJsonString(got) != tt.want {
				t.Errorf("Handle() got = %v, want %v", got, tt.want)
			}
		})
	}
}

var input0 = `
{
  "@timestamp_fluentbit": 1715053811.758151,
  "filepath": "/var/log/app/pos/pos-aioserv/pos-aio-container.log",
  "log": "2024-05-07 11:50:11.758 [pos-aio-container] [XNIO-1 task-94] INFO  c.m.c.r.l.s.l.i.ControllerLogHandleStrategy - []-[]-[a4d5fac156fe4663940f0894a14d0d61] [POS-AIO] [controller] [com.cn.rdc.core.soc.controller.TaskController] [uploadRetry] [] [] [] [true] [] [] [1] [{\"EndTime\":\"2024-05-07 11:50:11.758\",\"StartTime\":\"2024-05-07 11:50:11.757\",\"monitorFlag\":\"true\",\"UrlFunctionDesc\":\"定时任务-订单上报oms重试\"}] url=http://pos-aioserv.pos:8080/soc-coreserv/v1/task/upload-retry, request=null, response=\"ok\"",
  "env": "cn-rke-ptozonesit"
}
`
var input1 = `
{
  "message": "# Time: 2024-11-06T08:40:05.519318Z\n# User@Host: soc_rw[soc_rw] @  [172.21.1.251]  Id: 9558478\n# Schema: soc  Last_errno: 0  Killed: 0\n# Query_time: 0.105158  Lock_time: 0.000001  Rows_sent: 33  Rows_examined: 132734  Rows_affected: 0  Bytes_sent: 13391\nuse soc;\nSET timestamp=1730882405;\nSELECT  \n id,snow_id,order_item_id,order_id,store_code,product_code,product_name,qty,unit_price,product_level,parent_product_id,day_part_code,order_type,qty_promo,real_qty,total_price,cost_include,avg_discount,avg_discount_amount,real_amount,tax_rate,tax_sub_type,tax_id,family_group,sub_family_group,category,department,department_class,department_sub_class,avg_discount_id,stld_transaction_kind,kvs_name_cn,kvs_name_en,product_type,action,round,compose_qty,promo_amount,tax_amount,discount_tax_amount,promo_tax_amount,bd_tax_amount,bp_tax_amount,create_time,update_time,create_by,update_by,deleted \n  FROM order_product      \n\n WHERE  deleted=0\n\nAND (store_code = '' AND order_item_id IN (12344));",
  "status": "info",
  "timestamp": 1730882406987,
  "agent_hostname": "cn-r1990564-edge01.store",
  "fcservice": "xxxx",
  "fcsource": "127.0.0.1",
  "fctags": "{\"dirname\":\"/var/lib/rancher/k3s/storage/pvc-12b8953a-2ebb-459a-b548-26f0cd866a19_pos-system_data-bitpoke-cluster-mysql-0\",\"env\":\"prod\",\"filename\":\"bitpoke-cluster-mysql-0-slow.log\",\"frontend\":\"store\",\"store\":\"r1990564\",\"storeNameCn\":\"餐厅\"}",
  "topic": "mysql-gzip",
  "msg_key": "cn-r1990564-edge01.store/"
}
`

var input2 = `
{
  "@timestamp_fluentbit": 1744727180.085154,
  "filepath": "/var/log/pods/kube-system_ingress-nginx-controller-595c45c584-8k7ks_bef96167-ff36-4659-91fb-7d8bdc5eeb3b/controller/0.log",
  "log": "{\"log\":\"127.0.0.1 - - [15/Apr/2025:22:26:20 +0800] \\\"GET /soc-coreserv/v1/order/rank/1030701600000542978167290385 HTTP/2.0\\\" 200 85 \\\"-\\\" \\\"okhttp/3.14.8\\\" 166 0.002 [pos-pos-aioserv-http] [] 172.23.86.254:8080 102 0.001 200 1023202\\n\",\"stream\":\"stdout\",\"time\":\"2025-04-15T14:26:20.085122099Z\"}",
  "env": "cn-rke-ptozone09",
  "marketCityNameCn": "上海"
}
`
