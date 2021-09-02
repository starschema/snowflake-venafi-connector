package cli_tool

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type AWSConfig struct {
	Zone        string
	S3Bucket    string
	AccessKeyID string
	AccessKey   string
}

type SnowflakeConfig struct {
	Role      string
	Account   string
	Username  string
	Password  string
	Warehouse string
	Database  string
	Schema    string
}

type Config struct {
	AWS       AWSConfig
	Snowflake SnowflakeConfig
}

func CreateConfigFile(conf Config) {

	data, err := yaml.Marshal(&conf)

	if err != nil {
		log.Fatal(err)
	}

	err2 := ioutil.WriteFile("./config.yml", data, 0)

	if err2 != nil {

		log.Fatal(err2)
	}

	err = os.Chmod("./config.yml", 0700)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("data written")
}
