package config

import "os"

// SQLConnection holds the PostgreSQL connection parameters.
type SQLConnection struct {
	// Host is the PostgreSQL server hostname.
	Host string
	// Port is the PostgreSQL server port.
	Port string
	// Password is the database user password, read from POSTGRES_PASSWORD.
	Password string
	// User is the database username, read from POSTGRES_USER.
	User string
	// Name is the database name, read from POSTGRES_DB.
	Name string
}

// SQLConnect is the global PostgreSQL connection configuration.
var SQLConnect = SQLConnection{
	Host:     "email_postgres",
	Port:     "5432",
	User:     os.Getenv("POSTGRES_USER"),
	Password: os.Getenv("POSTGRES_PASSWORD"),
	Name:     os.Getenv("POSTGRES_DB"),
}
