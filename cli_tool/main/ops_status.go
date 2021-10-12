package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type ServiceStatus struct {
	AwsConnection              StatusResult
	AwsCredentials             StatusResult
	AwsBucketFound             StatusResult
	AwsDeploymentInfoRead      StatusResult
	AwsGateway                 StatusResult
	AwsLambdas                 StatusResult
	AwsLambas_Details          AwsLambdaStatuses
	AwsLambdaS3Role            StatusResult
	AwsSnowflakeRole           StatusResult
	AwsPolicyToS3              StatusResult
	SnowflakeHealth            StatusResult
	SnowflakeFunctions_Details []SnowflakeFunctionStatuses
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
	SnowflakeAccount    string
	SnowflakeDb         string
	SnowflakeWarehouse  string
	SnowflakeSchema     string
	SnowflakeUser       string
	SnowflakeRole       string
	SnowflakeConnection StatusResult
	GetMachineId        StatusResult
	RequestMachineId    StatusResult
	ListMachineIds      StatusResult
	RenewMachineId      StatusResult
	RevokeMachineId     StatusResult
	GetMachineIdStatus  StatusResult
}

type FunctionCheckState struct {
	AnyError   bool
	AnyMissing bool
}

func GetStatus(tabIndex int, c ConfigOptions, s3Client *s3.Client, lambdaClient *lambda.Client, iamClient *iam.Client, gatewayClient *apigateway.Client, accountId string) ServiceStatus {
	ret := ServiceStatus{}
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
				suc, err = IsFileUploaded(context.TODO(), s3Client, c.Aws.Bucket, S3_CRED_FILE_NAME)
				ret.AwsDeploymentInfoRead.State = 1
				if !suc {
					if err == nil {
						ret.AwsDeploymentInfoRead.State = 3
						ret.AwsDeploymentInfoRead.Error = errors.New("Deployment Info file not found")
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

	deploymentInfo, err := getDeploymentInfo(s3Client, c.Aws.Bucket)
	if err != nil {
		ret.AwsDeploymentInfoRead.State = 2
		ret.AwsDeploymentInfoRead.Error = fmt.Errorf("Cannot read deployment file: %v", err)
		Log(true, "Failed to read Deployment info. Message: "+ret.AwsDeploymentInfoRead.Error.Error(), 0)
		return ret
	} else {
		ret.AwsDeploymentInfoRead.State = 1
	}

	// Check Roles
	roleName := deploymentInfo.Aws_lambda_role_name
	if roleName == "" {
		roleName = AWS_LAMBDA_ROLE_NAME
	}

	ret.AwsLambdaS3Role = GetAwsRole(iamClient, roleName)

	snowflakeRole := deploymentInfo.Aws_snowflake_role_name
	if snowflakeRole == "" {
		snowflakeRole = AWS_SNOWFLAKE_ROLE_NAME
	}

	ret.AwsSnowflakeRole = GetAwsRole(iamClient, snowflakeRole)

	policyArn := deploymentInfo.Aws_policy_arn
	if policyArn == "" {
		ret.AwsPolicyToS3 = StatusResult{State: 2}
	} else {
		ret.AwsPolicyToS3 = GetAwsPolicy(iamClient, snowflakeRole)
	}

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

	// Check RestAPI exists
	if deploymentInfo.Aws_gateway_id != "" {
		err := GetRestApi(gatewayClient, deploymentInfo.Aws_gateway_id)
		if err != nil {
			ret.AwsGateway.State = 2
			ret.AwsGateway.Error = errors.New("Could not get Rest API or Rest API is missing")
		} else {
			ret.AwsGateway.State = 1
		}

	} else {
		ret.AwsGateway.State = 3
		ret.AwsGateway.Error = errors.New("No RestApi ID in deployment info")
	}
	snowflake_state := FunctionCheckState{}
	for _, snowflake := range c.Snowflake {
		sfd := SnowflakeFunctionStatuses{
			SnowflakeAccount:   snowflake.Account,
			SnowflakeDb:        snowflake.Database,
			SnowflakeRole:      snowflake.Role,
			SnowflakeSchema:    snowflake.Schema,
			SnowflakeUser:      snowflake.Username,
			SnowflakeWarehouse: snowflake.Warehouse,
		}
		err = CheckSnowflakeConnection(snowflake)
		if err != nil {
			sfd.SnowflakeConnection.State = 2
			sfd.SnowflakeConnection.Error = err
		} else {
			ret.SnowflakeHealth.State = 1
		}

		// Check Snowflake External Functions
		sfd.GetMachineId = getSnowflakeFunctionStatus(snowflake, SNOWFLAKE_FUNCTION_NAME_GETMACHINEID, &snowflake_state)
		sfd.RequestMachineId = getSnowflakeFunctionStatus(snowflake, SNOWFLAKE_FUNCTION_NAME_REQUESTMACHINEID, &snowflake_state)
		sfd.ListMachineIds = getSnowflakeFunctionStatus(snowflake, SNOWFLAKE_FUNCTION_NAME_LISTMACHINEIDS, &snowflake_state)
		sfd.RenewMachineId = getSnowflakeFunctionStatus(snowflake, SNOWFLAKE_FUNCTION_NAME_RENEWMACHINEID, &snowflake_state)
		sfd.RevokeMachineId = getSnowflakeFunctionStatus(snowflake, SNOWFLAKE_FUNCTION_NAME_REVOKEMACHINEID, &snowflake_state)
		sfd.GetMachineIdStatus = getSnowflakeFunctionStatus(snowflake, SNOWFLAKE_FUNCTION_NAME_GETMACHINEIDSTATUS, &snowflake_state)

		ret.SnowflakeFunctions_Details = append(ret.SnowflakeFunctions_Details, sfd)
	}

	if snowflake_state.AnyError {
		ret.SnowflakeHealth.State = 3
		ret.SnowflakeHealth.Error = fmt.Errorf("At least one snowlake function has an error")
	} else if snowflake_state.AnyMissing {
		ret.SnowflakeHealth.State = 3
		ret.SnowflakeHealth.Error = fmt.Errorf("At least one snowlake function is missing")
	} else {
		ret.SnowflakeHealth.State = 1
	}
	return ret

}

func PrintStatus(status ServiceStatus) {

	printStatusResult("AWS Connection", status.AwsConnection, "Success", "Error", "")
	printStatusResult("AWS Credentials", status.AwsCredentials, "Valid", "Invalid", "")
	printStatusResult("AWS Bucket", status.AwsBucketFound, "Exists", "Missing", "Exists, credential file not found")
	fmt.Printf("")
	printStatusResult("AWS Lambda <-> S3 role", status.AwsLambdaS3Role, "Exists", "Failed to check", "Missing")
	printStatusResult("Snowflake <-> Lambda", status.AwsSnowflakeRole, "Exists", "Failed to check", "Missing")

	// printStatusResult("AWS Lambda <-> SF role", status.AwsLambdaSnowflakeRole, "Exists", "Failed to check", "Missing")
	fmt.Printf("")
	printStatusResult("AWS API Gateway", status.AwsGateway, "Aws API Gateway exists", "AWS API Gateway not found", "AWS APi Gateway not found")
	printStatusResult("AWS Lambdas", status.AwsLambdas, "All lambdas are online", "One or more lambdas are in error state or missing", "One or more lambdas are missing")
	printAwsLambdaResult("GetMachineId", status.AwsLambas_Details.GetMachineId, 1)
	printAwsLambdaResult("RequestMachineId", status.AwsLambas_Details.RequestMachineId, 1)
	printAwsLambdaResult("ListMachineIds", status.AwsLambas_Details.ListMachineIds, 1)
	printAwsLambdaResult("RenewMachineId", status.AwsLambas_Details.RenewMachineId, 1)
	printAwsLambdaResult("RevokeMachineId", status.AwsLambas_Details.RevokeMachineId, 1)
	printAwsLambdaResult("GetMachineIdStatus", status.AwsLambas_Details.GetMachineIdStatus, 1)
	fmt.Printf("")
	printStatusResult("Snowflake health", status.SnowflakeHealth, "Success", "Error", "")
	fmt.Printf("")
	fmt.Printf("Detailed snowflake results: %v", len(status.SnowflakeFunctions_Details))
	for _, status := range status.SnowflakeFunctions_Details {
		fmt.Printf("\n\n\tSnowflake Account: '%v', Warehouse: '%v', Schema: '%v':\n", status.SnowflakeAccount, status.SnowflakeWarehouse, status.SnowflakeSchema)
		printAwsLambdaResult("GetMachineId", status.GetMachineId, 2)
		printAwsLambdaResult("RequestMachineId", status.RequestMachineId, 2)
		printAwsLambdaResult("ListMachineIds", status.ListMachineIds, 2)
		printAwsLambdaResult("RenewMachineId", status.RenewMachineId, 2)
		printAwsLambdaResult("RevokeMachineId", status.RevokeMachineId, 2)
		printAwsLambdaResult("GetMachineIdStatus", status.GetMachineIdStatus, 2)
	}

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
func printAwsLambdaResult(name string, status StatusResult, indent int) {
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
	pretext := ""
	for i := 0; i < indent; i++ {
		pretext += "\t"
	}
	fmt.Printf("%v%v: %v\n", pretext, name, s)
}
