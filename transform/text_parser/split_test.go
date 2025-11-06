package text_parser

import (
	"fmt"
	"testing"
)

func Test_SplitCommon(t *testing.T) {
	inputs := []string{
		`fund_account|futu_entrust_price|entrust_bs|futu_code|entrust_amount|entrust_time|entrust_no|subcribe_type|trade_server|function_id|
12345678|10915.000|2|OI301|1|22:58:42||112|CTP|112|`,
		"bank_no|money_type|fund_account|password|trade_server|UserCode|SessionNo|branch_no|op_branch_no|client_id|UserParam1|UserParam2|UserParam3|entrust_safety|entrust_way|op_entrust_way|mac_addr|client_ver|cpuid|safety_info|op_station|internal_ip|disk_serial_id|terminal_type|imei_code|terminal_way|request_num|\n||12345678|******|12345678||427ab61a-4008-11ed-b10f-b4969165e65f|7090|7090|85194167||||1|8|8|A4:50:46:D4:05:53|2.2.1|||12345678;10.0.0.5;A4:50:46:D4:05:53;701bf4ce0e364b274fb4adf37b4b4a8b;02|10.0.0.5|701bf4ce0e364b274fb4adf37b4b4a8b|02|701bf4ce0e364b274fb4adf37b4b4a8b|8|1000|",
	}
	parser, err := NewSplitParser("|", false)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		items, err := parser.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(items)
	}
}

func Test_SplitArgsJson(t *testing.T) {
	inputs := []string{
		`fund_account|futu_entrust_price|entrust_bs|futu_code|entrust_amount|entrust_time|entrust_no|subcribe_type|trade_server|function_id|
12345678|10915.000|2|OI301|1|22:58:42||112|CTP|112|`,
		"bank_no|money_type|fund_account|password|trade_server|UserCode|SessionNo|branch_no|op_branch_no|client_id|UserParam1|UserParam2|UserParam3|entrust_safety|entrust_way|op_entrust_way|mac_addr|client_ver|cpuid|safety_info|op_station|internal_ip|disk_serial_id|terminal_type|imei_code|terminal_way|request_num|\n||12345678|******|12345678||427ab61a-4008-11ed-b10f-b4969165e65f|7090|7090|85194167||||1|8|8|A4:50:46:D4:05:53|2.2.1|||12345678;10.0.0.5;A4:50:46:D4:05:53;701bf4ce0e364b274fb4adf37b4b4a8b;02|10.0.0.5|701bf4ce0e364b274fb4adf37b4b4a8b|02|701bf4ce0e364b274fb4adf37b4b4a8b|8|1000|",
	}
	parser, err := NewSplitParser(`{"delimiter":"|"}`, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		items, err := parser.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(items)
	}
}

func Test_SplitArgsLimit(t *testing.T) {
	inputs := []string{
		`fund_account|futu_entrust_price|entrust_bs|futu_code|entrust_amount|entrust_time|entrust_no|subcribe_type|trade_server|function_id|
12345678|10915.000|2|OI301|1|22:58:42||112|CTP|112|`,
		"bank_no|money_type|fund_account|password|trade_server|UserCode|SessionNo|branch_no|op_branch_no|client_id|UserParam1|UserParam2|UserParam3|entrust_safety|entrust_way|op_entrust_way|mac_addr|client_ver|cpuid|safety_info|op_station|internal_ip|disk_serial_id|terminal_type|imei_code|terminal_way|request_num|\n||12345678|******|12345678||427ab61a-4008-11ed-b10f-b4969165e65f|7090|7090|85194167||||1|8|8|A4:50:46:D4:05:53|2.2.1|||12345678;10.0.0.5;A4:50:46:D4:05:53;701bf4ce0e364b274fb4adf37b4b4a8b;02|10.0.0.5|701bf4ce0e364b274fb4adf37b4b4a8b|02|701bf4ce0e364b274fb4adf37b4b4a8b|8|1000|",
	}
	parser, err := NewSplitParser(`{"delimiter":"|", "num_of_substring":4}`, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range inputs {
		items, err := parser.Parse(input)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println(items)
	}
}
