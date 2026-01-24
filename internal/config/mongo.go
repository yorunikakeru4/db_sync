// Package config provides configuration settings for the application.
package config

import "os"

type MongoConnection struct {
	User     string
	Password string
	URL      string
	Port     string
}

var MongoConnect = MongoConnection{
	User:     os.Getenv("MONGO_INITDB_ROOT_USERNAME"),
	Password: os.Getenv("MONGO_INITDB_ROOT_PASSWORD"),
	URL:      "email_mongo",
	Port:     "27017",
}
