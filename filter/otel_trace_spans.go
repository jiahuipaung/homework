package filter

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/flashcatcloud/fc-stash/types"
)

var (
	PresetOTELTraceSpans = "otel_trace_spans"
)

func init() {
	registerParser(PresetOTELTraceSpans, NewOTELTraceSpansParser)
}

type OTELTraceSpansParser struct{}

func NewOTELTraceSpansParser(format string, needOrigin bool) (types.TextParser, error) {
	return &OTELTraceSpansParser{}, nil
}

func (p *OTELTraceSpansParser) Name() string {
	return PresetOTELTraceSpans
}

func (p *OTELTraceSpansParser) Parse(origin string) (map[string]interface{}, error) {
	if len(origin) == 0 {
		return nil, nil
	}
	items := make(map[string]interface{})
	if err := json.Unmarshal([]byte(origin), &items); err != nil {
		return nil, err
	}
	_spans, found := items["resourceSpans"]
	if !found {
		return nil, errors.New("key resourceSpans not found")
	}
	cansink := [3]bool{}
	spans, ok := _spans.([]interface{})
	if ok {
		for _, span := range spans {
			spanmap, ok := span.(map[string]interface{})
			if ok {
				var servicename string
				if _, found := spanmap["resource"]; found {
					resourcemap, ok := spanmap["resource"].(map[string]interface{})
					if !ok {
						continue
					}
					_attrs, found := resourcemap["attributes"]
					if found {
						attrs, ok := _attrs.([]interface{})
						if ok {
							for _, attr := range attrs {
								attrmap, ok := attr.(map[string]interface{})
								if ok {
									if key, found := attrmap["key"]; found {
										keystr, ok := key.(string)
										if ok && keystr == "service.name" {
											if value, found := attrmap["value"]; found {
												valuemap, ok := value.(map[string]interface{})
												if ok {
													if value, found := valuemap["stringValue"]; found {
														valuestr, ok := value.(string)
														if ok {
															servicename = valuestr
															break
														}
													}
												}
											}
										}
									}
								}
							}
							if len(servicename) > 0 {
								resourcemap["serviceName"] = servicename
								cansink[0] = true
							}
						}
					}
				}
				if _, found := spanmap["scopeSpans"]; found {
					scopes, ok := spanmap["scopeSpans"].([]interface{})
					if ok {
						for _, scope := range scopes {
							scopemap, ok := scope.(map[string]interface{})
							if ok {
								if _, found := scopemap["spans"]; found {
									scopespans, ok := scopemap["spans"].([]interface{})
									if ok {
										for _, sspan := range scopespans {
											sspanmap, ok := sspan.(map[string]interface{})
											if ok {
												if start, found := sspanmap["startTimeUnixNano"]; found {
													startstr, ok := start.(string)
													if ok {
														startint, err := strconv.ParseInt(startstr, 10, 64)
														if err == nil {
															sspanmap["startTimeUnixNano"] = startint
															cansink[1] = true
														}
													}
												}
												if end, found := sspanmap["endTimeUnixNano"]; found {
													endstr, ok := end.(string)
													if ok {
														endint, err := strconv.ParseInt(endstr, 10, 64)
														if err == nil {
															sspanmap["endTimeUnixNano"] = endint
															cansink[2] = true
														}
													}
												}
											}
										}
									}
								}
							}
						}
					}
				}
			}
		}
	}
	if !(cansink[0] && cansink[1] && cansink[2]) {
		return nil, errors.New("invalid otel trace spans format")
	}
	return items, nil
}
