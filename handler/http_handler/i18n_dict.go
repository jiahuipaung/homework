package http_handler

var i18ndict string = `
# 日志分析kvpair
allmatch:
  ch: 全部提取
  hk: 全部提取
  en: All Match
  jp: 全て抽出する
uripath:
  ch: URL归一化
  hk: URL歸一化
  en: URL Normalization
  jp: URL正規化
regexp:
  ch: 正则提取
  hk: 正則提取
  en: Regexp
  jp: 正規表現による抽出
desensitize:
  ch: 数据脱敏
  hk: 數據脫敏
  en: Data Desensitize
  jp: データ脱敏
golden_metrics:
  ch: 黄金指标模板
  hk: 黃金指標模板
  en: Golden Metrics
  jp: ゴールデン指標
logx_golden_succrate:
  ch: 成功率(%%)
  hk: 成功率(%%)
  en: Success Rate
  jp: 成功率
logx_golden_count:
  ch: 流量
  hk: 流量
  en: Traffic
  jp: トラフィック
logx_golden_latency:
  ch: 延时
  hk: 延時
  en: Latency
  jp: 遅延
logx_golden_status:
  ch: 响应码统计(min)
  hk: 響應碼統計(min)
  en: Traffic By Status
  jp: レスポンスコード統計
logx_golden_others:
  ch: 其他
  hk: 其他
  en: Others
  jp: その他
customize:
  ch: 自定义观测值
  hk: 自定義觀測值
  en: Custom Observations
  jp: カスタム観測値
logx_count:
  ch: 日志行数
  hk: 日誌行數
  en: Log Line Count
  jp: ログ行数
logx_max:
  ch: 最大值
  hk: 最大值
  en: Max
  jp: 最大値
logx_min:
  ch: 最小值
  hk: 最小值
  en: Min
  jp: 最小値
logx_sum:
  ch: 和值
  hk: 和值
  en: Sum
  jp: 合計
logx_avg:
  ch: 平均值
  hk: 平均值
  en: Average
  jp: 平均値
logx_p50:
  ch: 50分位值
  hk: 50分位值
  en: 50th Percentile
  jp: 50th Percentile
logx_p90:
  ch: 90分位值
  hk: 90分位值
  en: 90th Percentile
  jp: 90th Percentile
logx_p95:
  ch: 95分位值
  hk: 95分位值
  en: 95th Percentile
  jp: 95th Percentile
logx_p98:
  ch: 98分位值
  hk: 98分位值
  en: 98th Percentile
  jp: 98th Percentile
logx_p99:
  ch: 99分位值
  hk: 99分位值
  en: 99th Percentile
  jp: 99th Percentile
logx_p99.9:
  ch: 99.9分位值
  hk: 99.9分位值
  en: 99.9th Percentile
  jp: 99.9th Percentile
time_us:
  ch: 微秒
  hk: 微秒
  en: Microsecond
  jp: マイクロ秒
time_ms:
  ch: 毫秒
  hk: 毫秒
  en: Millisecond
  jp: ミリ秒
time_s:
  ch: 秒
  hk: 秒
  en: Second
  jp: 秒
time_min:
  ch: 分钟
  hk: 分鐘
  en: Minute
  jp: 分
time_hour:
  ch: 小时
  hk: 小時
  en: Hour
  jp: 時間
text:
  ch: 字符串
  hk: 字符串
  en: text
  jp: 文字列
float:
  ch: 浮点数
  hk: 浮點數
  en: float
  jp: 浮動小数点数
long:
  ch: 整数
  hk: 整數
  en: interger
  jp: 整数
date:
  ch: 日期
  hk: 日期
  en: date
  jp: 日付
object:
  ch: JSON结构体
  hk: JSON結構體
  en: JSON object
  jp: JSON オブジェクト
array:
  ch: 数组
  hk: 数組
  en: array
  jp: 配列
suffix_default:
  ch: 平台自动管理
  hk: 平台自動管理
  en: Platform Auto-Managed
  jp: プラットフォーム自動管理
suffix_none:
  ch: 不分割
  hk: 不分割
  en: None Suffix
  jp: 分割しない
suffix_hourly:
  ch: 小时
  hk: 小時
  en: Hourly
  jp: 時間
suffix_daily:
  ch: 天
  hk: 天
  en: Daily
  jp: 日
suffix_weekly:
  ch: 周
  hk: 周
  en: Weekly
  jp: 週
retention_0:
  ch: 永久保存
  hk: 永久保存
  en: Never Expired
  jp: 永久保存
retention_2h:
  ch: 2小时
  hk: 2小時
  en: 2 Hours
  jp: 2 時間
retention_1d:
  ch: 1天
  hk: 1天
  en: 1 Day
  jp: 1 日
retention_2d:
  ch: 2天
  hk: 2天
  en: 2 Days
  jp: 2 日
retention_5d:
  ch: 5天
  hk: 5天
  en: 5 Days
  jp: 5 日
retention_1w:
  ch: 1周
  hk: 1周
  en: 1 Week
  jp: 1 週間
retention_2w:
  ch: 2周
  hk: 2周
  en: 2 Weeks
  jp: 2 週間
retention_30d:
  ch: 1个月
  hk: 1個月
  en: 1 Month
  jp: 1 ヶ月
retention_92d:
  ch: 3个月
  hk: 3個月
  en: 3 Month
  jp: 3 ヶ月
retention_185d:
  ch: 6个月
  hk: 6個月
  en: 6 Month
  jp: 6 ヶ月
logx_gonx:
  ch: log_format
  hk: log_format
  en: log_format
  jp: log_format
logx_input_gonx:
  ch: $remote_ip || [$local_time] || "$request" || $status || $body_byte_send || $http_refer
  hk: $remote_ip || [$local_time] || "$request" || $status || $body_byte_send || $http_refer
  en: $remote_ip || [$local_time] || "$request" || $status || $body_byte_send || $http_refer
  jp: $remote_ip || [$local_time] || "$request" || $status || $body_byte_send || $http_refer
logx_desc_gonx:
  ch: "日志样例: 62.173.145.171 || [01/Jan/2020:17:23:54 +0800] || \"GET /vvx/000000000000.cfg HTTP/1.1\" || 404 || 169 || \"-\"\n参数配置: $remote_ip || [$local_time] || \"$request\" || $status || $body_byte_send || $http_refer\n"
  hk: "日誌樣例: 62.173.145.171 || [01/Jan/2020:17:23:54 +0800] || \"GET /vvx/000000000000.cfg HTTP/1.1\" || 404 || 169 || \"-\"\n參數配置: $remote_ip || [$local_time] || \"$request\" || $status || $body_byte_send || $http_refer\n"
  en: "Sample: 62.173.145.171 || [01/Jan/2020:17:23:54 +0800] || \"GET /vvx/000000000000.cfg HTTP/1.1\" || 404 || 169 || \"-\"\nParams: $remote_ip || [$local_time] || \"$request\" || $status || $body_byte_send || $http_refer\n"
  jp: "ログサンプル: 62.173.145.171 || [01/Jan/2020:17:23:54 +0800] || \"GET /vvx/000000000000.cfg HTTP/1.1\" || 404 || 169 || \"-\"\nパラメータ設定: $remote_ip || [$local_time] || \"$request\" || $status || $body_byte_send || $http_refer\n"
logx_grok:
  ch: grok
  hk: grok
  en: grok
  jp: grok
logx_input_grok:
  ch: '%%{IPORHOST:remote_addr} - %%{DATA:remote_user} \[%%{HTTPDATE:time_local}\] \"%%{WORD:method} %%{URIPATH:request}(?:%%{URIPARAM:params})? %%{DATA:protocol}\" %%{NUMBER:status} %%{NUMBER:body_bytes_sent} %%{NUMBER:request_time}'
  hk: '%%{IPORHOST:remote_addr} - %%{DATA:remote_user} \[%%{HTTPDATE:time_local}\] \"%%{WORD:method} %%{URIPATH:request}(?:%%{URIPARAM:params})? %%{DATA:protocol}\" %%{NUMBER:status} %%{NUMBER:body_bytes_sent} %%{NUMBER:request_time}'
  en: '%%{IPORHOST:remote_addr} - %%{DATA:remote_user} \[%%{HTTPDATE:time_local}\] \"%%{WORD:method} %%{URIPATH:request}(?:%%{URIPARAM:params})? %%{DATA:protocol}\" %%{NUMBER:status} %%{NUMBER:body_bytes_sent} %%{NUMBER:request_time}'
  jp: '%%{IPORHOST:remote_addr} - %%{DATA:remote_user} \[%%{HTTPDATE:time_local}\] \"%%{WORD:method} %%{URIPATH:request}(?:%%{URIPARAM:params})? %%{DATA:protocol}\" %%{NUMBER:status} %%{NUMBER:body_bytes_sent} %%{NUMBER:request_time}'
logx_desc_grok:
  ch: '日志样例: 62.173.145.171 - - [16/Apr/2024:13:11:45 +0800] \"GET /api/v1/2/pure HTTP/1.1\" 200 196 2.948NEWLINE参数配置: %%{IPORHOST:remote_addr} - %%{DATA:remote_user} \[%%{HTTPDATE:time_local}\] \"%%{WORD:method} %%{URIPATH:request}(?:%%{URIPARAM:params})? %%{DATA:protocol}\" %%{NUMBER:status} %%{NUMBER:body_bytes_sent} %%{NUMBER:request_time}'
  hk: '日誌樣例: 62.173.145.171 - - [16/Apr/2024:13:11:45 +0800] \"GET /api/v1/2/pure HTTP/1.1\" 200 196 2.948NEWLINE參數配置: %%{IPORHOST:remote_addr} - %%{DATA:remote_user} \[%%{HTTPDATE:time_local}\] \"%%{WORD:method} %%{URIPATH:request}(?:%%{URIPARAM:params})? %%{DATA:protocol}\" %%{NUMBER:status} %%{NUMBER:body_bytes_sent} %%{NUMBER:request_time}'
  en: 'Sample: 62.173.145.171 - - [16/Apr/2024:13:11:45 +0800] \"GET /api/v1/2/pure HTTP/1.1\" 200 196 2.948NEWLINEParams: %%{IPORHOST:remote_addr} - %%{DATA:remote_user} \[%%{HTTPDATE:time_local}\] \"%%{WORD:method} %%{URIPATH:request}(?:%%{URIPARAM:params})? %%{DATA:protocol}\" %%{NUMBER:status} %%{NUMBER:body_bytes_sent} %%{NUMBER:request_time}'
  jp: 'ログサンプル: 62.173.145.171 - - [16/Apr/2024:13:11:45 +0800] \"GET /api/v1/2/pure HTTP/1.1\" 200 196 2.948NEWLINEパラメータ設定: %%{IPORHOST:remote_addr} - %%{DATA:remote_user} \[%%{HTTPDATE:time_local}\] \"%%{WORD:method} %%{URIPATH:request}(?:%%{URIPARAM:params})? %%{DATA:protocol}\" %%{NUMBER:status} %%{NUMBER:body_bytes_sent} %%{NUMBER:request_time}'
logx_regexp:
  ch: 正则提取
  hk: 正則提取
  en: regexp
  jp: regexp
logx_input_regexp_0:
  ch: "remote_ip:(?P<remote_ip>[^|]+) || time_local:(?P<time_local >[^|]+) || request:(?P<request>[^|]+) || status:(?P<status>[^|]+)"
  hk: "remote_ip:(?P<remote_ip>[^|]+) || time_local:(?P<time_local >[^|]+) || request:(?P<request>[^|]+) || status:(?P<status>[^|]+)"
  en: "remote_ip:(?P<remote_ip>[^|]+) || time_local:(?P<time_local >[^|]+) || request:(?P<request>[^|]+) || status:(?P<status>[^|]+)"
  jp: "remote_ip:(?P<remote_ip>[^|]+) || time_local:(?P<time_local >[^|]+) || request:(?P<request>[^|]+) || status:(?P<status>[^|]+)"
logx_desc_regexp:
  ch: "日志样例: remote_ip:62.173.145.171 || local_time:[01/Jan/2020:17:23:54 +0800] || request:\"GET /vvx/000000000000.cfg HTTP/1.1\" || status:404\n参数配置: remote_ip:(?P<remote_ip>[^|]+) || time_local:(?P<time_local >[^|]+) || request:?P<request>[^|]+) || status:(?P<status>[^|]+)\n"
  hk: "日誌樣例: remote_ip:62.173.145.171 || local_time:[01/Jan/2020:17:23:54 +0800] || request:\"GET /vvx/000000000000.cfg HTTP/1.1\" || status:404\n參數配置: remote_ip:(?P<remote_ip>[^|]+) || time_local:(?P<time_local >[^|]+) || request:?P<request>[^|]+) || status:(?P<status>[^|]+)\n"
  en: "Sample: remote_ip:62.173.145.171 || local_time:[01/Jan/2020:17:23:54 +0800] || request:\"GET /vvx/000000000000.cfg HTTP/1.1\" || status:404\nParams: remote_ip:(?P<remote_ip>[^|]+) || time_local:(?P<time_local >[^|]+) || request:?P<request>[^|]+) || status:(?P<status>[^|]+)\n"
  jp: "ログサンプル: remote_ip:62.173.145.171 || local_time:[01/Jan/2020:17:23:54 +0800] || request:\"GET /vvx/000000000000.cfg HTTP/1.1\" || status:404\nパラメータ設定: remote_ip:(?P<remote_ip>[^|]+) || time_local:(?P<time_local >[^|]+) || request:?P<request>[^|]+) || status:(?P<status>[^|]+)\n"
logx_split:
  ch: 分隔符
  hk: 分隔符
  en: split
  jp: split
logx_input_split:
  ch: "一个字符或多个字符分割: ,;:等; 正则分割: /[1-9]+/, 必须由前后的'/'包裹"
  hk: "一個字符或多個字符分割: ,;:等; 正則分割: /[1-9]+/, 必須由前後的'/'包裹"
  en: "single string or regexp format are both supported"
  jp: "文字または複数文字で分割：,;:など；正規表現で分割：/[1-9]+/、必ず前後に'/'をつける"
logx_desc_split:
  ch: "日志样例: 62.173.145.171 || [01/Jan/2020:17:23:54 +0800] || \"GET /vvx/000000000000.cfg HTTP/1.1\" || 404 || 169 || \"-\"\n参数配置: ||\n"
  hk: "日誌樣例: 62.173.145.171 || [01/Jan/2020:17:23:54 +0800] || \"GET /vvx/000000000000.cfg HTTP/1.1\" || 404 || 169 || \"-\"\n參數配置: ||\n"
  en: "Sample: 62.173.145.171 || [01/Jan/2020:17:23:54 +0800] || \"GET /vvx/000000000000.cfg HTTP/1.1\" || 404 || 169 || \"-\"\nParams: ||\n"
  jp: "ログサンプル: 62.173.145.171 || [01/Jan/2020:17:23:54 +0800] || \"GET /vvx/000000000000.cfg HTTP/1.1\" || 404 || 169 || \"-\"\nパラメータ設定: ||\n"
logx_json:
  ch: JSON反序列化
  hk: JSON反序列化
  en: codec json
  jp: JSON 逆シリアル化
logx_input_json:
  ch: "{}"
  hk: "{}"
  en: "{}"
  jp: "{}"
logx_desc_json:
  ch: "日志文本必须是json字符串\n"
  hk: "日誌文本必須是json字符串\n"
  en: "log text must be json string\n"
  jp: "ログテキストは必ず JSON 文字列でなければならない\n"
logx_key_value:
  ch: key_value提取
  hk: key_value提取
  en: key_value
  jp: key_value抽出
logx_input_key_value:
  ch: "{\"field_delimiter\":\",\", \"key_value_delimiter\":\"=\"}"
  hk: "{\"field_delimiter\":\",\", \"key_value_delimiter\":\"=\"}"
  en: "{\"field_delimiter\":\",\", \"key_value_delimiter\":\"=\"}"
  jp: "{\"field_delimiter\":\",\", \"key_value_delimiter\":\"=\"}"
logx_desc_key_value:
  ch: "日志样例: remote_ip:62.173.145.171 || local_time:[01/Jan/2020:17:23:54 +0800] || request:\"GET /vvx/000000000000.cfg HTTP/1.1\" || status:404\n参数配置: {\"field_delimiter\":\" || \", \"key_value_delimiter\":\":\"}\n"
  hk: "日誌樣例: remote_ip:62.173.145.171 || local_time:[01/Jan/2020:17:23:54 +0800] || request:\"GET /vvx/000000000000.cfg HTTP/1.1\" || status:404\n參數配置: {\"field_delimiter\":\" || \", \"key_value_delimiter\":\":\"}\n"
  en: "Sample: remote_ip:62.173.145.171 || local_time:[01/Jan/2020:17:23:54 +0800] || request:\"GET /vvx/000000000000.cfg HTTP/1.1\" || status:404\nParams: {\"field_delimiter\":\" || \", \"key_value_delimiter\":\":\"}\n"
  jp: "ログサンプル: remote_ip:62.173.145.171 || local_time:[01/Jan/2020:17:23:54 +0800] || request:\"GET /vvx/000000000000.cfg HTTP/1.1\" || status:404\nパラメータ設定: {\"field_delimiter\":\" || \", \"key_value_delimiter\":\":\"}\n"
logx_desc_geoip:
  ch: "日志文本必须是公网IP\n"
  hk: "日誌文本必須是公網IP\n"
  en: "log text must be public IP string\n"
  jp: "ログテキストは必ずパブリック IP でなければならない\n"
logx_otel_trace_spans:
  ch: OTEL Trace Spans解析
  hk: OTEL Trace Spans解析
  en: decode otel-trace-spans
  jp: OTEL Trace Spans解析する
logx_input_otel_trace_spans:
  ch: "{}"
  hk: "{}"
  en: "{}"
  jp: "{}"
logx_desc_otel_trace_spans:
  ch: "日志文本必须包含\"resourceSpans\"\n"
  hk: "日誌文本必須包含\"resourceSpans\"\n"
  en: "log text must contain \"resourceSpans\"\n"
  jp: "ログテキストには必ず「resourceSpans」を含まなければならない\n"
logx_contains:
  ch: 包含子串
  hk: 包含子串
  en: Contains
  jp: 部分文字列を含む
logx_not_contains:
  ch: 排除子串
  hk: 排除子串
  en: Not Contains
  jp: 部分文字列を除外する
logx_match:
  ch: 包含正则
  hk: 包含正則
  en: Regexp Match
  jp: 正規表現を含む
logx_not_match:
  ch: 排除正则
  hk: 排除正則
  en: Regexp Not Match
  jp: 正規表現を除外する
logx_strip_ansi:
  ch: 移除ANSI转义序列
  hk: 移除ANSI轉義序列
  en: Strip ANSI Escape Codes
  jp: ANSI エスケープシーケンスを削除する
logx_format_to_json:
  ch: 转化为JSON格式
  hk: 轉化為JSON格式
  en: Convert to JSON Format
  jp: JSON 形式に変換する
logx_decompress:
  ch: 解压缩
  hk: 解壓縮
  en: Decompress
  jp: 展開する
logx_split_json_array:
  ch: 遍历JSON数组
  hk: 遍歷JSON數組
  en: Range JSON Array
  jp: JSON 配列を反復処理する
logx_format:
  ch: 格式化
  hk: 格式化
  en: Format
  jp: フォーマットする
logx_desc_format:
  ch: "值的格式化, 比如字符串转化为数字或日期, 字符串的子串提取等"
  hk: "值的格式化, 比如字符串轉化為數字或日期, 字符串的子串提取等"
  en: "find a substring and transform to datetime or interger or float"
  jp: "値の型変換、たとえば文字列から数字や日付への変換、文字列の部分文字列抽出など"
logx_clone:
  ch: 克隆
  hk: 克隆
  en: Clone
  jp: クローンする
logx_desc_clone:
  ch: "克隆一个字段, 比如克隆一个字符串或子结构体"
  hk: "克隆一個字段, 比如克隆一個字符串或子結構體"
  en: "clone a field, like clone a string or sub-object"
  jp: "フィールドを複製します。たとえば、文字列を複製します、またはサブオブジェクトを複製します"
logx_rename:
  ch: 重命名
  hk: 重命名
  en: Rename
  jp: 名前を変更する
logx_desc_rename:
  ch: "修改字段的名称, 比如原名称是a.b.c, 重命名后是a.b.d"
  hk: "修改字段名稱, 比如原名稱是a.b.c, 重命名後是a.b.d"
  en: "rename a field, like rename a.b.c to a.b.d"
  jp: "フィールド名を変更します。たとえば、a.b.c を a.b.d に変更します"
logx_delete:
  ch: 删除
  hk: 删除
  en: Delete
  jp: 削除する
logx_desc_delete:
  ch: "从原始日志中删除一个字段"
  hk: "從原始日誌中刪除一個字段"
  en: "delete a field from original log"
  jp: "元のログからフィールドを削除します"
logx_append:
  ch: 新增
  hk: 新增
  en: Add
  jp: 新規追加する
logx_desc_append:
  ch: "添加原始日志中不存在的字段, 值类型可以自定义"
  hk: "添加原始日誌中不存在的字段, 值類型可以自定義"
  en: "add a new item with self-defined path"
  jp: "元のログに存在しないフィールドを追加します。値の型はカスタマイズ可能です"
logx_clone_and_format:
  ch: 克隆并格式化
  hk: 克隆並格式化
  en: Clone and Format
  jp: クローンしてフォーマットする
logx_desc_clone_and_format:
  ch: "从原字段克隆, 并对新字段进行格式化, 比如克隆一个字符串并转化为数字"
  hk: "從原字段克隆, 並對新字段進行格式化, 比如克隆一個字符串並轉化為數字"
  en: "clone a field and format it, like clone a string and transform to number"
  jp: "フィールドを複製してフォーマットします。たとえば、文字列を複製して数字に変換します"
logx_allmatch:
  ch: 全部提取
  hk: 全部提取
  en: All Match
  jp: 全て抽出する
logx_regexp:
  ch: 正则提取
  hk: 正則提取
  en: Regexp
  jp: 正規表現による抽出
logx_desensitize:
  ch: 数据脱敏
  hk: 數據脫敏
  en: Data Desensitize
  jp: データ脱敏
logx_desc_desensitize:
  ch: "隐藏日志中的敏感信息, 比如手机号/邮箱/身份证号等"
  hk: "隱藏日誌中的敏感信息, 比如手機號/郵箱/身份證號等"
  en: "Hide sensitive information in log, like phone number/email/id card number etc."
  jp: "ログから敏感情報を隠す, たとえば電話番号/メールアドレス/身分証番号など"
logx_regexp_mapping:
  ch: 正则映射
  hk: 正則映射
  en: Regexp Mapping
  jp: 正規表現による映射
logx_desc_regexp_mapping:
  ch: "通过映射规则, 将日志中的内容转化成其他文本"
  hk: "通過映射規則, 將日誌中的內容轉化成其他文本"
  en: "Map the content of log to other text through mapping rule"
  jp: "マッピングルールに従って、ログの内容を他のテキストに変換します"
logx_uripath:
  ch: URL归一化
  hk: URL歸一化
  en: URL Normalization
  jp: URL正規化
logx_desc_uripath:
  ch: "包括静态路径丢弃/restful参数替换等, 比如/api/v1/order/123456 转换为 /api/v1/order/:id"
  hk: "包括靜態路徑丟棄/restful參數替換等, 比如/api/v1/order/123456 轉換為 /api/v1/order/:id"
  en: "include static path discard/restful parameter replacement, etc., like /api/v1/order/123456 to /api/v1/order/:id"
  jp: "静的パスの破棄/restfulパラメータの置換など、たとえば /api/v1/order/123456 を /api/v1/order/:id に変換します"
logx_desensitize_phone:
  ch: 手机号码
  hk: 手機號碼
  en: Phone Number
  jp: 電話番号
logx_desc_desensitize_phone:
  ch: "样例: 13800138000 转换为 138****8000"
  hk: "樣例: 13800138000 轉換為 138****8000"
  en: "Sample: 13800138000 to 138****8000"
  jp: "サンプル: 13800138000 を 138****8000 に変換します"
logx_desensitize_email:
  ch: 邮箱地址
  hk: 郵箱地址
  en: Email Address
  jp: メールアドレス
logx_desc_desensitize_email:
  ch: "样例: test@test.com 转换为 ****@test.com"
  hk: "樣例: test@test.com 轉換為 ****@test.com"
  en: "Sample: test@test.com to ****@test.com"
  jp: "サンプル: test@test.com を ****@test.com に変換します"
logx_desensitize_ip:
  ch: IP地址
  hk: IP地址
  en: IP Address
  jp: IPアドレス
logx_desc_desensitize_ip:
  ch: "样例: 192.168.1.1 转换为 192.****"
  hk: "樣例: 192.168.1.1 轉換為 192.****"
  en: "Sample: 192.168.1.1 to 192.****"
  jp: "サンプル: 192.168.1.1 を 192.**** に変換します"
logx_desensitize_bank_card:
  ch: 银行卡号
  hk: 銀行卡號
  en: Bank Card Number
  jp: 銀行カード番号
logx_desc_desensitize_bank_card:
  ch: "样例: 1234567890123456789 转换为 ****56789"
  hk: "樣例: 1234567890123456789 轉換為 ****56789"
  en: "Sample: 1234567890123456789 to ****56789"
  jp: "サンプル: 1234567890123456789 を ****56789 に変換します"
logx_desensitize_id_card:
  ch: 身份证号
  hk: 身份證號
  en: ID Card Number
  jp: 身分証番号
logx_desc_desensitize_id_card:
  ch: "样例: 1234567890123456789 转换为 1234****"
  hk: "樣例: 1234567890123456789 轉換為 1234****"
  en: "Sample: 1234567890123456789 to 1234****"
  jp: "サンプル: 1234567890123456789 を 1234**** に変換します"
logx_desensitize_access_key:
  ch: 访问密钥
  hk: 訪問密鑰
  en: Access Key
  jp: アクセスキー
logx_desc_desensitize_access_key:
  ch: "样例: aBcd9efghIjklmnOpqrstuvwxyz 转换为 aBcd****"
  hk: "樣例: aBcd9efghIjklmnOpqrstuvwxyz 轉換為 aBcd****"
  en: "Sample: aBcd9efghIjklmnOpqrstuvwxyz to aBcd****"
  jp: "サンプル: aBcd9efghIjklmnOpqrstuvwxyz を aBcd**** に変換します"
logx_desensitize_keep_first_last:
  ch: 首尾保留
  hk: 首尾保留
  en: Keep First and Last
  jp: 最初と最後を保持する
logx_desc_desensitize_keep_first_last:
  ch: "样例: 1234567890 转换为 1****0"
  hk: "樣例: 1234567890 轉換為 1****0"
  en: "Sample: 1234567890 to 1****0"
  jp: "サンプル: 1234567890 を 1****0 に変換します"
logx_desensitize_keep_last_four:
  ch: 末尾保留
  hk: 末尾保留
  en: Keep Last Four
  jp: 最後の4文字を保持する
logx_desc_desensitize_keep_last_four:
  ch: "样例: 1234567890 转换为 ****7890"
  hk: "樣例: 1234567890 轉換為 ****7890"
  en: "Sample: 1234567890 to ****7890"
  jp: "サンプル: 1234567890 を ****7890 に変換します"
logx_desensitize_md5:
  ch: MD5编码
  hk: MD5編碼
  en: MD5 Encoding
  jp: MD5 エンコーディング
logx_desc_desensitize_md5:
  ch: "样例: 1234567890 转换为 e807f1fcf82d132f9bb018ca6738a19f"
  hk: "樣例: 1234567890 轉換為 e807f1fcf82d132f9bb018ca6738a19f"
  en: "Sample: 1234567890 to e807f1fcf82d132f9bb018ca6738a19f"
  jp: "サンプル: 1234567890 を e807f1fcf82d132f9bb018ca6738a19f に変換します"
logx_desensitize_custom:
  ch: 自定义规则
  hk: 自定義規則
  en: Custom Rule
  jp: カスタマイズされたルール
logx_desc_desensitize_custom:
  ch: "样例: 根据正则表达式(\\d{4})\\d{2}(\\d{4})和替换规则$1****$2, 将1234567890 转换为 1234****7890"
  hk: "樣例: 根據正則表達式(\\d{4})\\d{2}(\\d{4})和替換規則$1****$2, 將1234567890 轉換為 1234****7890"
  en: "Sample: 1234567890 to 1234****7890, regexp: (\\d{4})\\d{2}(\\d{4}), replace: $1****$2"
  jp: "サンプル: 正規表現 (\\d{4})\\d{2}(\\d{4}) と置換規則 $1****$2 に従って、1234567890 を 1234****7890 に変換します"
logx_submatch:
  ch: 格式化
  hk: 格式化
  en: Format
  jp: フォーマットする
logx_desc_submatch:
  ch: "格式化: 值的格式化, 比如字符串转化为数字或日期, 字符串的子串提取等\n\n"
  hk: "格式化: 值的格式化, 比如字符串轉化為數字或日期, 字符串的子串提取等\n\n"
  en: "Format: find a substring and transform to datetime or interger or float\n\n"
  jp: "フォーマット：値の型変換、たとえば文字列から数字や日付への変換、文字列の部分文字列抽出など\n\n"
logx_graft:
  ch: 移动
  hk: 移動
  en: Move
  jp: 移動する
logx_desc_graft:
  ch: "移动: 改变源字段的位置, 其中__root__是系统保留字, 代表移动到根节点\n\n"
  hk: "移動: 改變源字段的位置, 其中__root__是系統保留字, 代表移動到根節點\n\n"
  en: "Move-to: change the target value's path\n\n"
  jp: "移動: ソースフィールドの位置を変更します。ここで、「__root__」はシステム予約語で、ルートノードに移動することを表します\n\n"
logx_submatch_graft:
  ch: 格式化并移动
  hk: 格式化並移動
  en: Format-And-Move-to
  jp: フォーマットして移動する
logx_desc_submatch_graft:
  ch: "格式化并移动: 值类型转换成功后改变源字段的位置\n\n"
  hk: "格式化並移動: 值類型轉換成功後改變源字段的位置\n\n"
  en: "Format-And-Move: change the substring's type and path\n\n"
  jp: "フォーマットして移動する：値の型変換に成功した後、ソースフィールドの位置を変更する\n\n"
logx_remove:
  ch: 删除
  hk: 删除
  en: Delete
  jp: 削除する
logx_desc_remove:
  ch: "删除: 从原始日志中去掉某个字段或某个叶子节点\n\n"
  hk: "删除: 從原始日誌中去掉某個字段或某個葉子節點\n\n"
  en: "Delete: delete an old item by existed path\n\n"
  jp: "削除：元のログからあるフィールドまたはあるリーフノードを取り除きます\n\n"
logx_origin:
  ch: 保持原状
  hk: 保持原狀
  en: Remain
  jp: そのまま保持する
logx_desc_origin:
  ch: "保持原状: 原始日志保持不变\n\n"
  hk: "保持原状: 原始日誌保持不變\n\n"
  en: "Remain: do nothing and leave it as it is\n\n"
  jp: "そのまま保持する：元のログは変更せずそのまま維持されます\n\n"
logx_input_text:
  ch: 请输入字符串
  hk: 請輸入字符串
  en: Please input a string
  jp: 文字列を入力してください
logx_input_regexp:
  ch: 请输入正则
  hk: 請輸入正則
  en: Please input a regexp
  jp: 正規表現を入力してください
logx_aggr_none:
  ch: 无
  hk: 無
  en: None
  jp: なし
logx_aggr_url_trim:
  ch: URL参数截断
  hk: URL參數截斷
  en: Trim params in query string
  jp: URL パラメータを切り捨てる
logx_aggr_restful_trim:
  ch: RESTful参数替换
  hk: RESTful參數替換
  en: Replace params in RESTful API
  jp: RESTful パラメータを置き換える
`
