package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

const VENAFI_SNOWFLAKE_INTEGRATION_NAME = "venafi_integration"

func Install(config ConfigOptions, s3Client *s3.Client, lambdaClient *lambda.Client, iamClient *iam.Client, gatewayClient *apigateway.Client, accountId string) {
	Log(true, "Getting service status", 0)
	status := GetStatus(1, config, s3Client, lambdaClient, iamClient, gatewayClient, accountId)

	if status.AwsConnection.State != 1 || status.AwsCredentials.State != 1 {
		log.Println("Failed to connection to AWS. Please check the configuration.\n\nStatus:\r")
		PrintStatus(status)
		os.Exit(1)
	}
	// first, check if bucket exists
	Log(true, "1. Deploying AWS Bucket: "+config.Aws.Bucket, 1)
	if status.AwsBucketFound.State != 1 {
		Log(true, "Double checking bucket exists", 1)
		_, _, bucketNotFound, _ := GetBucket(s3Client, config.Aws.Bucket)
		if bucketNotFound {
			Log(true, "Creating bucket...", 1)
			createError := CreateBucket(s3Client, config.Aws.Bucket)
			if createError != nil {
				log.Fatalf("Failed to create bucket: " + createError.Error())
			}
		} else {
			Log(true, "Bucket already exists", 1)
		}
		Log(true, "Constructing credentials file", 1)
		credFileReader := createCredentialFileReader(config)
		Log(true, "Uploading credentials file", 1)
		uploadError := UploadFile(context.TODO(), s3Client, config.Aws.Bucket, S3_CRED_FILE_NAME, &credFileReader)
		if uploadError != nil {
			log.Fatalf("Failed to upload credentials file: " + uploadError.Error())
		}
		Log(true, "Credentials file uploaded", 1)
		fmt.Print("Completed\n")
	} else {
		Log(true, "Bucket already exists", 1)
	}
	Log(true, "2. Create Lambda Execution Roles..", 1)

	if status.AwsLambdaS3Role.State != 1 {
		createRoleError := CreateLambdaS3Role(iamClient, AWS_ROLE_NAME)
		if createRoleError != nil {
			log.Fatalf("Failed to create roles in AWS: " + createRoleError.Error())
		}
		_, err := updateDeploymentInfo(s3Client, config.Aws.Bucket, func(info *DeploymentInfo) {
			info.Aws_role_name = AWS_ROLE_NAME
		})
		if err != nil {
			log.Fatalf("Failed to save deployment info: %v", err.Error())
		}
	} else {
		Log(true, "S3 Lambda role already exists\n", 1)
	}

	Log(true, "Finished installing roles", 1)

	var endpointUrl string
	var err error
	var restApiID string
	var parentResourceID string
	externalRole := AWS_ROLE_NAME
	info, _ := getDeploymentInfo(s3Client, config.Aws.Bucket)
	if info.Aws_gateway_endpoint_url != "" {
		endpointUrl = info.Aws_gateway_endpoint_url
	}
	if info.Aws_gateway_id != "" {
		restApiID = info.Aws_gateway_id
	}
	if info.Aws_gateway_parent_resource_id != "" {
		parentResourceID = info.Aws_gateway_parent_resource_id
	}
	if info.Aws_role_name != "" {
		externalRole = info.Aws_role_name
	}

	if status.AwsGateway.State != 1 {
		parentResourceID, restApiID, endpointUrl, err = CreateRestAPI(gatewayClient, externalRole, accountId, config)
		if err != nil {
			log.Fatalf("Failed to create API Integration: " + err.Error())
		}
		_, err = updateDeploymentInfo(s3Client, config.Aws.Bucket, func(info *DeploymentInfo) {
			info.Aws_gateway_endpoint_url = endpointUrl
			info.Aws_gateway_parent_resource_id = parentResourceID
			info.Aws_gateway_id = restApiID
		})
	}
	if err != nil {
		log.Fatalf("Failed to save deployment info: %v", err.Error())
	}
	if status.AwsLambdas.State != 1 || status.AwsGateway.State != 1 { // we have to redo aws lambdas if api gateway is missing
		Log(true, "3. Deploying AWS Lambdas and API Gateway ... ", 1)

		zipContent := createAwsLambdaZip()
		manageAwsLambda(LAMBDA_FUNCTION_NAME_GETMACHINEID, status.AwsLambas_Details.GetMachineId, lambdaClient, zipContent, restApiID, config.Aws.Zone, accountId, config.Aws.Bucket)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_REQUESTMACHINEID, status.AwsLambas_Details.RequestMachineId, lambdaClient, zipContent, restApiID, config.Aws.Zone, accountId, config.Aws.Bucket)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_LISTMACHINEIDS, status.AwsLambas_Details.ListMachineIds, lambdaClient, zipContent, restApiID, config.Aws.Zone, accountId, config.Aws.Bucket)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_RENEWMACHINEID, status.AwsLambas_Details.RenewMachineId, lambdaClient, zipContent, restApiID, config.Aws.Zone, accountId, config.Aws.Bucket)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_REVOKEMACHINEID, status.AwsLambas_Details.RevokeMachineId, lambdaClient, zipContent, restApiID, config.Aws.Zone, accountId, config.Aws.Bucket)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_GETMACHINEIDSTATUS, status.AwsLambas_Details.GetMachineIdStatus, lambdaClient, zipContent, restApiID, config.Aws.Zone, accountId, config.Aws.Bucket)

		err = IntegrateLambdaWithRestApi(gatewayClient, restApiID, parentResourceID, LAMBDA_FUNCTION_NAME_GETMACHINEID, accountId, config.Aws.Zone)
		if err != nil {
			log.Fatalf("Failed to integrate Lambda: %s : " + LAMBDA_FUNCTION_NAME_GETMACHINEID + "Error: " + err.Error())
		}

		err = IntegrateLambdaWithRestApi(gatewayClient, restApiID, parentResourceID, LAMBDA_FUNCTION_NAME_REQUESTMACHINEID, accountId, config.Aws.Zone)
		if err != nil {
			log.Fatalf("Failed to integrate Lambda: " + LAMBDA_FUNCTION_NAME_REQUESTMACHINEID + "Error: " + err.Error())
		}

		err = IntegrateLambdaWithRestApi(gatewayClient, restApiID, parentResourceID, LAMBDA_FUNCTION_NAME_LISTMACHINEIDS, accountId, config.Aws.Zone)
		if err != nil {
			log.Fatalf("Failed to integrate Lambda: " + LAMBDA_FUNCTION_NAME_LISTMACHINEIDS + "Error: " + err.Error())
		}

		err = IntegrateLambdaWithRestApi(gatewayClient, restApiID, parentResourceID, LAMBDA_FUNCTION_NAME_RENEWMACHINEID, accountId, config.Aws.Zone)
		if err != nil {
			log.Fatalf("Failed to integrate Lambda: " + LAMBDA_FUNCTION_NAME_RENEWMACHINEID + "Error: " + err.Error())
		}

		err = IntegrateLambdaWithRestApi(gatewayClient, restApiID, parentResourceID, LAMBDA_FUNCTION_NAME_REVOKEMACHINEID, accountId, config.Aws.Zone)
		if err != nil {
			log.Fatalf("Failed to integrate Lambda: " + LAMBDA_FUNCTION_NAME_RENEWMACHINEID + "Error: " + err.Error())
		}

		err = IntegrateLambdaWithRestApi(gatewayClient, restApiID, parentResourceID, LAMBDA_FUNCTION_NAME_GETMACHINEIDSTATUS, accountId, config.Aws.Zone)
		if err != nil {
			log.Fatalf("Failed to integrate Lambda: " + LAMBDA_FUNCTION_NAME_GETMACHINEIDSTATUS + "Error: " + err.Error())
			// }
			Log(true, "Created all lambda functions\n", 1)
		} else {
			Log(true, "All Lambdas exists already\n", 1)
		}

		err = DeployRestAPI(gatewayClient, restApiID)
		if err != nil {
			log.Fatalf("Failed to deploy Rest API")
		}

	} else {
		Log(true, "3. AWS Lambdas are ready ", 1)
	}
	if status.SnowflakeHealth.State != 1 || status.AwsGateway.State != 1 { // we have to redo snowflake functions if api gateway is created with a new id
		Log(true, "Deploying Snowflake API Integration and External Functions ... ", 0)

		for _, snowflake := range config.Snowflake {
			Log(true, "1. Deploying Snowflake API Integration and External Functions ... ", 1)
			externalID, policyARN, err := CreateSnowflakeApiIntegration(VENAFI_SNOWFLAKE_INTEGRATION_NAME, fmt.Sprintf("arn:aws:iam::%s:role/%s", accountId, externalRole), endpointUrl, snowflake)
			if err != nil {
				log.Fatalf("Failed to create Snowflake API Integration: " + err.Error())
			}

			_, err = updateDeploymentInfo(s3Client, config.Aws.Bucket, func(info *DeploymentInfo) {
				info.Sf_eternal_id = externalID
				info.Sf_user_arn = policyARN
			})

			Log(true, "2. Attach AWS policies to Snowflake Integration ... ", 1)
			err = AttachSnowflakePropertiesToPolicy(iamClient, externalRole, externalID, policyARN)
			if err != nil {
				log.Fatal("Failed to add permission for Snowflake to use AWS Lambda + ", err.Error())
			}

			Log(true, "2. Create Snowflake External Functions ... ", 1)

			CreateSnowflakeFunction(SNOWFLAKE_FUNCTION_NAME_GETMACHINEID, SNOWFLAKE_FUNCTION_ALIAS_GETMACHINEID, endpointUrl, snowflake, VENAFI_SNOWFLAKE_INTEGRATION_NAME)
			CreateSnowflakeFunction(SNOWFLAKE_FUNCTION_NAME_REQUESTMACHINEID, SNOWFLAKE_FUNCTION_ALIAS_REQUESTMACHINEID, endpointUrl, snowflake, VENAFI_SNOWFLAKE_INTEGRATION_NAME)
			CreateSnowflakeFunction(SNOWFLAKE_FUNCTION_NAME_LISTMACHINEIDS, SNOWFLAKE_FUNCTION_ALIAS_LISTMACHINEIDS, endpointUrl, snowflake, VENAFI_SNOWFLAKE_INTEGRATION_NAME)
			CreateSnowflakeFunction(SNOWFLAKE_FUNCTION_NAME_RENEWMACHINEID, SNOWFLAKE_FUNCTION_ALIAS_RENEWMACHINEID, endpointUrl, snowflake, VENAFI_SNOWFLAKE_INTEGRATION_NAME)
			CreateSnowflakeFunction(SNOWFLAKE_FUNCTION_NAME_RENEWMACHINEID, SNOWFLAKE_FUNCTION_ALIAS_RENEWMACHINEID, endpointUrl, snowflake, VENAFI_SNOWFLAKE_INTEGRATION_NAME)
			CreateSnowflakeFunction(SNOWFLAKE_FUNCTION_NAME_REVOKEMACHINEID, SNOWFLAKE_FUNCTION_ALIAS_REVOKEMACHINEID, endpointUrl, snowflake, VENAFI_SNOWFLAKE_INTEGRATION_NAME)
			CreateSnowflakeFunction(SNOWFLAKE_FUNCTION_NAME_GETMACHINEIDSTATUS, SNOWFLAKE_FUNCTION_ALIAS_GETMACHINEIDSTATUS, endpointUrl, snowflake, VENAFI_SNOWFLAKE_INTEGRATION_NAME)
		}

		Log(true, "Created all Snowflake External Functions\n", 1)
	} else {
		Log(true, "Api Integration and Snowflake Functions are ready\n", 1)
	}

}

