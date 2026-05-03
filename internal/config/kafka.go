// Package config provides configuration settings for the application.
package config

// KafkaConfig holds the Kafka broker and topic settings.
type KafkaConfig struct {
	// Topic is the Kafka topic consumed by the sync service.
	Topic string
	// Host is the Kafka broker hostname.
	Host string
	// Port is the Kafka broker port as a string.
	Port string
}

// KafkaConf is the global Kafka configuration used by the consumer.
var KafkaConf = KafkaConfig{
	Topic: "sync_topic",
	Host:  "kafka",
	Port:  "9092",
}
