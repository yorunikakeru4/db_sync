// Package config provides configuration settings for the application.
package config

import "os"

// KafkaConfig holds the Kafka broker and topic settings.
type KafkaConfig struct {
	// Topic is the Kafka topic consumed by the sync service.
	Topic string
	// Host is the Kafka broker hostname.
	Host string
	// Port is the Kafka broker port as a string.
	Port string
	// GroupID is the Kafka consumer group identifier.
	GroupID string
}

// KafkaConf is the global Kafka configuration used by the consumer.
var KafkaConf = KafkaConfig{
	Topic:   getEnv("KAFKA_TOPIC", "sync_topic"),
	Host:    getEnv("KAFKA_HOST", "kafka"),
	Port:    getEnv("KAFKA_PORT", "9092"),
	GroupID: getEnv("KAFKA_GROUP_ID", "db_sync"),
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
