package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/lambda"
)

type ServiceStatus struct {
	AwsConnection              StatusResult
	AwsCredentials             StatusResult
	AwsBucketFound             StatusResult
	AwsLambdas                 StatusResult
	AwsLambas_Details          AwsLambdaStatuses
	AwsLambdaS3Role            StatusResult
	AwsLambdaSnowflakeRole     StatusResult
	SnowflakeConnection        StatusResult
	SnowflakeFunctions         StatusResult
	SnowflakeFunctions_Details SnowflakeFunctionStatuses
	AwsToVenafiConnectivity    StatusResult
	SfToAwsConnectivity        StatusResult
}
type StatusResult struct {
	// 0 - NotChecked
	// 1 - Success
	// 2 - Error
	// 3 - PartialError
	State int
	Error error
}
type AwsLambdaStatuses struct {
	GetMachineId       StatusResult
	RequestMachineId   StatusResult
	ListMachineIds     StatusResult
	RenewMachineId     StatusResult
	RevokeMachineId    StatusResult
	GetMachineIdStatus StatusResult
}

type SnowflakeFunctionStatuses struct {
	GetMachineId       StatusResult
	RequestMachineId   StatusResult
	ListMachineIds     StatusResult
	RenewMachineId     StatusResult
	RevokeMachineId    StatusResult
	GetMachineIdStatus StatusResult
}

type FunctionCheckState struct {
	AnyError   bool
	AnyMissing bool
}

func GetStatus(tabIndex int) ServiceStatus {
	ret := ServiceStatus{}
	c, _, s3Client, lambdaClient, iamClient, _ := bootstrapOperation(tabIndex)
	Log(true, "Connecting to AWS & getting Bucket '%v' ... ", 0, c.Aws.Bucket)
	_, credsInvalid, bucketNotFound, err := GetBucket(s3Client, c.Aws.Bucket)
	if err != nil {
		ret.AwsConnection.Error = err
		ret.AwsConnection.State = 2
	} else {
		ret.AwsConnection.State = 1
		if credsInvalid {
			ret.AwsCredentials.Error = err
			ret.AwsCredentials.State = 2
		} else {
			ret.AwsCredentials.State = 1
			if bucketNotFound {
				ret.AwsBucketFound.Error = err
				ret.AwsBucketFound.State = 2
			} else {
				ret.AwsBucketFound.State = 1
				suc, err := IsFileUploaded(context.TODO(), s3Client, c.Aws.Bucket, S3_CRED_FILE_NAME)
				if !suc {
					if err == nil {
						ret.AwsBucketFound.State = 3
						ret.AwsBucketFound.Error = errors.New("Credential file not found")
					} else {
						ret.AwsBucketFound.State = 3
						ret.AwsBucketFound.Error = fmt.Errorf("Failed to get credential file information: %v", err)
					}
				}
			}

		}
	}

	if ret.AwsConnection.State != 1 || ret.AwsCredentials.State != 1 {
		Log(true, "Basic AWS connection failed, returning getStatus()", 0)
		// checkSnowflake(&ret)
		return ret
	}

	// Check Roles
	roleName := "Venafi-test-access"

	ret.AwsLambdaS3Role = GetLambdaRole(iamClient, roleName)
	ret.AwsLambdaSnowflakeRole = GetLambdaRole(iamClient, "snowflake-execute")

	// Check AWS lambas
	lambda_state := FunctionCheckState{}
	ret.AwsLambas_Details.GetMachineId = getLambdaFunctionStatus(lambdaClient, LAMBDA_FUNCTION_NAME_GETMACHINEID, &lambda_state)
	ret.AwsLambas_Details.RequestMachineId = getLambdaFunctionStatus(lambdaClient, LAMBDA_FUNCTION_NAME_REQUESTMACHINEID, &lambda_state)
	ret.AwsLambas_Details.ListMachineIds = getLambdaFunctionStatus(lambdaClient, LAMBDA_FUNCTION_NAME_LISTMACHINEIDS, &lambda_state)
	ret.AwsLambas_Details.RenewMachineId = getLambdaFunctionStatus(lambdaClient, LAMBDA_FUNCTION_NAME_RENEWMACHINEID, &lambda_state)
	ret.AwsLambas_Details.RevokeMachineId = getLambdaFunctionStatus(lambdaClient, LAMBDA_FUNCTION_NAME_REVOKEMACHINEID, &lambda_state)
	ret.AwsLambas_Details.GetMachineIdStatus = getLambdaFunctionStatus(lambdaClient, LAMBDA_FUNCTION_NAME_GETMACHINEIDSTATUS, &lambda_state)

	if lambda_state.AnyError {
		ret.AwsLambdas.State = 2
	} else if lambda_state.AnyMissing {
		ret.AwsLambdas.State = 3
	} else {
		ret.AwsLambdas.State = 1
	}

	err = CheckSnowflakeConnection()
	if err != nil {
		ret.SnowflakeConnection.State = 2
		ret.SnowflakeConnection.Error = err
	} else {
		ret.SnowflakeConnection.State = 1
	}

	// Check Snowflake External Functions
	snowflake_state := FunctionCheckState{}
	ret.SnowflakeFunctions_Details.GetMachineId = getSnowflakeFunctionStatus(c.Snowflake[0], LAMBDA_FUNCTION_NAME_GETMACHINEID, &snowflake_state)
	ret.SnowflakeFunctions_Details.RequestMachineId = getSnowflakeFunctionStatus(c.Snowflake[0], LAMBDA_FUNCTION_NAME_REQUESTMACHINEID, &snowflake_state)
	ret.SnowflakeFunctions_Details.ListMachineIds = getSnowflakeFunctionStatus(c.Snowflake[0], LAMBDA_FUNCTION_NAME_LISTMACHINEIDS, &snowflake_state)
	ret.SnowflakeFunctions_Details.RenewMachineId = getSnowflakeFunctionStatus(c.Snowflake[0], LAMBDA_FUNCTION_NAME_RENEWMACHINEID, &snowflake_state)
	ret.SnowflakeFunctions_Details.RevokeMachineId = getSnowflakeFunctionStatus(c.Snowflake[0], LAMBDA_FUNCTION_NAME_REVOKEMACHINEID, &snowflake_state)
	ret.SnowflakeFunctions_Details.GetMachineIdStatus = getSnowflakeFunctionStatus(c.Snowflake[0], LAMBDA_FUNCTION_NAME_GETMACHINEIDSTATUS, &snowflake_state)

	if snowflake_state.AnyError {
		ret.SnowflakeFunctions.State = 2
	} else if lambda_state.AnyMissing {
		ret.SnowflakeFunctions.State = 3
	} else {
		ret.SnowflakeFunctions.State = 1
	}

	return ret

}

