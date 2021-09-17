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

	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

func Install() {
	config, _, s3Client, lambdaClient, iamClient, gatewayClient := bootstrapOperation(0)
	Log(true, "Getting service status", 0)
	status := GetStatus(1)

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
		createRoleError := CreateLambdaS3Role(iamClient, "venafi-test-access")
		if createRoleError != nil {
			log.Fatalf("Failed to create roles in AWS: " + createRoleError.Error())
		}
	} else {
		Log(true, "S3 Lambda role already exists\n", 1)
	}

	// if status.AwsLambdaSnowflakeRole.State != 1 {
	// 	createRoleError := CreateAWSExecutionRole(iamClient)
	// 	if createRoleError != nil {
	// 		log.Fatalf("Failed to create role in AWS: " + createRoleError.Error())
	// 	}
	// } else {
	// 	Log(true, "Lambda Snowflake Execution role already exists\n", 1)
	//}
	Log(true, "Finished installing roles", 1)

	Log(true, "3. Deploying AWS Lambdas ... ", 1)
	if status.AwsLambdas.State != 1 {
		zipContent := createAwsLambdaZip()
		manageAwsLambda(LAMBDA_FUNCTION_NAME_GETMACHINEID, status.AwsLambas_Details.GetMachineId, lambdaClient, zipContent)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_REQUESTMACHINEID, status.AwsLambas_Details.GetMachineId, lambdaClient, zipContent)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_LISTMACHINEIDS, status.AwsLambas_Details.GetMachineId, lambdaClient, zipContent)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_RENEWMACHINEID, status.AwsLambas_Details.GetMachineId, lambdaClient, zipContent)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_REVOKEMACHINEID, status.AwsLambas_Details.GetMachineId, lambdaClient, zipContent)
		manageAwsLambda(LAMBDA_FUNCTION_NAME_GETMACHINEIDSTATUS, status.AwsLambas_Details.GetMachineId, lambdaClient, zipContent)
		Log(true, "Created all lambda functions\n", 1)
	} else {
		Log(true, "All Lambdas exists already\n", 1)
	}

	Log(true, "4. Deploying Snowflake API Integration and External Functions ... ", 1)
	endpointUrl, err := CreateRestAPI(gatewayClient)
	if err != nil {
		log.Fatalf("Failed to create API Integration: " + err.Error())
	}

	_, _, err = CreateSnowflakeApiIntegration("SNOWFLAKE_TEST_INT", "arn:aws:iam::300480681691:role/venafi-snowflake-connector-dev-eu-west-1-lambdaRole", endpointUrl)
	if err != nil {
		log.Fatalf("Failed to create Snowflake API Integration: " + err.Error())
	}

	if status.SnowflakeFunctions.State != 1 { // TODO
		// CreateSnowflakeFunction(LAMBDA_FUNCTION_NAME_GETMACHINEID, "", "")
		// CreateSnowflakeFunction(LAMBDA_FUNCTION_NAME_REQUESTMACHINEID, "", "")
		// CreateSnowflakeFunction(LAMBDA_FUNCTION_NAME_LISTMACHINEIDS, "", "")
		// CreateSnowflakeFunction(LAMBDA_FUNCTION_NAME_RENEWMACHINEID, "", "")
		// CreateSnowflakeFunction(LAMBDA_FUNCTION_NAME_RENEWMACHINEID, "", "")
		// CreateSnowflakeFunction(LAMBDA_FUNCTION_NAME_REVOKEMACHINEID, "", "")
		// CreateSnowflakeFunction(LAMBDA_FUNCTION_NAME_GETMACHINEIDSTATUS, "", "")
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

func manageAwsLambda(functionName string, status StatusResult, lambdaClient *lambda.Client, zipContent []byte) {
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
		err := CreateLambdaFunction(lambdaClient, name, strings.Replace(functionName, "-", "", 0), zipContent)
		if err != nil {
			log.Fatalf("Failed to create function '%v': " + err.Error())
		}

	}

}
