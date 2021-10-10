# Venafi Integration For Snowflake

This integration allows you to manage your machine identities directly from Snowflake with the power of External Functions and AWS Lambdas

In the current version six Venafi REST API endpoints are integrated. You can use this integration to:

* Request a new machine identitiy
* Pick up your machine identity
* List your machine identitites from the TPP server
* Get status of a machine identity
* Revoke a machine identity
* Renew a machine identity

## Table of content

* [Prerequisites for install](#prerequisites)
* [Integration Components](#integration-components)
* [Usage with Examples](#usage-with-examples)
* [Install with Command Line Tool](#install-with-command-line-tool)
* [Install Manually using AWS Console](#install-manually-using-aws-console)

## Prerequisites

Integration was built and tested with Go version 1.16

For installation a Snowflake account needed where the user who would like to install the solution has permission to create external functions and api integrations.

An AWS account with the following permissions:

* List buckets
* Create buckets
* List AWS Lambda functions
* Create AWS Lambda functions
* Create role

## Integration Components

### AWS

* An S3 bucket in AWS where a json file stored with Venafi credentials.

* A role which has permission both to the Lambda functions and to access the bucket with the credentials.
More information: [https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create.html]

* A json file which will store the credentials for Venafi TPP servers. Multiple TPP server entries are allowed.
Valid format of the credential file:
```
[
    {
      "Url": <url>
      "AccessToken": <access_token>,
      "AccessTokenExpires": <expiration_date>,
      "RefreshToken": <refresh_token>
     }
]
```
**NOTE**: If VCert command line tool is used to get credentials for the first time please make sure to require credentials with the flag --client-id 'vcert-sdk'
### Lambda functions
Six lambda functions will be deployed each for every Snowflake External Function. These functions are wrappers around Venafi's vCert Go SDK.

### Environment Variables

Three environment variables should be set for each function. These are:
```
 S3_BUCKET: <name of the S3 bucket where the credential file is tored>
 CREDENTIAL_FILE_NAME: <name of the credential file>
 ZONE: <name of the region where functions and S3 bucket are created>
 ```

## Snowflake

In Snowflake the following external functions will be created:
 * REQUEST_MACHINE_ID (<type:string>, <tpp_url:string>, <upn:array>, <zone:string>, <dns:array>, <common_name:string>)
 * GET_MACHINE_ID (<type:string>, <tpp_url:string>, <request_id:string>)
 * GET_MACHINE_ID_STATUS (<type:string>, <tpp_url:string>, <zone:string)
 * RENEW_MACHINE_ID  (<type:string>, <tpp_url:string>, <zone:string)
 * REVOKE_MACHINE_ID  (<type:string>, <tpp_url:string>, <request_id:string)
 * LIST_MACHINE_IDS (<type:string>, <tpp_url:string>, <request_id:string)

### Usage with Examples

**NOTE: Currently only TLS type is supported.**
The functions will either return a json string or a single string. You can parse the needed values by using the built-in json parser from Snowflake.
#### Request a new Machine Identity

When you request a new machine identity you will get back the requested certificate with a private key and passphrase.
```
SELECT
    JSON_REQUEST_CERT:Certificate AS CERT,
    JSON_REQUEST_CERT:PrivateKey AS PRIVATE_KEY,
    JSON_REQUEST_CERT:passphrase AS PASSPHRASE
FROM (
    SELECT PARSE_JSON(
        REQUEST_MACHINE_ID
            ('TLS',
             '<TPP URL>',
             ARRAY_CONSTRUCT('APP6-SAN.VENAFIDEMO.COM','TEST.VENAFIDEMO.COM'),
             'ZONE\\WHERE\\CERT\\SHOULD\\BE',
              ARRAY_CONSTRUCT('TESTDNS@MAIL.COM'),
             'TESTING-CERT-NAME'
            )
        ) JSON_REQUEST_CERT
    );
```
#### Pick up a Machine Identity
When you request a new certificate the function will retrieve you the request id for the new cert. You can also list the request ids for existing certificates using LIST_MACHINE_IDS endpoint.
```
SELECT
    JSON_CERT:Certificate
FROM (
    SELECT PARSE_JSON(
            GET_MACHINE_ID(
                'TLS',
                '<TPP_URL>',
                '<REQUEST_ID>')
    ) JSON_CERT);
```
#### List Machine Identities from a specific zone
In this example we will select the name of the machine identitites in a specific zone, along with their IDs and the DNS names for those certificates. You can also search for a specific certificate in the list by adding a where clause.

```
SELECT
    LIST_MACHINE_JSON.VALUE:CN AS CERT_NAME,
    LIST_MACHINE_JSON.VALUE:ID AS CERTIFICATE_REQUEST_ID,
    LIST_MACHINE_JSON.VALUE:SANS:DNS AS DNS
FROM (
    LATERAL FLATTEN(
        SELECT PARSE_JSON(
            LIST_MACHINE_IDS(
                'TLS',
                '<TPP_URL>',
                '<ZONE_ON_TPP_SERVER>'
            )
        )
    ) LIST_MACHINE_JSON
  )
 WHERE CERT_NAME LIKE '<NAME-OF-THE-CERTIFICATE-YOU-ARE-LOOKING-FOR>'
```
#### Get status of a Machine Identity
With this function you can check if a machine identity is enabled or disabled. You will get a single string as a response.
To request a machine identity:
```
SELECT GET_MACHINE_ID_STATUS('TLS', <tpp_url>, <zone>, <common_name>);
```
#### Renew a Machine Identitiy

With this function you can renew a machine identity. If the renewal was successful you will get back the ID of the machine identity as a single string.
```
SELECT RENEW_MACHINE_ID('TLS', '<tpp_url>', '<request_id>');
```

#### Revoke a Machine Identitiy

With this function you can revoke a machine identity. With a boolean as a last parameter you can decide if you would like the machine identity to be disabled as well. Disabled machine identities cannot be renewed.

```
SELECT REVOKE_MACHINE_ID('TLS', '<tpp_url>', '<request_id>', <should-disable>);
```

## Install Manually Using AWS Console

You can create the components of the integration manually from AWS Console and directly from Snowflake. In this way you are able to recieve a more customied solution.

### Installation steps

In this walkthrough we will install the "REQUEST_MACHINE_ID" external function.

#### Clone repository and get the executables

`git clone https://github.com/starschema/snowflake-venafi-connector`
`cd connector/bin/handlers`
Here you can see the executables for the functions. These will be needed for the AWS Lambda creation. You can also build your own executable from code. Example:
`cd main/lambda/request_machine_id`
`GOOS=linux GOARCH=amd64  go build -o /path/to/new/executable/requestmachineid` command
To finish the next steps log in to AWS Console: https://console.aws.amazon.com

#### Create an empty role for Snowflake

Go to iAM console to Roles tab, and create a new empty role. This role will be added to you API Gateway and Snowflake api integration to implement the ability to call Lambdas from Snowflake.

#### Create AWS Lambda

Detailed documentation: https://docs.aws.amazon.com/lambda/latest/dg/getting-started-create-function.html

1. On Lambda/functions console click to "Create Function" button
2. On the Basic Information page name your function (e.g: request-machine-id), select Go.1 as a runtime, and choose "Create a role with basic lambda permissions".
3. Now you arrived to your new function's page. Under the code section, upload the requestmachineid executable from bin/handlers folder.
4. **Under the Runtime Settings click "Edit". Change the handler "hello" to the name of the executable. In this case it is "requestmachineid".**
5. If you click to the "Test" section you can run this new Lambda function, and you should get an error like "Failed to get access token". This is fine for now, all we want to see is the underlying code was triggered.
6. Go to the configuration page and click on "Environment Variables". You have to set up 3 [environment variables](##environment-variables).

#### Create S3 Bucket with Venafi credentials

1. On the S3 console create a new bucket to store your Venafi credentials by following this documentation: https://docs.aws.amazon.com/AmazonS3/latest/userguide/creating-bucket.html
2. Upload a json file which will store your access token and refresh token to Venafi TPP server. The file should look like [this](###aws)

#### Change Lambda role to allow access to the bucket

1. Go back to your Lambda function page and click on configuration, then permissions tab. You can see the current execution role of your lambda function. Click on the role.
2. Under the permissions tab you can see your Lambda has permission to write logs to CloudWatch. But it doesn't have a permission to access the S3 Bucket you created. You need to create a permission which allows you to access that specific bucket. Example json for permission:

```
{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Action": [
                "s3:ListBucket"
            ],
            "Resource": "arn:aws:s3:::<your-bucket-name>"
        },
        {
            "Effect": "Allow",
            "Action": [
                "s3:PutObject",
                "s3:GetObject"
            ],
            "Resource": "arn:aws:s3:::<you-bucket-name>/*"
        }
    ]
}
```

3. Assign this permission to your Lambda role.

Go back to your Lambda function configuration page, click on permissions and click on the lambda role. It will takes you to the iAM console roles page. You can assign the new permission to your role. If you follow this way, all lambdas will have their own roles. The other way is to create a custom role, with access to Cloudwatch, and access to the s3 bucket. And you can assign that single role to all of your Lambdas. In this way you only have to add the bucket permission once.
More information about creating roles for AWS Lambdas: https://docs.aws.amazon.com/lambda/latest/dg/lambda-intro-execution-role.html

#### Create an API Gateway

1. Go to API Gateway Console and create a Rest API. Name the Rest API and set endpoint type to Regional.  Click on actions and select "Add Resource". Add `requestmachineid` as a resource.
2. Click on "Create Method" and choose POST method. Select Use Lambda Proxy Integration, then add your function. Click on "Method Request" and add iAM authorization.
3. Fill and add the following json to the API resource policy:

```{
    "Version": "2012-10-17",
    "Statement": [
        {
            "Effect": "Allow",
            "Principal": {
                "AWS": "arn:aws:sts::<Account_ID>:assumed-role/snowflake"
            },
            "Action": "execute-api:Invoke",
            "Resource": "<POST_METHOD_REQUSET_ARN>"
        }
    ]
}
```

Now you can deploy your api to a specific stage such as "dev" or "prod". Once you deployed your Rest API you will get the endpoint url for it. The endpoint url is the same what we will use in the next step as api_allowed_prefixes
You are all set from AWS side for now!

#### Set up Api Integration and External functions in Snowflake

You can create an api integration with the following query:
```
create or replace api integration venafi_manual
    api_provider = aws_api_gateway
    api_aws_role_arn = '<arn-of-your-role>'
    api_allowed_prefixes = ('https://<id-of-your-rest-api>.execute-api.<region>.amazonaws.com/<stage>/')
    enabled = true;
```
If you would like to order your endpoints otherwise for example under a resource called "venafi-integration" the query would be:
```
create or replace api integration venafi_manual
    api_provider = aws_api_gateway
    api_aws_role_arn = '<arn-of-your-role>/snowflake'
    api_allowed_prefixes = ('https://<id-of-your-rest-api>.execute-api.<region>.amazonaws.com/<stage>/venafi-integration/')
    enabled = true;
```
You can create an external function to request machine id with the following query:
```
create external function REQUEST_MACHINE_ID(type varchar, tpp_url varchar, dns array, zone varchar, upn array, common_name varchar)
    returns variant
    api_integration = venafi_manual
    as 'https://<id-of-rest-api>.execute-api.eu-west-1.amazonaws.com/dev/requestmachineid'
```

Last step is to run `describe api_integration <your_integration_name>`
From the result set you will need API_AWS_IAM_USER_ARN and API_AWS_EXTERNAL_ID
Go back to your AWS console to the role you created for Snowflake.  Click on 'Edit Trust Relationship' and change the json to this:
```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "AWS": "<API_AWS_IAM_USER_ARN>"
      },
      "Action": "sts:AssumeRole",
      "Condition": {
        "StringEquals": {
          "sts:ExternalId": "API_AWS_EXTERNAL_ID"
        }
      }
    }
  ]
}
```

Now you are ready. Make sure that:

 1. Your Lambda execution role has permission to create CloudWatch logs and to access your bucket
 2. You added Invoke permission for your Api Gateway
 3. The API_AWS_IAM_USER_ARN and API_AWS_EXTERNAL_ID are set in the Snowflake role policy.
 4. You created the S3 BUCKET and uploaded a valid credential json file.
 5. You added the 3 needed environment variables to your lambda

If all good, you can run your first query to request a machine id:

```SELECT
    JSON_REQUEST_CERT:Certificate AS CERT,
    JSON_REQUEST_CERT:PrivateKey AS PRIVATE_KEY,
    JSON_REQUEST_CERT:passhprase AS PASSPHRASE
FROM (
    SELECT PARSE_JSON(
        REQUEST_MACHINE_ID
            ('TLS',
             '<TPP URL>',
             ARRAY_CONSTRUCT('APP6-SAN.VENAFIDEMO.COM','TEST.VENAFIDEMO.COM'),
             'ZONE\\WHERE\\CERT\\SHOULD\\BE',
              ARRAY_CONSTRUCT('TESTDNS@MAIL.COM'),
             'TESTING-CERT-NAME'
            )
        ) JSON_REQUEST_CERT
    );
```

To install all of the functions you have to repeat the Lambda function creation step, and you have to add a "POST" method for each of the resources in your Rest API configuration. **You do not have to create another Rest API in AWS and you do not need another api integration in Snowflake.**

You can rename the code zip files, the handlers and the resources, but make sure that the handler is matching with your uploaded zip name.

Useful resources about AWS Lambda and Snowflake integration:
https://docs.snowflake.com/en/sql-reference/external-functions-creating-aws-ui.html

## Install with Command Line Tool

The CLI tool provides an easy deploy for Venafi Snowflake Integration.
Features:

* Check the current status of your integration
* Install AWS Lambdas and Snowflake External Functions to your environment
* Get credentials for Venafi Rest API if needed

### Commands

- *getcreds* - Get access token from Venafi Rest API with vcert-sdk client id

- *install* - Get the status of the integration components in your environment

- *state* - Show the

### Prerequsites

Make sure you have a valid aws credential file in your ~/.aws folder. The credential file has to contain a region config. You should select a region where you would like to install your bucket and Lambdas.
Example aws credential file:

```
[venafi-snowflake-connector]
aws_access_key_id=DEMOKEYID
aws_secret_access_key=DEMOKEY
region=eu-west-1
```

### Usage

1. Clone the repository with `git clone https://github.com/starschema/snowflake-venafi-connector`
2. `cd connector/cli_tool/main`
3. There is an example_config.yml file which needs to be filled with your AWS, Snowflake and Venafi TPP server. This config file will be used by the installer to create the components of the integration. See the comments in the file.
4. Run `go run . status --file=<path-to-your-config>` to check the current status of your integration. This will give you a detailed description in which components are installed, and which not. Before the first install none of the components should be installed.
5. Run `go run . install --file=<path-to-your-config>` Command to install the external functions to the configured Snowflake database and AWS Lambdas
6. Run `go run . status --file=<path-to-your-config>` once again to check if all the components are available.

### Install steps

1. An AWS configuration will be created by using your Aws profile.

2. If the bucket you provided not exists the installer will create an S3 bucket and upload a json file to it which will contain your refresh token, access token and date of the token expiration.

3. The installer will create a Lambda execution role and give permission to it, to access the bucket, to write logs in Cloudwatch and to execute the function.

4. The installer will create a role which later will be set to the Rest Api to allow to call the execute API from Snowflake

5. The installer will create an AWS Rest Api and assign the created role to it.

6. The installer will upload to executables from the repository and create a Lambda for each function. The Lambdas will be added under a POST method as resources to the Rest Api

7. In Snowflake an api integration will be created.

8. The External ID from Snowflake will be added to the Rest Api!s role. This will allow Snowflake to call the AWS Lambdas through the Rest Api.

9. Snowflake functions will be created on the api integration.

All set!

## Security and important notes

* **If you would not like to put your Venafi credentials into the config file, you can create a bucket and upload it manually with the AWS console. In this way the cli tool will skip the credential file creation step. You still need to provide the name of your uploaded file and a bucket name created by you. Please check the Install Manually section to see the steps for uploading credentials.**
* **You can set up additional security to your bucket in the AWS Console. Useful link: https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingEncryption.html**
* **The installer will create two roles in AWS. The first role will be the execute role for the AWS Lambdas. This role will contains permissions to invoke the function, to write CloudWatch logs, and to access the S3 bucket where Venafi credentials are stored.**
* **All functions will use the same role**
* **The second role will be used by the AWS Rest API and the Snowflake API integration and will give a permission to call the AWS execute api from Snowflake.**
* **There might be cases when you would like to modify the permission configuration of the Lambdas or re-upload your credentials, etc. Every step from the automatic installation can be done manually from AWS Console and Snowflake.**

## Troubleshoot

If the automatic installation fails at one point, run a status command to see which components are missing. You can create them manually by following the Manual Tutorial or you can try to re-run an install. In the bucket there will be a deployment info file, which will provide you information about your existing components. Make sure that you use the proper ID's during manual install.

Make sure that your AWS user has permission to create bucket, list buckets, create lambdas, list lambdas.

If the installation was successful but the Snowflake functions are still not working, check the logs of the function in the AWS Console Cloudwatch.

Please reach out with any question you might have.