func PrintStatus(status ServiceStatus) {
	printStatusResult("AWS Connection", status.AwsConnection, "Success", "Error", "")
	printStatusResult("AWS Credentials", status.AwsCredentials, "Valid", "Invalid", "")
	printStatusResult("AWS Bucket", status.AwsBucketFound, "Exists", "Missing", "Exists, credential file not found")

	printStatusResult("AWS Lambda <-> S3 role", status.AwsLambdaS3Role, "Exists", "Failed to check", "Missing")
	printStatusResult("AWS Lambda <-> SF role", status.AwsLambdaSnowflakeRole, "Exists", "Failed to check", "Missing")

	printStatusResult("AWS Lambdas", status.AwsLambdas, "All lambdas are online", "One or more lambdas are in error state or missing", "One or more lambdas are missing")
	printAwsLambdaResult("GetMachineId", status.AwsLambas_Details.GetMachineId)
	printAwsLambdaResult("RequestMachineId", status.AwsLambas_Details.RequestMachineId)
	printAwsLambdaResult("ListMachineIds", status.AwsLambas_Details.ListMachineIds)
	printAwsLambdaResult("RenewMachineId", status.AwsLambas_Details.RenewMachineId)
	printAwsLambdaResult("RevokeMachineId", status.AwsLambas_Details.RevokeMachineId)
	printAwsLambdaResult("GetMachineIdStatus", status.AwsLambas_Details.GetMachineIdStatus)

	printStatusResult("Snowflake connection", status.SnowflakeConnection, "Success", "Error", "")

	printStatusResult("Snowflake functions", status.SnowflakeFunctions, "All Snowflake External Functions are online", "One or more lambdas are in error state or missing", "One or more lambdas are in error state")
	printAwsLambdaResult("GetMachineId", status.SnowflakeFunctions_Details.GetMachineId)
	printAwsLambdaResult("RequestMachineId", status.SnowflakeFunctions_Details.RequestMachineId)
	printAwsLambdaResult("ListMachineIds", status.SnowflakeFunctions_Details.ListMachineIds)
	printAwsLambdaResult("RenewMachineId", status.SnowflakeFunctions_Details.RenewMachineId)
	printAwsLambdaResult("RevokeMachineId", status.SnowflakeFunctions_Details.RevokeMachineId)
	printAwsLambdaResult("GetMachineIdStatus", status.SnowflakeFunctions_Details.GetMachineIdStatus)

}
func printStatusResult(name string, status StatusResult, valueSuccess string, valueError string, valuePartialError string) {
	s := ""
	switch status.State {
	case 0:
		s = "NOT CHECKED"
		break
	case 1:
		s = valueSuccess
		break
	case 2:
		s = valueError
		break
	case 3:
		s = valuePartialError
		break
	}
	fmt.Printf("%v: %v\n", name, s)
	if status.Error != nil {
		fmt.Printf("\n\tError:\n\t%v\n", status.Error.Error())
	}
}
func getLambdaFunctionStatus(svc *lambda.Client, name string, state *FunctionCheckState) StatusResult {
	ret := GetLambdaFunction(svc, GetLambdaFunctionName(name))
	switch ret.State {
	case 3:
		state.AnyMissing = true
		break
	case 2:
		state.AnyError = true
		break
	}
	return ret
}
func printAwsLambdaResult(name string, status StatusResult) {
	s := ""
	switch status.State {
	case 0:
		s = "NOT CHECKED"
		break
	case 1:
		s = "ONLINE"
		break
	case 2:
		s = "ERROR"
		break
	case 3:
		s = "MISSING"
		break
	}
	fmt.Printf("\t%v: %v\n", name, s)
}
