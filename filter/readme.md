# 词表映射包

词表映射包是一个用于根据预定义的词表将输入字符串映射为对应的词表中的词的工具包。

## 主要函数介绍

### `ParseString` 函数

`ParseString` 实现filter接口，负责将给定的输入字符串根据预定义的翻译映射转换成对应的翻译输出。这个函数首先会查找是否存在输入字符串的翻译映射，如果找到了，它将返回映射中定义的翻译字符串；如果没有找到，它将返回原始输入字符串。

#### 参数

- `origin string`：原始的待翻译字符串。这是函数的输入值，代表需要被翻译或查找翻译映射的文本。

#### 返回值

- `string`：翻译后的字符串或原始输入字符串。如果在翻译映射中找到了`origin`的对应翻译，则返回该翻译；如果没有找到对应翻译，则返回`origin`本身。
- `error`：处理过程中遇到的任何错误。正常情况下返回`nil`。

### `ParseTranslationMap` 函数

`ParseTranslationMap` 函数用于解析包含键值对翻译映射的字节切片数据。它支持两种格式的数据：标准CSV格式和自定义分隔符格式。当分隔符为空字符串时，默认解析为CSV格式；否则，将使用指定的分隔符来解析数据。这个函数将解析的数据存储在一个映射中，以便`ParseString`函数进行查询和翻译。

#### 参数说明

- `data []byte`：包含翻译映射的字节切片，可以从文件、网络或其他源获取。
- `Delimiter string`（在`TranslateArgument`结构体中定义）：用于指定自定义分隔符。如果为空，则假定数据为CSV格式。

#### 返回值

- `map[string]string`：一个从字符串映射到字符串的字典，其中包含了从`data`参数中解析出的所有键值对翻译映射。
- `error`：处理过程中遇到的任何错误。如果解析过程顺利完成，则返回`nil`；否则，返回一个描述错误详情的错误对象。


# KeyValue解析器包
KeyValue解析器用于解析包含键值对的文本数据。

## 支持的模式
- **PresetKeyValueModeTable**：解析横向表格结构的数据，键位于表头。
- **PresetKeyValueModeVerticalTable**：解析纵向表格结构的数据，键和值分布在不同的列中。
- **PresetKeyValueModeDelimiterInValue**：解析值中包含分隔符的数据。

## 配置参数

在初始化解析器时，可以使用`KeyValueArgument`结构体提供多项配置：
- `FieldDelimiter`、`KeyValueDelimiter`：设置字段和键值对的分隔符。
- `Mode`：指定解析模式。
- 表格相关参数：例如`TableRowDelimiter`、`TableKeyDelimiter`、`TableValueDelimiter`等，用于指定解析表格数据的规则。

## `ParseKeyValue` 函数
`ParseKeyValue` 函数是设计用来解析包含键值对的字符串的。它支持自定义字段分隔符和键值分隔符，可以处理在双引号或单引号内的值，允许这些值中包含字段分隔符。

### 参数说明
- `ret map[string]interface{}`：用于存储解析后的键值对，键为字符串类型，值为接口类型，可以存储任意类型的值。
- `origin string`：原始的待解析字符串。
- `fieldDelimiter string`：字段分隔符，用于分隔不同的键值对。
- `keyValueDelimiter string`：键值分隔符，用于分隔键和值。
- 
### 错误处理
该函数在解析过程中遇到错误会返回相应的错误信息，例如：
- 遇到未闭合的引号时返回`"unterminated quote"`错误。
- 当键后直接跟了字段分隔符但没有找到对应的值时，返回`"missing value for key"`错误。
