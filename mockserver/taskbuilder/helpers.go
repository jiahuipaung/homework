package taskbuilder

// VMLogsTaskConfig holds configuration for creating a VMLogs task
type VMLogsTaskConfig struct {
	TaskID           int
	KafkaSourceID    int
	VMLogsSourceID   int
	PrefixMatchFields []string
	MessageField     string
	TimeField        string
	StreamFields     []string
	TimestampField   string
	TimestampFormat  string
	TimestampLocation string
}

// NewDefaultVMLogsTask creates a VMLogs task with common default settings
func NewDefaultVMLogsTask(taskID, kafkaSourceID, vmlogsSourceID int) *Task {
	config := VMLogsTaskConfig{
		TaskID:           taskID,
		KafkaSourceID:    kafkaSourceID,
		VMLogsSourceID:   vmlogsSourceID,
		PrefixMatchFields: []string{"message", "status", "timestamp", "agent_hostname", "fcservice", "fcsource", "level"},
		MessageField:     "message",
		TimeField:        "@timestamp",
		StreamFields:     []string{"fcservice", "level", "agent_hostname"},
		TimestampField:   "timestamp",
		TimestampFormat:  "%s%3N",
		TimestampLocation: "Local",
	}
	return NewVMLogsTask(config)
}

// NewVMLogsTask creates a VMLogs task from configuration
func NewVMLogsTask(config VMLogsTaskConfig) *Task {
	builder := NewTaskBuilder().
		WithID(config.TaskID).
		WithDataSourceID(config.KafkaSourceID).
		WithPrefixMatch(config.PrefixMatchFields...)

	// Add timestamp formatting if specified
	if config.TimestampField != "" && config.TimestampFormat != "" {
		builder.AddTimestampFormat(config.TimestampField, config.TimestampFormat, config.TimestampLocation)

		// Add rename if time field is different
		if config.TimeField != "" && config.TimeField != config.TimestampField {
			builder.AddFieldRename(config.TimestampField, config.TimeField, "float", "text")
		}
	}

	// Configure VMLogs storage
	builder.WithVMLogsStorage(config.VMLogsSourceID, config.MessageField, config.TimeField, config.StreamFields)

	return builder.Build()
}

// AddVMLogsTaskSimple is a convenience method on ConfigBuilder to add a VMLogs task with minimal config
func (cb *ConfigBuilder) AddVMLogsTaskSimple(taskID, kafkaSourceID, vmlogsSourceID int) *ConfigBuilder {
	task := NewDefaultVMLogsTask(taskID, kafkaSourceID, vmlogsSourceID)
	return cb.AddTask(task)
}

// AddVMLogsTaskCustom is a convenience method to add a VMLogs task with custom configuration
func (cb *ConfigBuilder) AddVMLogsTaskCustom(config VMLogsTaskConfig) *ConfigBuilder {
	task := NewVMLogsTask(config)
	return cb.AddTask(task)
}

// QuickVMLogsConfig creates a complete config with Kafka + VMLogs sources and one task
func QuickVMLogsConfig(
	kafkaBrokers []string,
	kafkaTopic string,
	kafkaGroup string,
	vmlogsEndpoints []string,
	vmlogsUser string,
	vmlogsPassword string,
) *ConfigBuilder {
	return NewConfigBuilder().
		AddKafkaSource("20001", kafkaBrokers, kafkaTopic, kafkaGroup).
		AddVMLogsSource("20002", vmlogsEndpoints, vmlogsUser, vmlogsPassword).
		AddVMLogsTaskSimple(1001, 20001, 20002)
}
