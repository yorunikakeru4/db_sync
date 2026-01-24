package config

import "os"

type SQLConnection struct {
	Host     string
	Port     string
	Password string
	User     string
	Name     string
}

var SQLConnect = SQLConnection{
	Host:     "email_postgres",
	Port:     "5432",
	User:     os.Getenv("POSTGRES_USER"),
	Password: os.Getenv("POSTGRES_PASSWORD"),
	Name:     os.Getenv("POSTGRES_DB"),
}
