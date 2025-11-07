package taskbuilder

// Root structure that matches the fc-stash task JSON format
type ConfigRoot struct {
	Data ConfigData `json:"data"`
}

// ConfigData contains all data sources, tasks, and label mappings
type ConfigData struct {
	DataSources   map[string]*DataSource `json:"data_sources"`
	Tasks         []*Task                `json:"tasks"`
	LabelMappings map[string]interface{} `json:"label_mappings"`
}

// DataSource can be Kafka, VictoriaLogs, Elasticsearch, or Doris
type DataSource struct {
	Type     string                 `json:"type"`
	Settings map[string]interface{} `json:"settings"`
}

// Task represents a single log processing task
type Task struct {
	ID           int                    `json:"id"`
	DataSourceID int                    `json:"data_source_id"`
	Settings     *TaskSettings          `json:"settings"`
	Storage      *StorageConfig         `json:"storage"`
}

// TaskSettings defines how to extract and transform logs
type TaskSettings struct {
	Mode         string       `json:"mode"`
	Version      string       `json:"version"`
	JSONSettings *JSONSettings `json:"json_settings,omitempty"`
}

// JSONSettings for JSON mode extraction and transformation
type JSONSettings struct {
	PreFunction        []interface{}       `json:"pre_function"`
	PrefixMatch        []string            `json:"prefix_match"`
	PreExtract         []interface{}       `json:"pre_extract"`
	MultiPreExtract    bool                `json:"multi_pre_extract"`
	SkipExtractError   bool                `json:"skip_extract_error"`
	OverwriteOrigin    bool                `json:"overwrite_origin"`
	FieldsTransform    []*FieldTransform   `json:"fields_transform"`
}

// FieldTransform defines a single field transformation rule
type FieldTransform struct {
	RuleType          string                 `json:"rule_type"`
	OriginField       string                 `json:"origin_field"`
	OriginType        string                 `json:"origin_type"`
	TargetField       string                 `json:"target_field,omitempty"`
	TargetType        string                 `json:"target_type"`
	FormatSettings    *FormatSettings        `json:"format_settings"`
	AppendSettings    *AppendSettings        `json:"append_settings"`
	DropIfNotFound    bool                   `json:"drop_if_not_found"`
	IgnoreIfConflict  bool                   `json:"ignore_if_conflict"`
}

// FormatSettings for field formatting
type FormatSettings struct {
	FormatType     string                 `json:"format_type,omitempty"`
	TimeFormat     *TimeFormat            `json:"time_format,omitempty"`
	Desensitize    map[string]interface{} `json:"desensitize"`
	RegexpMapping  map[string]interface{} `json:"regexp_mapping"`
}

// TimeFormat for timestamp formatting
type TimeFormat struct {
	DateFormat   string `json:"date_format,omitempty"`
	DateLocation string `json:"date_location,omitempty"`
}

// AppendSettings for appending labels
type AppendSettings struct {
	LabelMapping map[string]interface{} `json:"label_mapping"`
}

// StorageConfig defines where to store the processed logs
type StorageConfig struct {
	DataSourceType string              `json:"data_source_type"`
	VictoriaLogs   *VMLogsStorage      `json:"victoria_logs,omitempty"`
	Elasticsearch  *ElasticsearchStorage `json:"elasticsearch,omitempty"`
	Doris          *DorisStorage       `json:"doris,omitempty"`
}

// VMLogsStorage configuration for VictoriaLogs
type VMLogsStorage struct {
	DataSourceID         int                   `json:"data_source_id"`
	MsgFieldSettings     *MsgFieldSettings     `json:"msg_field_settings"`
	TimeFieldSettings    *TimeFieldSettings    `json:"time_field_settings"`
	StreamFieldsSettings *StreamFieldsSettings `json:"stream_fields_settings"`
}

// MsgFieldSettings for message field configuration
type MsgFieldSettings struct {
	AutoMsg      bool   `json:"auto_msg"`
	MsgFieldName string `json:"msg_field_name,omitempty"`
}

// TimeFieldSettings for timestamp field configuration
type TimeFieldSettings struct {
	AutoTime      bool   `json:"auto_time"`
	TimeFieldName string `json:"time_field_name,omitempty"`
}

// StreamFieldsSettings for stream fields configuration
type StreamFieldsSettings struct {
	Unspecified      bool     `json:"unspecified"`
	StreamFieldNames []string `json:"stream_field_names,omitempty"`
}

// ElasticsearchStorage configuration (placeholder for future implementation)
type ElasticsearchStorage struct {
	DataSourceID int                    `json:"data_source_id"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
}

// DorisStorage configuration (placeholder for future implementation)
type DorisStorage struct {
	DataSourceID int                    `json:"data_source_id"`
	Settings     map[string]interface{} `json:"settings,omitempty"`
}
