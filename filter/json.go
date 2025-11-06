package filter

import (
	"bytes"
	"encoding/json"

	"github.com/flashcatcloud/fc-stash/types"
)

var (
	PresetJson = "json"
)

func init() {
	registerParser(PresetJson, NewJsonParser)
}

type JsonParser struct {
	needOrigin bool
	args       JsonArgument
}

// 暂时不支持正则
type JsonArgument struct {
	StripControl bool `json:"strip_control"` // 移除json中的无效字符 0x00~0x20
}

// json反序列化原始文本直接丢掉
func NewJsonParser(format string, needOrigin bool) (types.TextParser, error) {
	var args JsonArgument
	if len(format) > 0 && json.Valid([]byte(format)) {
		// 参数解析失败, 不影响json反序列化的执行
		_ = json.Unmarshal([]byte(format), &args)
	}
	return &JsonParser{
		needOrigin: false,
		args:       args,
	}, nil
}

func (p *JsonParser) Name() string {
	return PresetJson
}

func (p *JsonParser) Parse(origin string) (map[string]interface{}, error) {
	if len(origin) == 0 {
		return nil, nil
	}
	if p.args.StripControl {
		buf := bytesPool.Get().(*bytes.Buffer)
		buf.Reset()
		defer bytesPool.Put(buf)
		for _, ch := range []byte(origin) {
			if ch < 0x20 {
				continue
			}
			buf.WriteByte(ch)
		}
		ret := make(map[string]interface{})
		if err := json.Unmarshal(buf.Bytes(), &ret); err != nil {
			return nil, err
		}
		if p.needOrigin {
			ret[types.LogParserOriginText] = origin
		}
		return ret, nil
	}
	ret := make(map[string]interface{})
	if err := json.Unmarshal([]byte(origin), &ret); err != nil {
		return nil, err
	}
	if p.needOrigin {
		ret[types.LogParserOriginText] = origin
	}
	return ret, nil
}
