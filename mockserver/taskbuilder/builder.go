package taskbuilder

import (
	"encoding/json"
	"fmt"
	"os"
)

// ConfigBuilder helps build complete task configurations
type ConfigBuilder struct {
	dataSources   map[string]*DataSource
	tasks         []*Task
	labelMappings map[string]interface{}
}

// NewConfigBuilder creates a new config builder
func NewConfigBuilder() *ConfigBuilder {
	return &ConfigBuilder{
		dataSources:   make(map[string]*DataSource),
		tasks:         make([]*Task, 0),
		labelMappings: make(map[string]interface{}),
	}
}

// AddDataSource adds a custom data source
func (cb *ConfigBuilder) AddDataSource(id string, ds *DataSource) *ConfigBuilder {
	cb.dataSources[id] = ds
	return cb
}

// AddKafkaSource adds a Kafka data source with common settings
func (cb *ConfigBuilder) AddKafkaSource(id string, brokers []string, topic, group string) *ConfigBuilder {
	ds := &DataSource{
		Type: "kafka",
		Settings: map[string]interface{}{
			"kafka.brokers": brokers,
			"kafka.topic":   topic,
			"kafka.group":   group,
			"kafka.auth": map[string]interface{}{
				"method": "",
				"mtls": map[string]interface{}{
					"client.cert": "",
					"client.key":  "",
					"ca.cert":     "",
					"server.name": "",
				},
				"sasl": map[string]interface{}{
					"client.username": "",
					"client.password": "",
				},
			},
			"kafka.message.sample": "",
		},
	}
	cb.dataSources[id] = ds
	return cb
}

// AddVMLogsSource adds a VictoriaLogs data source
func (cb *ConfigBuilder) AddVMLogsSource(id string, endpoints []string, user, password string) *ConfigBuilder {
	ds := &DataSource{
		Type: "victoria_logs",
		Settings: map[string]interface{}{
			"vmlogs.endpoints": endpoints,
			"vmlogs.timeout":   10,
			"vmlogs.basic": map[string]interface{}{
				"vmlogs.auth.enable": true,
				"vmlogs.user":        user,
				"vmlogs.password":    password,
			},
			"vmlogs.tls": map[string]interface{}{
				"vmlogs.tls.skip_tls_verify": true,
			},
			"vmlogs.cluster_name": "",
			"header":              map[string]interface{}{},
		},
	}
	cb.dataSources[id] = ds
	return cb
}

// AddTask adds a task to the configuration
func (cb *ConfigBuilder) AddTask(task *Task) *ConfigBuilder {
	cb.tasks = append(cb.tasks, task)
	return cb
}

// Build creates the final ConfigRoot structure
func (cb *ConfigBuilder) Build() *ConfigRoot {
	return &ConfigRoot{
		Data: ConfigData{
			DataSources:   cb.dataSources,
			Tasks:         cb.tasks,
			LabelMappings: cb.labelMappings,
		},
	}
}

// ToJSON converts the configuration to JSON string
func (cb *ConfigBuilder) ToJSON() (string, error) {
	root := cb.Build()
	data, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SaveToFile saves the configuration to a JSON file
func (cb *ConfigBuilder) SaveToFile(filename string) error {
	jsonStr, err := cb.ToJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	err = os.WriteFile(filename, []byte(jsonStr), 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// TaskBuilder helps build individual tasks
type TaskBuilder struct {
	task *Task
}

// NewTaskBuilder creates a new task builder
func NewTaskBuilder() *TaskBuilder {
	return &TaskBuilder{
		task: &Task{
			Settings: &TaskSettings{
				Mode:    "json",
				Version: "log_transform",
				JSONSettings: &JSONSettings{
					PreFunction:      []interface{}{},
					PrefixMatch:      []string{},
					PreExtract:       []interface{}{},
					MultiPreExtract:  false,
					SkipExtractError: false,
					OverwriteOrigin:  false,
					FieldsTransform:  []*FieldTransform{},
				},
			},
			Storage: &StorageConfig{},
		},
	}
}

// WithID sets the task ID
func (tb *TaskBuilder) WithID(id int) *TaskBuilder {
	tb.task.ID = id
	return tb
}

// WithDataSourceID sets the input data source ID
func (tb *TaskBuilder) WithDataSourceID(id int) *TaskBuilder {
	tb.task.DataSourceID = id
	return tb
}

// WithPrefixMatch sets the prefix match fields for extraction
func (tb *TaskBuilder) WithPrefixMatch(fields ...string) *TaskBuilder {
	tb.task.Settings.JSONSettings.PrefixMatch = fields
	return tb
}

// AddFieldTransform adds a field transformation rule
func (tb *TaskBuilder) AddFieldTransform(transform *FieldTransform) *TaskBuilder {
	tb.task.Settings.JSONSettings.FieldsTransform = append(
		tb.task.Settings.JSONSettings.FieldsTransform,
		transform,
	)
	return tb
}

// AddTimestampFormat adds a timestamp format transformation
func (tb *TaskBuilder) AddTimestampFormat(originField string, dateFormat, location string) *TaskBuilder {
	transform := &FieldTransform{
		RuleType:    "format",
		OriginField: originField,
		OriginType:  "float",
		TargetType:  "date",
		FormatSettings: &FormatSettings{
			FormatType: "allmatch",
			TimeFormat: &TimeFormat{
				DateFormat:   dateFormat,
				DateLocation: location,
			},
			Desensitize:   map[string]interface{}{},
			RegexpMapping: map[string]interface{}{},
		},
		AppendSettings: &AppendSettings{
			LabelMapping: map[string]interface{}{},
		},
		DropIfNotFound:   false,
		IgnoreIfConflict: false,
	}
	tb.AddFieldTransform(transform)
	return tb
}

// AddFieldRename adds a field rename transformation
func (tb *TaskBuilder) AddFieldRename(originField, targetField, originType, targetType string) *TaskBuilder {
	transform := &FieldTransform{
		RuleType:    "rename",
		OriginField: originField,
		OriginType:  originType,
		TargetField: targetField,
		TargetType:  targetType,
		FormatSettings: &FormatSettings{
			TimeFormat:    &TimeFormat{},
			Desensitize:   map[string]interface{}{},
			RegexpMapping: map[string]interface{}{},
		},
		AppendSettings: &AppendSettings{
			LabelMapping: map[string]interface{}{},
		},
		DropIfNotFound:   false,
		IgnoreIfConflict: false,
	}
	tb.AddFieldTransform(transform)
	return tb
}

// WithVMLogsStorage configures VictoriaLogs storage
func (tb *TaskBuilder) WithVMLogsStorage(dataSourceID int, msgField, timeField string, streamFields []string) *TaskBuilder {
	tb.task.Storage.DataSourceType = "victoria_logs"
	tb.task.Storage.VictoriaLogs = &VMLogsStorage{
		DataSourceID: dataSourceID,
		MsgFieldSettings: &MsgFieldSettings{
			AutoMsg:      len(msgField) == 0,
			MsgFieldName: msgField,
		},
		TimeFieldSettings: &TimeFieldSettings{
			AutoTime:      len(timeField) == 0,
			TimeFieldName: timeField,
		},
		StreamFieldsSettings: &StreamFieldsSettings{
			Unspecified:      len(streamFields) == 0,
			StreamFieldNames: streamFields,
		},
	}
	return tb
}

// Build creates the final Task
func (tb *TaskBuilder) Build() *Task {
	return tb.task
}
