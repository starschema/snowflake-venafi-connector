package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awshttp "github.com/aws/aws-sdk-go-v2/aws/transport/http"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/apigateway"
	gatewayTypes "github.com/aws/aws-sdk-go-v2/service/apigateway/types"
	"github.com/aws/aws-sdk-go-v2/service/iam"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	lambdaTypes "github.com/aws/aws-sdk-go-v2/service/lambda/types"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

const S3_CRED_FILE_NAME = "credentials.json"
const LAMBDA_FUNCTION_NAME_PREFIX = "venafi-snowlake-func-"
const LAMBDA_FUNCTION_NAME_GETMACHINEID = "get-machine-id"
const LAMBDA_FUNCTION_NAME_REQUESTMACHINEID = "request-machine-id"
const LAMBDA_FUNCTION_NAME_LISTMACHINEIDS = "list-machine-ids"
const LAMBDA_FUNCTION_NAME_RENEWMACHINEID = "renew-machine-ids"
const LAMBDA_FUNCTION_NAME_REVOKEMACHINEID = "revoke-machine-ids"
const LAMBDA_FUNCTION_NAME_GETMACHINEIDSTATUS = "get-machine-id-status"

func GetAwsConfig(awsConfig AwsOptions) aws.Config {

	os.Setenv("AWS_ACCESS_KEY_ID", awsConfig.AccessKeyID)
	os.Setenv("AWS_SECRET_ACCESS_KEY", awsConfig.AccessKey)
	os.Setenv("AWS_DEFAULT_REGION", awsConfig.Zone)
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		LogFatal("%v", err)
	}
	return cfg
}

func GetBucket(svc *s3.Client, bucketName string) (bucket []types.Object, credsInvalid bool, bucketNotFound bool, err error) {
	input := &s3.ListObjectsInput{
		Bucket:  aws.String(bucketName + ""),
		MaxKeys: *aws.Int32(2),
	}

	result, err := svc.ListObjects(context.TODO(), input)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchBucket") {
			return []types.Object{}, false, true, nil
		}
		if strings.Contains(err.Error(), "SignatureDoesNotMatch:") {
			return []types.Object{}, true, true, nil
		}
		return []types.Object{}, false, false, err
	}
	return result.Contents, false, false, nil
}

func CreateBucket(svc *s3.Client, bucketName string) error {
	input := &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		ACL:    types.BucketCannedACLPrivate,
		CreateBucketConfiguration: &types.CreateBucketConfiguration{
			LocationConstraint: types.BucketLocationConstraintEuWest1,
		},
	}
	_, err := svc.CreateBucket(context.TODO(), input)
	return err
}

func IsFileUploaded(ctx context.Context, client *s3.Client, bucket string, key string) (bool, error) {
	_, err := client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var responseError *awshttp.ResponseError
		if errors.As(err, &responseError) && responseError.ResponseError.HTTPStatusCode() == http.StatusNotFound {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func UploadFile(ctx context.Context, client *s3.Client, bucket string, key string, reader *io.Reader) error {
	uploader := manager.NewUploader(client)
	_, err := uploader.Upload(context.TODO(), &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Body:   *reader,
	})
	return err
}
func GetLambdaFunctionName(name string) string {
	return LAMBDA_FUNCTION_NAME_PREFIX + name
}

func GetLambdaFunction(svc *lambda.Client, functionName string) StatusResult {
	ret := StatusResult{}

	_, err := svc.GetFunction(context.TODO(), &lambda.GetFunctionInput{
		FunctionName: aws.String(functionName),
	})

	if err != nil {
		if strings.Contains(err.Error(), "StatusCode: 404") {
			ret.State = 3
		} else {
			ret.State = 2
			ret.Error = err
		}
	} else {
		ret.State = 1
	}
	return ret
}
func DeleteLambdaFunction(svc *lambda.Client, functionName string) error {
	_, err := svc.DeleteFunction(context.TODO(), &lambda.DeleteFunctionInput{
		FunctionName: aws.String(functionName),
	})
	return err
}
func CreateLambdaFunction(svc *lambda.Client, functionName string, binaryName string, zipContent []byte) error {
	_, err := svc.CreateFunction(context.TODO(), &lambda.CreateFunctionInput{
		FunctionName: aws.String(functionName),
		Role:         aws.String("arn:aws:iam::300480681691:role/Venafi-full-access-to-s3-and-lambda"),
		Code: &lambdaTypes.FunctionCode{
			ZipFile: zipContent,
		},
		Runtime: lambdaTypes.RuntimeGo1x,
		Handler: aws.String(binaryName),
	})
	return err
}

func GetLambdaRole(svc *iam.Client, roleName string) StatusResult {
	_, err := svc.GetRole(context.TODO(), &iam.GetRoleInput{RoleName: aws.String(roleName)})
	ret := StatusResult{}
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchEntity") {
			ret.State = 3
		} else {
			ret.State = 2
			ret.Error = err
		}
	} else {
		ret.State = 1
	}
	return ret
}

