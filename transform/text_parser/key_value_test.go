package text_parser

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/flashcatcloud/fc-stash/utils"
)

func Test_KeyValue(t *testing.T) {
	input := map[string]string{
		"":                            "k1=v1,k2=v2,k3=v3,k4=v4",
		"{\"field_delimiter\":\" \"}": "k11=v11 k22=v22 k33=v33    k44=v44",
		"{\"key_value_delimiter\":\":\",\"field_delimiter\":\" \"}": "k111:v11 k222:v22 k333:v333    k444:v444",
	}
	for k, v := range input {
		parser, err := NewKeyValueParser(k, true)
		if err != nil {
			t.Fatal(err)
		}
		fields, err := parser.Parse(v)
		if err != nil {
			fmt.Println(err)
		} else {
			for k := range fields {
				fmt.Println(k, fields[k])
			}
		}
	}
}

func Test_KeyValueModeTable(t *testing.T) {
	inputs := []string{
		`fund_account|futu_entrust_price|entrust_bs|futu_code|entrust_amount|entrust_time|entrust_no|subcribe_type|trade_server|function_id|
12345678|10915.000|2|OI301|1|22:58:42||112|CTP|112|`,
		"bank_no|money_type|fund_account|password|trade_server|UserCode|SessionNo|branch_no|op_branch_no|client_id|UserParam1|UserParam2|UserParam3|entrust_safety|entrust_way|op_entrust_way|mac_addr|client_ver|cpuid|safety_info|op_station|internal_ip|disk_serial_id|terminal_type|imei_code|terminal_way|request_num|\n||12345678|******|12345678||427ab61a-4008-11ed-b10f-b4969165e65f|7090|7090|85194167||||1|8|8|A4:50:46:D4:05:53|2.2.1|||12345678;10.0.0.5;A4:50:46:D4:05:53;701bf4ce0e364b274fb4adf37b4b4a8b;02|10.0.0.5|701bf4ce0e364b274fb4adf37b4b4a8b|02|701bf4ce0e364b274fb4adf37b4b4a8b|8|1000|",
	}
	parser, err := NewKeyValueParser(`{"mode":"table"}`, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		items, err := parser.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(items)
		fmt.Println(len(items))
	}
}

func Test_KeyValueModeMulitRows(t *testing.T) {
	inputs := []string{
		`contract_code|entrust_time|futu_entrust_price|entrust_amount|entrust_bs|futures_direction|entrust_status|entrust_no|cancel_amount|business_amount|init_date|futu_exch_type|exchange_name|futures_account|hedge_type|business_balance|
合约代码|委托时间|委托价格|委托数量|买卖方向|开平方向|委托状态|委托编号|撤单数量|成交数量|发生日期||交易名称|期货账号|套保标志|成交金额|
名称|时间|价格|数量|买卖|开平|状态|||成交数|||||||
cu2210|10573|8760.000|1|卖出|平仓|已报|7979|0|0|202001|F3|海易所|027110|1|0|
cu2210|10037|8660.000|2|卖出|平仓|已撤|3894|2|0|202901|F3|海易所|022510|1|0|`,
		`mac_addr|client_ver|op_branch_no|UserCode|fund_account|op_entrust_way|branch_no|entrust_way|internal_ip|version|UserParam1|UserParam2|terminal_type|mobile_uuid|UserParam3|trade_server|client_id|op_station|password|SessionNo|terminal_os|
9C0E942-38F-4A8B-ACF-2009611738|2..0|2071||100066|1000066|271|8|19.16.32.2||||01|9C0C942-38F-4AB-A7C-20036113D8||206041512|100066|;12.16.32.2;9CCE942-33F-4AB-A7CF-209361173D;01|******|b7515d2-2a1-11d-810-52400cc55c|15.5|`,
	}
	parser, err := NewKeyValueParser(
		`{
			"mode":"table",
			"table_value_multiline": true,
			"table_value_pattern": "[\\da-zA-Z]+"
		 }`,
		false)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		items, err := parser.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(utils.MustToJsonString(items))
	}
}

