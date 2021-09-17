package main

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func bootstrapOperation(tabIndex int) (ConfigOptions, aws.Config, *s3.Client, *lambda.Client, *iam.Client, *apigateway.Client) {
	Log(true, "Getting app config", tabIndex)
	c := GetConfig()
	Log(true, "Getting AWS config", tabIndex)
	awsConfig := GetAwsConfig(c.Aws)
	Log(true, "Create S3 client", tabIndex)
	s3Client := s3.NewFromConfig(awsConfig)
	Log(true, "Create Lambda client", tabIndex)
	lambdaClient := lambda.NewFromConfig(awsConfig)
	Log(true, "Create iAM client", tabIndex)
	iamClient := iam.NewFromConfig(awsConfig)
	Log(true, "Create API Gateway client", tabIndex)
	gatewayClient := apigateway.NewFromConfig(awsConfig)

	return c, awsConfig, s3Client, lambdaClient, iamClient, gatewayClient
}
