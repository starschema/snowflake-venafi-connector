package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
)

func RunServerlessDeployment() {
	cmd := exec.Command("/bin/bash", "-c", "serverless deploy")
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "serverless deploy")
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}

}

func getCommandStringLinux(accessKeyId, accessKey string) string {
	return "serverless config credentials --provider aws --profile venafi-snowflake-connector -o" + fmt.Sprintf(" --key %s ", accessKeyId) + fmt.Sprintf(" --s %s ", accessKey)
}

func RunConfigServerlessAWSCredentials(accessKeyId, accessKey string) {
	cmdStr := getCommandStringLinux(accessKeyId, accessKey)
	cmd := exec.Command("/bin/bash", "-c", cmdStr)
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "serverless config credentials --provider aws --profile venafi-snowflake-connector -o"+fmt.Sprintf(" --key %s ", accessKeyId)+fmt.Sprintf(" --s %s ", accessKey))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
