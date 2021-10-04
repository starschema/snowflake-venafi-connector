package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const DEPLOYMENTINFO_FILE_NAME = "deploymentinfo.json"

type DeploymentInfo struct {
	Aws_function_arns              []string
	Aws_role_name                  string
	Aws_gateway_id                 string
	Aws_gateway_parent_resource_id string
	Aws_gateway_endpoint_url       string
	Sf_eternal_id                  string
	Sf_user_arn                    string
}

func getDeploymentInfo(client *s3.Client, bucketName string) (*DeploymentInfo, error) {
	fileContents, err := ReadFile(client, bucketName, DEPLOYMENTINFO_FILE_NAME)
	if err != nil {
		if strings.Contains(err.Error(), "StatusCode: 404") {
			return &DeploymentInfo{}, nil
		}
		return nil, fmt.Errorf("Cannot get DeploymentInfo. Failed to read file: %v", err.Error())
	}
	var deploymentInfo DeploymentInfo
	err = json.Unmarshal(fileContents, &deploymentInfo)
	if err != nil {
		return nil, fmt.Errorf("Cannot get DeploymentInfo. Failed to deserialize file: %v", err.Error())
	}
	return &deploymentInfo, nil
}

func updateDeploymentInfo(client *s3.Client, bucketName string, fn func(info *DeploymentInfo)) (*DeploymentInfo, error) {
	info, err := getDeploymentInfo(client, bucketName)
	if err != nil {
		return nil, fmt.Errorf("Cannot update DeploymentInfo. Failed to get: %v", err.Error())
	}
	fn(info)
	i := *info
	requestByte, err := json.Marshal(i)
	if err != nil {
		return nil, fmt.Errorf("Cannot update DeploymentInfo. Failed to serialize file: %v", err.Error())
	}
	ioReader := ioReader(requestByte)
	err = UploadFile(context.TODO(), client, bucketName, DEPLOYMENTINFO_FILE_NAME, &ioReader)
	if err != nil {
		return nil, fmt.Errorf("Cannot update DeploymentInfo. Failed to upload file: %v", err.Error())
	}
	return info, nil
}
func ioReader(b []byte) io.Reader {
	return bytes.NewReader(b)
}

func ReadFile(svc *s3.Client, bucket string, key string) ([]byte, error) {
	deploymentInfo, err := svc.GetObject(context.TODO(), &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return []byte{}, err
	}
	s3objectBytes, err := ioutil.ReadAll(deploymentInfo.Body)
	if err != nil {
		return []byte{}, err
	}
	return s3objectBytes, nil
}
