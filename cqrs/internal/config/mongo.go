// Package config provides configuration settings for the application.
package config

import "os"

// MongoConnection holds the MongoDB connection parameters.
type MongoConnection struct {
	// User is the MongoDB username, read from MONGO_INITDB_ROOT_USERNAME.
	User string
	// Password is the MongoDB user password, read from MONGO_INITDB_ROOT_PASSWORD.
	Password string
	// URL is the MongoDB server hostname.
	URL string
	// Port is the MongoDB server port.
	Port string
}

// MongoConnect is the global MongoDB connection configuration.
var MongoConnect = MongoConnection{
	User:     os.Getenv("MONGO_INITDB_ROOT_USERNAME"),
	Password: os.Getenv("MONGO_INITDB_ROOT_PASSWORD"),
	URL:      "email_mongo",
	Port:     "27017",
}
