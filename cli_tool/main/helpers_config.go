package main

import (
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/go-yaml/yaml"
)

type ConfigOptions struct {
	Aws       AwsOptions
	Snowflake []SnowflakeOptions
	Venafi   []VenafiOptions
}
type AwsOptions struct {
	AccessKeyID string
	AccessKey   string
	Zone        string
	Bucket      string
}
type SnowflakeOptions struct {
	Role      string
	Account   string
	Username  string
	Password  string
	Warehouse string
	Database  string
	Schema    string
}
type VenafiOptions struct {
	AccessToken        string
	AccessTokenExpires string
	RefreshToken       string
	Url                string
}

func GetConfig() ConfigOptions {
	filepath := getConfigFilePath()
	if !verifyPathExist(filepath) {
		log.Fatal("App is not configured.")
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
func SetConfig(config ConfigOptions) {

	c, err := yaml.Marshal(config)
	if err != nil {
		log.Fatal("Failed to convert config to YAML: ", err)
	}

	p := getConfigFilePath()
	dirpath := filepath.Dir(p)
	if !verifyPathExist(dirpath) {
		os.Mkdir(dirpath, 0755)
	}

	f, err := os.Create(p)
	if err != nil {
		log.Fatal("Failed to create config file: ", err)
	}
	if _, err := f.Write(c); err != nil {
		log.Fatal("Failed to write data to config file: ", err)
	}

}

func getConfigFilePath() string {
	dirname, err := os.UserHomeDir()
	if err != nil {
		log.Fatal("Failed to find user home directory: ", err)
	}
	return filepath.Join(dirname, ".vsi", "config.yaml")

}
func verifyPathExist(p string) bool {
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return false
	}
	return true
}
