package text_parser

import (
	"fmt"
	"testing"
	"time"
)

func Test_Gonx(t *testing.T) {
	input := []string{
		"2022-06-16 15:41:33,595||INFO |[vn,ws4zs,]|Net_Business_Thread_1-6-1|NetService.java,173:网型变更:ws4zs开始,变更时间戳:1655365290514",
		"2022-06-17 14:15:10,503||INFO |[vn,,]|com.alibaba.nacos.naming.push.receiver|PushReceiver.java,86:received_push_data:{\"type\":\"dom\",\"data\":\"{\\\"name\\\":\\\"hps@@vn\\\",\\\"clusters\\\":\\\"DEFAULT\\\",\\\"cacheMillis\\\":10000,\\\"hosts\\\":[{\\\"instanceId\\\":\\\"10.200.100.67#8090#DEFAULT#hps@@vn\\\",\\\"ip\\\":\\\"10.200.100.67\\\",\\\"port\\\":8090,\\\"weight\\\":1.0,\\\"healthy\\\":true,\\\"enabled\\\":true,\\\"ephemeral\\\":true,\\\"clusterName\\\":\\\"DEFAULT\\\",\\\"serviceName\\\":\\\"hps@@vn\\\",\\\"metadata\\\":{\\\"preserved.register.source\\\":\\\"SPRING_CLOUD\\\"},\\\"instanceHeartBeatInterval\\\":5000,\\\"instanceHeartBeatTimeOut\\\":15000,\\\"ipDeleteTimeout\\\":30000}],\\\"lastRefTime\\\":1655446510502,\\\"checksum\\\":\\\"\\\",\\\"allIPs\\\":false,\\\"reachProtectionThreshold\\\":false,\\\"valid\\\":true}\",\"lastRefTime\":1479591867954158}:from:/10.200.100.26",
	}
	// format := "$date|$pid|$log_level|[$app_name,$biz_id,$user_id]|$thread_name|$file_name,$line:$message"
	format := "$date|$pid|$log_level|[$app_biz_user]|$thread_name|$file_message"
	parser, err := NewGonxParser(format, true)
	if err != nil {
		t.Fatal(err)
	}
	for i := range input {
		fields, err := parser.Parse(input[i])
		if err != nil {
			fmt.Println(err)
		} else {
			for k := range fields {
				fmt.Println(k, fields[k], len(fields[k].(string)))
			}
		}
	}
}

func Test_Regexp(t *testing.T) {
	input := []string{
		"2022-06-16 15:41:33,595||INFO |[vn,ws4zs,]|Net_Business_Thread_1-6-1|NetService.java,173:网型变更:ws4zs开始,变更时间戳:1655365290514",
		"2022-06-17 14:15:10,503||INFO |[vn,,]|com.alibaba.nacos.naming.push.receiver|PushReceiver.java,86:received_push_data:{\"type\":\"dom\",\"data\":\"{\\\"name\\\":\\\"hps@@vn\\\",\\\"clusters\\\":\\\"DEFAULT\\\",\\\"cacheMillis\\\":10000,\\\"hosts\\\":[{\\\"instanceId\\\":\\\"10.200.100.67#8090#DEFAULT#hps@@vn\\\",\\\"ip\\\":\\\"10.200.100.67\\\",\\\"port\\\":8090,\\\"weight\\\":1.0,\\\"healthy\\\":true,\\\"enabled\\\":true,\\\"ephemeral\\\":true,\\\"clusterName\\\":\\\"DEFAULT\\\",\\\"serviceName\\\":\\\"hps@@vn\\\",\\\"metadata\\\":{\\\"preserved.register.source\\\":\\\"SPRING_CLOUD\\\"},\\\"instanceHeartBeatInterval\\\":5000,\\\"instanceHeartBeatTimeOut\\\":15000,\\\"ipDeleteTimeout\\\":30000}],\\\"lastRefTime\\\":1655446510502,\\\"checksum\\\":\\\"\\\",\\\"allIPs\\\":false,\\\"reachProtectionThreshold\\\":false,\\\"valid\\\":true}\",\"lastRefTime\":1479591867954158}:from:/10.200.100.26",
	}
	// format := "^(?P<date>[^|]*)\\|(?P<pid>[^|]*)\\|(?P<log_level>[^|]*)\\|\\[(?P<app_name>[^,]*),(?P<biz_id>[^,]*),(?P<user_id>[^]]*)\\]\\|(?P<thread_name>[^|]*)\\|(?P<file_name>[^,]*),(?P<line>[^:]*):(?P<message>.*)$"
	// format := `^(?P<date>[^|]*)\|(?P<pid>[^|]*)\|(?P<log_level>[^|]*)\|\[(?P<app_name>[^,]*),(?P<biz_id>[^,]*),(?P<user_id>[^]]*)\]\|(?P<thread_name>[^|]*)\|(?P<file_name>[^,]*),(?P<line>[^:]*):(?P<message>.*)$`
	format := "^(?P<date>[^|]*)\\|(?P<pid>[^|]*)\\|(?P<log_level>[^|]*)\\|\\[(?P<app_biz_user>[^|]*)\\]\\|(?P<thread_name>[^|]*)\\|(?P<file_message>.*)$"
	parser, err := NewRegexpParser(format, true)
	if err != nil {
		t.Fatal(err)
	}
	for i := range input {
		fields, err := parser.Parse(input[i])
		if err != nil {
			fmt.Println(err)
		} else {
			for k := range fields {
				fmt.Println(k, fields[k], len(fields[k].(string)))
			}
		}
	}
}

func Test_Time(t *testing.T) {
	loc, err := time.LoadLocation("Local")
	if err != nil {
		t.Fatal(err)
	}
	t0, err := time.ParseInLocation("2006-01-02 15:04:05.999", "2022-07-29 16:14:11,832", loc)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(t0)
}
