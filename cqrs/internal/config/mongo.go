// Package config provides configuration settings for the application.
package config

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
	// DBName is the MongoDB database name for read-model projections.
	DBName string
}

// MongoConnect is the global MongoDB connection configuration.
var MongoConnect = MongoConnection{
	User:     getEnv("MONGO_INITDB_ROOT_USERNAME", "mongo_user"),
	Password: getEnv("MONGO_INITDB_ROOT_PASSWORD", "mongo_password"),
	URL:      getEnv("MONGO_HOST", "localhost"),
	Port:     getEnv("MONGO_PORT", "27017"),
	DBName:   getEnv("MONGO_DB_NAME", "email_service"),
}
