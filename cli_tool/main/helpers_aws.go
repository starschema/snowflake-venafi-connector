package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
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

	"github.com/aws/aws-sdk-go-v2/service/sts"
)

const S3_CRED_FILE_NAME = "credentials.json"
const LAMBDA_FUNCTION_NAME_PREFIX = "venafi-snowflake-func-"
const LAMBDA_FUNCTION_NAME_GETMACHINEID = "getmachineid"
const LAMBDA_FUNCTION_NAME_REQUESTMACHINEID = "requestmachineid"
const LAMBDA_FUNCTION_NAME_LISTMACHINEIDS = "listmachineids"
const LAMBDA_FUNCTION_NAME_RENEWMACHINEID = "renewmachineid"
const LAMBDA_FUNCTION_NAME_REVOKEMACHINEID = "revokemachineid"
const LAMBDA_FUNCTION_NAME_GETMACHINEIDSTATUS = "getmachineidstatus"
const AWS_LAMBDA_ROLE_NAME = "lambda-execute-role"
const AWS_SNOWFLAKE_ROLE_NAME = "snowflake-role"
const AWS_POLICY_TO_ACCESS_BUCKET = "venafi-lambda-access-to-s3-bucket"

const AWS_REST_API_NAME = "venafi-snowflake-rest-api"

func GetAwsConfig(awsConfig AwsOptions) aws.Config {
	if awsConfig.Profile != "" {
		cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithSharedConfigProfile(awsConfig.Profile))
		if err != nil {
			LogFatal("Please check if you has valid credential and config file in your .aws folder%v", err)
		}
		return cfg
	} else {
		LogFatal("PLease define your AWS profile in the config file ")
	}
	return aws.Config{}
}

