### 正常的请求

#### response只有一行数据
```
2022-09-01 10:57:08,191 nioEventLoopGroup-6-1 [CLIENT_REQ()]-[INFO] | request
-no:
1661988106
-tag:
13:8|3:0|4:111|5:1544|6:2006041512|11:520|12:1015|69:qt#truevtk#eyJhbGciOiJTTTIiLCiI6MTY2MjAwMDk0N30.MEUCIBqsZrrSKINN9Qwk8c4SesoAV4pEVVFPEQCXhrqegqmLQT0DYuY0WNW6G_xUNUqI4#1markid#2ef79692dd174af0abc2776df282c532|10:[microservices,nginx_test_8.94#5,,t2,1568669700]|connection_no:156866700|
-data:
mac_addr|client_ver|op_branch_no|UserCode|fund_account|op_entrust_way|branch_no|entrust_way|internal_ip|version|UserParam1|UserParam2|terminal_type|mobile_uuid|UserParam3|trade_server|client_id|op_station|password|SessionNo|terminal_os|hedge_type|futu_exch_type|init_date|entrust_bs|contract_type|futu_product_type|contract_code|
9C0C942-383F-4A8B-A7CF-200931173D8|2.3.0|271||1000066|1000666|2071|8|19.168.3.82||||01|9C0C942-383F-4A8B-A7CF-200961173D8||200604512|1000666|;192.168.32.82;9C0C942-383F-4A8B-A7CF-209361173D;01|******|7515d2-29a1-11ed-8a0-52500c555c|15.5|1|F3|022901||||cu2210|

2022-09-01 10:57:08,195 nioEventLoopGroup-6-1 [CLIENT_REQ()]-[INFO] | response
-no:
1661988106
-tag:
13:8|3:1|4:111|5:1544|6:2006041512|11:520|12:1015|69:qt#truevtk#eyJhbGciOiJTTTIiLCJraWQZVVFPEQCXhrqegqmLQT0DYuY0WNW6G_xUNUqI4#1markid#2ef79692dd174af0abc2776df282c532|10:[]|connection_no:156869700|
-data:
hedge_type|contract_code|entrust_bs|open_bail_ratio|open_bail_balance|
1|cu2210|1|0.15|0.00|
```

#### response有多行数据
```
2022-09-01 10:57:08,195 nioEventLoopGroup-6-1 [CLIENT_REQ()]-[INFO] | response
-no:
1661988106
-tag:
13:8|3:1|4:111|5:1544|6:2006041512|11:520|12:1015|69:qt#truevtk#eyJhbGciOiJTTTI6MTY2MjAwMDk0N30.MEUCIBqsZrrSKINN9Qwk8c4SesoAV4pExbnmQRUrhE-b5Y0dAiEAvMNwqYZVVFPEQCXhrqegqmLQT0DYuY0WNW6G_xUNUqI4#1markid#2ef79692dd174af0abc2776df282c532|10:[]|connection_no:156866900|
-data:
hedge_type|contract_code|entrust_bs|open_bail_ratio|open_bail_balance|
1|cu2210|1|0.15|0.00|
1|cu2210|2|0.15|0.00|
```

#### response中有中文释义
关键字是包含 中文

```
2022-09-01 10:57:15,278 nioEventLoopGroup-6-1 [CLIENT_REQ()]-[INFO] | response
-no:
1661988107
-tag:
13:8|3:1|4:111|5:1504|6:2006041512|11:521|12:1016|69:qt#truevtk#eyJhbGciOiJTTTIiLCJraWQiOiIwYTk3NjU2Yy1lZDI0LTRkNDYtOTY0MS05YTNlMzdjNDc0MGEifQ.eyJzdWIiOiJ2c2JRMWFEK2UxaMDk0N30.MEUCIBqsZrrSKINN9Qwk8c4SesoAV4pExbnmQRUrhE-b5Y0dAiEAvMNwqYZVVFPEQCXhrqegqmLQT0DYuY0WNW6G_xUNUqI4#1markid#2ef79692dd174af0abc2776df282c532|10:[]|connection_no:156866700|
-data:
contract_code|entrust_time|futu_entrust_price|entrust_amount|entrust_bs|futures_direction|entrust_status|entrust_no|cancel_amount|business_amount|init_date|futu_exch_type|exchange_name|futures_account|hedge_type|business_balance|
合约代码|委托时间|委托价格|委托数量|买卖方向|开平方向|委托状态|委托编号|撤单数量|成交数量|发生日期||交易名称|期货账号|套保标志|成交金额|
名称|时间|价格|数量|买卖|开平|状态|||成交数|||||||
cu2210|105209|8760.000|1|卖出|平仓|已撤|7145|1|0|20901|F3|上海交易所|02275110|1|0|
cu2210|105129|8760.000|1|卖出|平仓|已撤|7661|1|0|20901|F3|上海交易所|02275110|1|0|
cu2210|101422|8760.000|2|卖出|平仓|已撤|4761|1|1|20901|F3|上海交易所|02275110|1|43800.00|
```


#### response是表单
关键字是包含`check_tab_data|`