func CreateLambdaS3Role(svc *iam.Client, roleName string) error {
	_, err := svc.CreateRole(context.TODO(), &iam.CreateRoleInput{
		RoleName:    aws.String(roleName),
		Description: aws.String("Execution Role for AWS Lambdas to access S3"),
		AssumeRolePolicyDocument: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [{
				"Sid": "AmazonS3FullAccess",
				"Effect": "Allow",
				"Principal": {
					"Service": "s3.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "CloudWatchLogsFullAccess",
				"Effect": "Allow",
				"Principal": {
					"Service": "cloudwatch.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			},
			{
				"Sid": "AWSLambdaFullAccess",
				"Effect": "Allow",
				"Principal": {
					"Service": "lambda.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}]
		}`),
	})
	if err != nil {
		return err
	}
	_, err = svc.AttachRolePolicy(context.TODO(), &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AmazonS3FullAccess"), // access to s3
	})
	if err != nil {
		return err
	}
	_, err = svc.AttachRolePolicy(context.TODO(), &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/AWSLambda_FullAccess"), // access to lambdas
	})
	if err != nil {
		return err
	}

	_, err = svc.AttachRolePolicy(context.TODO(), &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: aws.String("arn:aws:iam::aws:policy/CloudWatchLogsFullAccess"), // access to cloudwatch logs
	})
	if err != nil {
		return err
	}

	return nil // TODO MORE MEANINGFUL ERRORS
}

// func CreateAWSExecutionRole(svc *iam.Client) error {
// 	_, err := svc.CreateRole(context.TODO(), &iam.CreateRoleInput{
// 		RoleName:    aws.String("snowflake-execute"),
// 		Description: aws.String("Execution Role for AWS Lambdas to access S3"),
// 		AssumeRolePolicyDocument: aws.String(`{
// 			"Version": "2012-10-17",
// 			"Statement": [
// 				{
// 					"Effect": "Allow",
// 					"Principal": {
// 						"AWS": "arn:aws:sts::300480681691:assumed-role/snowflake-execute/snowflake"
// 					},
// 					"Action": "execute-api:Invoke"
// 				}
// 			]
// 		}`),
// 	})
// 	if err != nil {
// 		return err
// 	}

// 	return nil // TODO MORE MEANINGFUL ERRORS
// }

func CreateRestAPI(svc *apigateway.Client) (string, error) {
	values, err := svc.CreateRestApi(context.TODO(), &apigateway.CreateRestApiInput{
		Name:                  aws.String("venafi-snowflake-func-test2"),
		Description:           aws.String("Api Gateway for AWS Lambda Venafi Functions"),
		EndpointConfiguration: &gatewayTypes.EndpointConfiguration{Types: []gatewayTypes.EndpointType{gatewayTypes.EndpointTypeRegional}},
		Policy: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Principal": {
						"AWS": "arn:aws:sts::300480681691:assumed-role/venafi-snowflake-connector-dev-eu-west-1-lambdaRole/snowflake"
					},
					"Action": "execute-api:Invoke",
					"Resource": "arn:aws:execute-api:eu-west-1:300480681691:*/dev/POST/*"
				}
			]
		}`),
	})
	if err != nil {
		return "", err
	}
	fmt.Printf("!!!VALIES: %v", *values.Id)
	parentResource, err := svc.GetResources(context.TODO(), &apigateway.GetResourcesInput{
		RestApiId: aws.String(*values.Id),
	})
	if err != nil {
		return "", err
	}
	resource, err := svc.CreateResource(context.TODO(), &apigateway.CreateResourceInput{
		RestApiId: aws.String(*values.Id),
		ParentId:  aws.String(*parentResource.Items[0].Id),
		PathPart:  aws.String("getmachineid"),
	})
	if err != nil {
		return "", err
	}
	_, err = svc.PutMethod(context.TODO(), &apigateway.PutMethodInput{
		AuthorizationType: aws.String("AWS_IAM"),
		HttpMethod:        aws.String("POST"),
		ResourceId:        aws.String(*resource.Id),
		RestApiId:         aws.String(*values.Id),
	})
	if err != nil {
		return "", err
	}
	_, err = svc.PutIntegration(context.TODO(), &apigateway.PutIntegrationInput{
		RestApiId:             aws.String(*values.Id),
		ResourceId:            resource.Id,
		IntegrationHttpMethod: aws.String("POST"),
		HttpMethod:            aws.String("POST"),
		// RequestParameters:     map[string]string{"type": "string", "tpp_url": "string", "request_id": "string"},
		Type: gatewayTypes.IntegrationTypeAwsProxy,
		Uri:  aws.String("arn:aws:apigateway:us-west-1:lambda:path/2015-03-31/functions/arn:aws:lambda:eu-west-1:300480681691:function:venafi-snowlake-func-get-machine-id/invocations"),
	})
	if err != nil {
		return "", err
	}

	_, err = svc.CreateDeployment(context.TODO(), &apigateway.CreateDeploymentInput{
		RestApiId: aws.String(*values.Id),
		StageName: aws.String("dev"),
	})
	if err != nil {
		return "", err
	}
	endpointUrl := fmt.Sprintf("https://%s.execute-api.eu-west-1.amazonaws.com/dev/", *values.Id)
	fmt.Printf("RestApi: %v", fmt.Sprintf("https://%s.execute-api.eu-west-1.amazonaws.com/dev/", *values.Id))
	return endpointUrl, nil
}