func Test_KeyValueModeVerticalTable(t *testing.T) {
	input := `
# 1	app_id	-------GTJA_LUCYAND_1.0.0
# 2	auth_code	-------20190507JHANDR01
# 3	app_sys_info	-------ZDkxqHG4FwcV50IZ8wNw8danLx6gUx24f9xCfyR2aExX6HRlg5P6BcgAbTD4RWBU6iwzScBpcEBQOTTMFkoqqNYyOWmubWLOaV5YWzvIO3pWb79+eKb1CXUqPuZg+eDTFvL+I0t7ugKnu1rDj9nwj1KanRGlNH3JCABk9NvtIO2fRgm/2RQkIKQcKIJFr6GXVXEVAhmYKzgE6y32pDzmmE7ElmpNAZV0LSJRiilr4A2LR9SJIDyA8d8yLwxTcXSHtbwY89rRwMPg82QhmoC56Xf+z/KCeVqkAQhE0qDn4TaZ01yImPvcarYm0D5uWOiLbpZ1GlhqEsRLPYTxdazybbSeOeod8LYV
# 4	ctp_app_sys_info	-------
# 5	ks_app_sys_info	-------
# 6	app_sys_info_integrity	-------
# 7	app_abnormal_type	-------
# 8	version	-------2.0.1.0
# 9	entrust_way	-------8
# 10	op_station	-------13122903262;10.76.45.20;DE:55:1D:EB:9C:E9;2d9c81a9148defd2d771c99a5bfc1ca1;MD
# 11	mobile_code	-------0081500186
# 12	mac_addr	-------DE:55:1D:EB:9C:E9
# 13	internal_ip	-------10.76.45.20
# 14	ip_port	-------43608
# 15	cpuid	-------
# 16	disk_serial_id	-------2d9c81a9148defd2d771c99a5bfc1ca1
# 17	client_ver	-------2.4.0
# 18	safety_info	-------
# 19	entrust_safety	-------0
# 20	branch_no	-------2071
# 21	input_content	-------1
# 22	account_content	-------0081500186
# 23	dyn_pwd	-------
# 24	comm_pwd	-------******
# 25	certficate_no	-------
# 26	cert_isscn	-------
# 27	cert_cn	-------
# 28	cert_info	-------
# 29	terminal_type	-------MD
# 30	trade_server	-------2006041512
# 31	password	-------******
`
	parser, err := NewKeyValueParser(`{
		"mode": "vertical_table",
		"table_key_index": 1,
		"table_value_index": 2,
		"table_value_trim": "-"
	}`, false)
	if err != nil {
		t.Fatal(err)
	}
	items, err := parser.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(items)
	fmt.Println(len(items))
}