func GetCallerIdentity(svc *sts.Client) (string, error) {
	input := &sts.GetCallerIdentityInput{}
	result, err := svc.GetCallerIdentity(context.TODO(), input)
	if err != nil {
		return "", err
	}
	return *result.Account, nil
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
func CreateLambdaFunction(svc *lambda.Client, functionName string, binaryName string, zipContent []byte, restAPIID, zone, accountID, bucket, lambdaRole string) error {
	sourceARN := fmt.Sprintf("arn:aws:execute-api:%s:%s:%s/*/*", zone, accountID, restAPIID)
	envVariables := make(map[string]string)
	envVariables["ZONE"] = zone
	envVariables["CREDENTIAL_FILE_NAME"] = S3_CRED_FILE_NAME
	envVariables["S3_BUCKET"] = bucket

	err := Retry(func() error {
		_, er := svc.CreateFunction(context.TODO(), &lambda.CreateFunctionInput{
			FunctionName: aws.String(functionName),
			Role:         aws.String(fmt.Sprintf("arn:aws:iam::%s:role/%s", accountID, lambdaRole)),
			Code: &lambdaTypes.FunctionCode{
				ZipFile: zipContent,
			},
			Runtime:     lambdaTypes.RuntimeGo1x,
			Handler:     aws.String(binaryName),
			Environment: &lambdaTypes.Environment{Variables: envVariables},
			Timeout:     aws.Int32(30)})
		return er
	})
	if err != nil {
		return err
	}
	_, err = svc.AddPermission(context.TODO(), &lambda.AddPermissionInput{
		FunctionName: aws.String(functionName),
		Action:       aws.String("lambda:InvokeFunction"),
		Principal:    aws.String("apigateway.amazonaws.com"),
		SourceArn:    aws.String(sourceARN),
		StatementId:  aws.String(fmt.Sprintf("%s-policy", functionName)),
	})
	if err != nil {
		return err
	}
	return nil
}

func GetAwsRole(svc *iam.Client, roleName string) StatusResult {
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

func GetAwsPolicy(svc *iam.Client, policyARN string) StatusResult {
	_, err := svc.GetPolicy(context.TODO(), &iam.GetPolicyInput{PolicyArn: aws.String(policyARN)})
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

func CreateLambdaS3Role(svc *iam.Client, roleName string, bucket string) error {
	policyResp, err := svc.CreatePolicy(context.TODO(), &iam.CreatePolicyInput{
		PolicyName: aws.String(fmt.Sprintf("%s-%s", AWS_POLICY_TO_ACCESS_BUCKET, bucket)), //put bucket name to policy for easier remove / debug
		PolicyDocument: aws.String(fmt.Sprintf(`{
			"Version": "2012-10-17",
			"Statement": [
				{
					"Effect": "Allow",
					"Action": [
						"s3:ListBucket"
					],
					"Resource": "arn:aws:s3:::%s"
				},
				{
					"Effect": "Allow",
					"Action": [
						"s3:PutObject",
						"s3:GetObject"
					],
					"Resource": "arn:aws:s3:::%s/*"
				}
			]
		}`, bucket, bucket)),
	})
	if err != nil {
		return err
	}
	_, err = svc.CreateRole(context.TODO(), &iam.CreateRoleInput{
		RoleName:    aws.String(roleName),
		Description: aws.String("Execution Role for AWS Lambdas to access S3"),
		AssumeRolePolicyDocument: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [{
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
		PolicyArn: aws.String("arn:aws:iam::aws:policy/CloudWatchLogsFullAccess"), // access to cloudwatch logs
	})
	if err != nil {
		return err
	}

	_, err = svc.AttachRolePolicy(context.TODO(), &iam.AttachRolePolicyInput{
		RoleName:  aws.String(roleName),
		PolicyArn: policyResp.Policy.Arn, // attach access to s3 bucket
	})
	if err != nil {
		return err
	}

	return nil // TODO MORE MEANINGFUL ERRORS
}

func CreateSnowflakeRole(svc *iam.Client, roleName string) error {
	_, err := svc.CreateRole(context.TODO(), &iam.CreateRoleInput{
		RoleName:    aws.String(roleName),
		Description: aws.String("Role for Rest Api to call from Snowflake"),
		AssumeRolePolicyDocument: aws.String(`{
			"Version": "2012-10-17",
			"Statement": [{
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

	return nil // TODO MORE MEANINGFUL ERRORS
}

func AttachSnowflakePropertiesToPolicy(svc *iam.Client, roleName string, externalID string, awsUserARN string) error {
	policiyStr := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"AWS": "%s"
				},
				"Action": "sts:AssumeRole",
				"Condition": {
					"StringEquals": {
						"sts:ExternalId": "%s"
					}
				}
			},
			{
				"Effect": "Allow",
				"Principal": {
					"Service": "lambda.amazonaws.com"
				},
				"Action": "sts:AssumeRole"
			}
		]
	}`, awsUserARN, externalID)
	_, err := svc.UpdateAssumeRolePolicy(context.TODO(), &iam.UpdateAssumeRolePolicyInput{
		RoleName:       aws.String(roleName),
		PolicyDocument: aws.String(policiyStr),
	}) // access to cloudwatch logs
	if err != nil {
		return err
	}

	return nil // TODO MORE MEANINGFUL ERRORS
}

func CreateRestAPI(svc *apigateway.Client, role, accountId string, region string) (string, string, string, error) {
	principalStr := fmt.Sprintf(`{
		"Version": "2012-10-17",
		"Statement": [
			{
				"Effect": "Allow",
				"Principal": {
					"AWS": "arn:aws:sts::%s:assumed-role/%s/snowflake"
				},
				"Action": "execute-api:Invoke",
				"Resource": "arn:aws:execute-api:%s:%s:*/dev/POST/*"
			}
		]
	}`, accountId, role, region, accountId)
	var values *apigateway.CreateRestApiOutput
	var restApiError error
	err := Retry(func() error {
		values, restApiError = svc.CreateRestApi(context.TODO(), &apigateway.CreateRestApiInput{
			Name:                  aws.String(AWS_REST_API_NAME),
			Description:           aws.String("Api Gateway for AWS Lambda Venafi Functions"),
			EndpointConfiguration: &gatewayTypes.EndpointConfiguration{Types: []gatewayTypes.EndpointType{gatewayTypes.EndpointTypeRegional}},
			Policy:                aws.String(principalStr),
		})
		return restApiError
	})

	if err != nil {
		return "", "", "", err
	}
	parentResource, err := svc.GetResources(context.TODO(), &apigateway.GetResourcesInput{
		RestApiId: aws.String(*values.Id),
	})
	if err != nil {
		return "", "", "", err
	}
	endpointUrl := fmt.Sprintf("https://%s.execute-api.%s.amazonaws.com/dev/", *values.Id, region)
	return *parentResource.Items[0].Id, *values.Id, endpointUrl, nil
}

func IntegrateLambdaWithRestApi(svc *apigateway.Client, restApiID, parentResourceID, functionName, accountId, zone string) error {
	uriStr := fmt.Sprintf("arn:aws:apigateway:%s:lambda:path/2015-03-31/functions/arn:aws:lambda:%s:%s:function:venafi-snowflake-func-%s/invocations", zone, zone, accountId, functionName)
	resource, err := svc.CreateResource(context.TODO(), &apigateway.CreateResourceInput{
		RestApiId: aws.String(restApiID),
		ParentId:  aws.String(parentResourceID),
		PathPart:  aws.String(functionName),
	})
	if err != nil {
		return err
	}
	_, err = svc.PutMethod(context.TODO(), &apigateway.PutMethodInput{
		AuthorizationType: aws.String("AWS_IAM"),
		HttpMethod:        aws.String("POST"),
		ResourceId:        aws.String(*resource.Id),
		RestApiId:         aws.String(restApiID),
	})
	if err != nil {
		return err
	}
	_, err = svc.PutIntegration(context.TODO(), &apigateway.PutIntegrationInput{
		RestApiId:             aws.String(restApiID),
		ResourceId:            resource.Id,
		IntegrationHttpMethod: aws.String("POST"),
		HttpMethod:            aws.String("POST"),
		// RequestParameters:     map[string]string{"type": "string", "tpp_url": "string", "request_id": "string"},
		Type: gatewayTypes.IntegrationTypeAwsProxy,
		Uri:  aws.String(uriStr),
	})
	if err != nil {
		return err
	}

	return nil
}

func DeployRestAPI(svc *apigateway.Client, restApiID string) error {
	_, err := svc.CreateDeployment(context.TODO(), &apigateway.CreateDeploymentInput{
		RestApiId: aws.String(restApiID),
		StageName: aws.String("dev"),
	})
	if err != nil {
		return err
	}
	return nil
}

func GetRestApi(svc *apigateway.Client, restApiID string) error {
	_, err := svc.GetRestApi(context.TODO(), &apigateway.GetRestApiInput{
		RestApiId: aws.String(restApiID),
	})
	if err != nil {
		return err
	}
	return nil
}
