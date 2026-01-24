package config

type KafkaConfig struct {
	Topic string
	Host  string
	Port  string
}

var KafkaConf = KafkaConfig{
	Topic: "sync_topic",
	Host:  "kafka",
	Port:  "9092",
}