```
2022-09-29 23:42:58,093 nioEventLoopGroup-27-34 [CLIENT_REQ()]-[INFO] | response
-no:
1664723518
-tag:
13:8|4:111|5:1521|6:2006041512|11:931|3:1|69:qt#true4#UTF-8markid#2e569f636dd44c3ca7d7983bd546f174|10:[]|connection_no:43|
-data:
check_tab_data|
                                            国泰君安期货                                            
                                                                    制表时间 Creation Date：20220929
----------------------------------------------------------------------------------------------------
                             交易结算单(盯市) Settlement Statement(MTM)                             
客户号 Client ID：  85197709          客户名称 Client Name：秦晓勇
日期 Date：20220929


尊敬的客户：为了保障您的保证金安全，中国期货市场监管中心要求期货公司发送每日的《交易结算单》，您可以按照您的用户名、密码，登陆www.cfmmc.com保证金监管中心网站，查询自己当日的《交易结算单》。 如有疑问，请立即向期货公司询问，我们将予以答复。

        资金状况   资金账号：85197709  币种：人民币  Account Summary AccountID：85197709 Currency：CNY    
----------------------------------------------------------------------------------------------------
期初结存 Balance b/f：                     4568.00  基础保证金 Initial Margin：                10.00
出 入 金 Deposit/Withdrawal：                 0.00  期末结存 Balance c/f：                   4704.00
平仓盈亏 Realized P/L：                     140.00  质 押 金 Pledge Amount：                    0.00
持仓盯市盈亏 MTM P/L：                        0.00  客户权益 Client Equity：                 4704.00
期权执行盈亏 Exercise P/L：                   0.00  货币质押保证金占用 FX Pledge Occ.：         0.00
手 续 费 Commission：                         4.00  保证金占用 Margin Occupied：                0.00
行权手续费 Exercise Fee：                     0.00  交割保证金 Delivery Margin：                0.00
交割手续费 Delivery Fee：                     0.00  多头期权市值 Market value(long)：           0.00
货币质入 New FX Pledge：                      0.00  空头期权市值 Market value(short)：          0.00
货币质出 FX Redemption：                      0.00  市值权益 Market value(equity)：          4704.00
质押变化金额 Chg in Pledge Amt：              0.00  可用资金 Fund Avail.：                   4704.00
权利金收入 Premium received：                 0.00  风 险 度 Risk Degree：                     0.00%
权利金支出 Premium paid：                     0.00  应追加资金 Margin Call：                    0.00
交割盈亏 Delivery P/L：                       0.00  货币质押变化金额 Chg in FX Pledge:          0.00
```

#### response包含error_no
```
2022-09-29 23:06:44,041 nioEventLoopGroup-27-64 [CLIENT_REQ()]-[INFO] | response
-no:
1664707476
-tag:
13:8|4:111|5:1004|6:2006041512|11:7584|3:1|69:4#UTF-8markid#eef7a45783304d66958614aae0276813|10:[]|connection_no:4275044481|
-data:
report_no|entrust_no|error_no|error_info|
18792|17636581|0|委托成功！|

```

### 出错的请求
#### 样例1
```
2023-03-21 15:36:36,389 nioEventLoopGroup-30-2 [CLIENT_REQ()]-[INFO] | response
-no:
1679365079
-tag:
13:8|4:111|5:1012|6:2006041512|11:2003|3:1|7:-1|19:-207|69:qt#true4#UTF-8markid#4b45404d02e74cb786c44cb13b296fad|10:[]|connection_no:1684013071|
-data:
error_no|error_info|
-207|未登录或登录失效|
```
#### 样例2
```
2023-03-21 14:17:00,360 nioEventLoopGroup-30-4 [CLIENT_REQ()]-[INFO] | response
-no:
1679363605
-tag:
13:8|3:1|4:111|5:28017|6:2006041511|11:504|12:1001|7:-1|69:qt#true4#1markid#f42a02e4e097430cab155fb61b6bf5e9|10:[]|connection_no:1735393334|
-data:
error_no|error_info|
-200|客户端认证失败|
```
#### 样例3
```

2023-03-03 11:13:31,292 nioEventLoopGroup-6-3 [CLIENT_REQ()]-[INFO] | response
-no:
1677798676
-tag:
13:8|3:1|4:111|5:1003|6:2006041512|11:564|12:1039|7:-1|69:qt#truevtk#eyJhbGciOiJTTd.MEQCIFPEMtThQ4P6eZH2gmbSoZwYfphW2Q6N3AILLRsAy81eAiAp4jS2agyV9bYUYbSQT1Xao2DlxrvYd00KIJ3AZPK7Lg4#1markid#25e0551a440c466eb99512a2407db24c|10:[]|connection_no:340367769|
-data:
error_no|error_info|
-1016|CTP:找不到合约|
```

### 日志关键字段
#### 按照`\n`切分成6段
切分后用于提取

#### request_time
请求发起的时间, 以`request`日志的时间为准

#### no+agent_hostname
同一个request和response一定出现在同一台机器上, 

其中`no`是请求号, 一天中可能会出现重复, 单个请求300秒超时, 从实际日志看1小时内不会出现重复

`agent_hostname`是categraf采集的元信息, 同一个`<no, agent_hostname>`对可以唯一确定两条日志

#### func_no
`tag`段的第4段, 以 `|5:`开头, 紧跟在后面的数字即是功能号 

#### fund_account
客户的账号ID, 不同的功能号日志`fund_account`的位置不同, 比如登录日志`fund_account`出现在response中(登录成功才会返回)

#### error_no
正常的请求一般不会有`errno_no`, 如果有也是0值; 错误的请求会有`error_no`和`error_info`, `error_no`一般是负数

#### request_time
根据`request`日志与`response`日志的时间字段计算差值, 就是request_time, 单位统一用毫秒

#### data
`data`字段除用于提取上述关键字段外, 不做其他处理, 原样写入ES, 页面上原样展示
有两个字段`request_data`和`response_data`分别代表请求参数和响应结果

#### tag
`tag`字段除用于提取`funcNo`外, 不做其他处理, 原样写入ES, 页面上原样展示, 仅保留`request`中的`tag`

### 字段映射
主要针对`funcNo`的展示, 非关键