func TestParseKeyValue(t *testing.T) {
	// 测试数据和期望结果
	tests := []struct {
		name              string
		origin            string
		fieldDelimiter    string
		keyValueDelimiter string
		want              map[string]interface{}
		wantErr           bool
	}{
		{
			name:              "Default delimiters",
			origin:            "k1=v1,k2=v2,k3=v3,k4=v4",
			fieldDelimiter:    ",",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"k1": "v1", "k2": "v2", "k3": "v3", "k4": "v4"},
			wantErr:           false,
		},

		{
			name:              "space key-value pairs",
			origin:            "k11=v11 k22=v22 k33=v33    k44=v44",
			fieldDelimiter:    " ",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"k11": "v11", "k22": "v22", "k33": "v33   ", "k44": "v44"},
			wantErr:           false,
		},

		{
			name:              "Key-value pairs with custom delimiters",
			origin:            "k111:v11 k222:v22 k333:v333    k444:v444",
			fieldDelimiter:    " ",
			keyValueDelimiter: ":",
			want:              map[string]interface{}{"k111": "v11", "k222": "v22", "k333": "v333   ", "k444": "v444"},
			wantErr:           false,
		},

		{
			name:              "Key-value pairs with spaces inside quotes",
			origin:            `k1="hello world" k2=v2`,
			fieldDelimiter:    " ",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"k1": "\"hello world\"", "k2": "v2"},
			wantErr:           false,
		},
		{
			name:              "Unbalanced quotes",
			origin:            `k1="hello world k2=v2`,
			fieldDelimiter:    " ",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"k1": "\"hello world", "k2": "v2"},
			wantErr:           false,
		},
		{
			name:              "Key-value pairs with spaces inside quotes and escape quotes",
			origin:            "k1=\"hello world\" k2=v2",
			fieldDelimiter:    " ",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"k1": "\"hello world\"", "k2": "v2"},
			wantErr:           false,
		},
		{
			name:              "Key-value pairs with spaces inside quotes and escape quotes",
			origin:            "k1='hello world' k2=v2",
			fieldDelimiter:    " ",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"k1": "'hello world'", "k2": "v2"},
			wantErr:           false,
		},
		{
			name:              "Key-value pairs with json inside quotes and escape quotes",
			origin:            "appname:vn,level:INFO,hostname:EA-Money-nginx-1,data:\"{\"topic\":\"filebeat\",\"type\":\"beat\"}\"",
			fieldDelimiter:    ",",
			keyValueDelimiter: ":",
			want:              map[string]interface{}{"appname": "vn", "level": "INFO", "hostname": "EA-Money-nginx-1", "data": "\"{\"topic\":\"filebeat\",\"type\":\"beat\"}\""},
			wantErr:           false,
		},
		{
			name:              "Key-value pairs with json inside",
			origin:            `source=nginx remote_addr=127.0.0.1 data="{\"key\":\"value\",\"inner_data\":\"hello world\"}"`,
			fieldDelimiter:    " ",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"source": "nginx", "remote_addr": "127.0.0.1", "data": `"{\"key\":\"value\",\"inner_data\":\"hello world\"}"`},
			wantErr:           false,
		},
		{
			name:              "Key-value pairs with json inside quotes and escape quotes",
			origin:            "date=2022-06-16T11:44:56.756Z appname=[vn,ws4zs,] message=\"网型变更: ws4zs开始,   变更时间戳:1655365290514\"",
			fieldDelimiter:    " ",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"date": "2022-06-16T11:44:56.756Z", "appname": "[vn,ws4zs,]", "message": "\"网型变更: ws4zs开始,   变更时间戳:1655365290514\""},
			wantErr:           false,
		},
		{
			name:              "Key-value pairs with json inside",
			origin:            "source=nginx remote_addr=127.0.0.1 upstream_response_time= data={\"key\":\"value\",\"inner_data\":\"hello world\"}",
			fieldDelimiter:    " ",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"source": "nginx", "remote_addr": "127.0.0.1", "upstream_response_time": "", "data": "{\"key\":\"value\",\"inner_data\":\"hello world\"}"},
			wantErr:           false,
		},
		{
			name:              "Key-value pairs with json inside",
			origin:            "source=nginx remote_addr=127.0.0.1 data={\"key\":\"value\",\"inner_data\":\"hello world\"}",
			fieldDelimiter:    " ",
			keyValueDelimiter: "=",
			want:              map[string]interface{}{"source": "nginx", "remote_addr": "127.0.0.1", "data": "{\"key\":\"value\",\"inner_data\":\"hello world\"}"},
			wantErr:           false,
		},
	}

	for idx, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p, err := NewKeyValueParser(fmt.Sprintf(`{"field_delimiter":"%s", "key_value_delimiter":"%s","key_pattern":"[\\w\\d]+"}`, tt.fieldDelimiter, tt.keyValueDelimiter), false)
			if err != nil {
				t.Errorf("error = %v", err)
			}
			got, err := p.Parse(tt.origin)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseKeyValue() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ParseKeyValue() got = %v, want %v, idx: %d", got, tt.want, idx)
			}
		})
	}
}

// 以下部分是性能测试

// generateInput 生成测试输入

var benchmarkInput = generateInput(100)

// randomString generates a random string of a given length
func randomString(length int) string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	return b.String()
}

// generateInput generates a string with a given number of key-value pairs with random lengths
func generateInput(size int) string {
	var sb strings.Builder
	for i := 0; i < size; i++ {
		keyLength := rand.Intn(20) + 3   // keys of random length between 3 and 10
		valueLength := rand.Intn(60) + 3 // values of random length between 3 and 20

		key := randomString(keyLength)
		value := randomString(valueLength)

		sb.WriteString(key)
		sb.WriteString("=")
		sb.WriteString(value)

		if i < size-1 {
			sb.WriteString("&")
		}
	}
	return sb.String()
}

func BenchmarkParseKeyVakue(b *testing.B) {
	b.ResetTimer()
	p, _ := NewKeyValueParser(`{"field_delimiter":"&", "key_value_delimiter":"=","key_pattern":"[\\w\\d]+"}`, false)

	for i := 0; i < b.N; i++ {
		_, err := p.Parse(benchmarkInput)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkSimpleParseKeyValue(b *testing.B) {
	b.ResetTimer()
	ret := make(map[string]interface{})
	for i := 0; i < b.N; i++ {
		err := SimpleParseKeyValue(ret, benchmarkInput, "&", "=")
		if err != nil {
			b.Fatal(err)
		}
	}
}