func createCredentialFileReader(config ConfigOptions) io.Reader {
	requestByte, _ := json.Marshal(config.Venafi)

	requestReader := bytes.NewReader(requestByte)
	return requestReader
}

// TODO: Change this to work from packaged executable, remove dependency on working dir
func createAwsLambdaZip() []byte {
	ex, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	toZipDir := filepath.Join(filepath.Dir(ex), "main", "bin", "handlers")
	files, err := ioutil.ReadDir(toZipDir)
	if err != nil {
		log.Fatal("Failed to read Handlers directory", err)
	}
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)
	for _, f := range files {

		zipFile, err := w.Create(f.Name())
		if err != nil {
			log.Fatalf("Failed to create resources ZIP entry: " + f.Name())
		}
		readFile, err := ioutil.ReadFile(filepath.Join(toZipDir, f.Name()))
		if err != nil {
			log.Fatalf("Failed to read resources ZIP entry: " + f.Name())
		}
		_, err = zipFile.Write(readFile)
		if err != nil {
			log.Fatalf("Failed to write resources ZIP entry: " + f.Name())
		}
	}
	e := w.Close()
	if e != nil {
		log.Fatalf("Failed to save resources ZIP")
	}
	return buf.Bytes()

}

func manageAwsLambda(functionName string, status StatusResult, lambdaClient *lambda.Client, zipContent []byte, restApiID string, zone string, accountId string, bucket string) {
	if status.State < 2 {
		return
	}
	name := GetLambdaFunctionName(functionName)
	// if its an error, then remove it first
	if status.State == 2 {
		err := DeleteLambdaFunction(lambdaClient, name)
		if err != nil {
			log.Fatalf("Failed to delete function '%v'. Please delete it manually.", name)
		}
	}

	if status.State == 3 || status.State == 2 {
		err := CreateLambdaFunction(lambdaClient, name, strings.Replace(functionName, "-", "", 0), zipContent, restApiID, zone, accountId, bucket)
		if err != nil {
			log.Fatalf("Failed to create function '%v': " + err.Error())
		}

	}

}
