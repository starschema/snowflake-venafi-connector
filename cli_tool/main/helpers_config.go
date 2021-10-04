package main

import (
	"io/ioutil"
	"log"
	"os"

	"github.com/go-yaml/yaml"
)

type ConfigOptions struct {
	Aws       AwsOptions         `yaml:"aws"`
	Snowflake []SnowflakeOptions `yaml:"snowflake"`
	Venafi    []VenafiOptions    `yaml:"venafi"`
}
type AwsOptions struct {
	AccessKeyID string
	AccessKey   string
	Zone        string
	Bucket      string
}
type SnowflakeOptions struct {
	Role      string `yaml:"role"`
	Account   string `yaml:"account"`
	Username  string `yaml:"username"`
	Password  string `yaml:"password"`
	Warehouse string `yaml:"warehouse"`
	Database  string `yaml:"database"`
	Schema    string `yaml:"schema"`
}
type VenafiOptions struct {
	AccessToken        string
	AccessTokenExpires string
	RefreshToken       string
	Url                string
}

func GetConfig(configFilePath string) ConfigOptions {
	filepath := configFilePath
	if !verifyPathExist(filepath) {
		log.Fatal("Configuration file is missing. Please use an existing file for the -file flag")
	}
	fileBytes, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatal("Failed to read config file: ", err)
	}

	config := ConfigOptions{}

	if err := yaml.Unmarshal(fileBytes, &config); err != nil {
		log.Fatal("Failed to parse config file: ", err)
	}

	return config
}

func verifyPathExist(p string) bool {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return false
	}
	return true
}
