package data_source

import (
	"context"
	"fmt"
	"testing"
)

func Test_KafkaMessage(t *testing.T) {
	plug := &PlugLoggingKafka{
		Brokers: []string{"10.99.1.105:9092"},
		Topic:   "fc_log_theme_test",
	}

	message, err := plug.GetMessageSample(context.TODO(), nil, true)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(message)
}
