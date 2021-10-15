# snowflake-venafi-connector

This integration allows you to manage your machine identities directly from Snowflake with the power of External Functions and AWS Lambdas

In the current version, six Venafi REST API endpoints are integrated. You can use this integration to:

* Request a new machine identitiy
* Pick up your machine identity
* List your machine identitites from the TPP server
* Get status of a machine identity
* Revoke a machine identity
* Renew a machine identity

## Table of content

* [Usage with Examples](#usage-with-examples)
* [Components](#components)
* [Installation](#installation)
    - [Preqreuisites](#prerequisites)
    - [Automatic installation](#automated-install)
    - [Manual installation](#install-manually-using-aws-console)
* [Security and important notes](#security-and-important-notes)
* [Troubleshoot](#troubleshoot)
* [Uninstall the integration](#uninstall)

## Usage with examples

Once the solution is installed in your environment, you can use native Snowflake functions to call the Venafi TPP system.
**Note:** you can only manage **TLS** certificates in the Venafi system using Snowflake.

The following Snowflake function calls will be available:

 * **REQUEST_MACHINE_ID**: Requests a new certificate with a private key
    *Parameters (must be provided in this order):*
    - **type** (string): The type of the certificate to generate. As of now, only **TLS** is supported
    - **tpp_url** (string): The URL of the Venafi TPP system. Example: https://test.env.cloudshare.com
    - **upn** (string array): The UPN (User Principal name) of the certificate to generate. Example: APP6-SAN.VENAFIDEMO.COM,TEST.VENAFIDEMO.COM
    - **zone** (string): The Zone in the TPP system in which to generate the certificate in
    - **dns** (string array): The DNS names (SANs) that the certificate should be valid for
    - **common_name** (string): The Common Name of the certificate to generate

    *Example:*
    ```
    SELECT
        JSON_REQUEST_CERT:Certificate AS CERT,
        JSON_REQUEST_CERT:PrivateKey AS PRIVATE_KEY
    FROM (
        SELECT PARSE_JSON(
            REQUEST_MACHINE_ID
                ('TLS',
                '<TPP URL>',
                ARRAY_CONSTRUCT('APP6-SAN.VENAFIDEMO.COM','TEST.VENAFIDEMO.COM'),
                'ZONE\\WHERE\\CERT\\SHOULD\\BE',
                ARRAY_CONSTRUCT('www.mydomain.com'),
                'TESTING-CERT-NAME'
                )
            ) JSON_REQUEST_CERT
        );
    ```
* **GET_MACHINE_ID**: Gets the certificate (only the public component)
    *Parameters (must be provided in this order):*
    - **type** (string): The type of the certificate to generate. As of now, only **TLS** is supported
    - **tpp_url** (string): The URL of the Venafi TPP system. Example: https://test.env.cloudshare.com
    - **request_id** (string): The ID of the certificate

    *Example:*
    ```
    SELECT
    JSON_CERT:Certificate
    FROM (
        SELECT PARSE_JSON(
                GET_MACHINE_ID(
                    'TLS',
                    '<TPP_URL>',
                    '<\\VED\\REQUEST_ID\\OF\\CERTIFICATE>')
        ) JSON_CERT);
    ```
* **GET_MACHINE_ID_STATUS**: Gets the status of a certificate (enabled/disabled)
    *Parameters (must be provided in this order):*
    - **type** (string): The type of the certificate to generate. As of now, only **TLS** is supported
    - **tpp_url** (string): The URL of the Venafi TPP system. Example: https://test.env.cloudshare.com
    - **zone** (string): The Zone in the TPP system in which to generate the certificate in

    *Example:*
    ```
    SELECT GET_MACHINE_ID_STATUS('TLS', <tpp_url>, <zone>, <common_name>);
    ```
* **LIST_MACHINE_IDS**: Lists the Machine IDs in a Zone
    *Parameters (must be provided in this order):*
    - **type** (string): The type of the certificate to generate. As of now, only **TLS** is supported
    - **tpp_url** (string): The URL of the Venafi TPP system. Example: https://test.env.cloudshare.com
    - **zone** (string): The Zone in the TPP system in which to generate the certificate in

    *Example:*
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
* **RENEW_MACHINE_ID**: Renews a certificate
    *Parameters (must be provided in this order):*
    - **type** (string): The type of the certificate to generate. As of now, only **TLS** is supported
    - **tpp_url** (string): The URL of the Venafi TPP system. Example: https://test.env.cloudshare.com
    - **request_id** (string): The ID of the certificate

    *Example:*
    ```
    SELECT RENEW_MACHINE_ID('TLS', '<tpp_url>', '<request_id>');
    ```
* **REVOKE_MACHINE_ID**: Revokes a certificate
    *Parameters (must be provided in this order):*
    - **type** (string): The type of the certificate to generate. As of now, only **TLS** is supported
    - **tpp_url** (string): The URL of the Venafi TPP system. Example: https://test.env.cloudshare.com
    - **request_id** (string): The ID of the certificate

    *Example:*
    ```
    SELECT REVOKE_MACHINE_ID('TLS', '<tpp_url>', '<request_id>', <should-disable>);
    ```


## Components
The solution consists of the following AWS & Snowflake components:
- Six lambda functions will be deployed each for every Snowflake External Function. These functions are wrappers around Venafi's vCert Go SDK. The lambda functions are written in [GoLang](https://golang.org/).
- An AWS Api Gateway that provides a REST interface to call the Snowflake functions.
- An S3 Bucket. A TPP credentials file will be stored on it that is read by the lambda functions.
- Two AWS Roles:
    1. A role to allow Snowflake to call the AWS lambdas using the REST API
    2. A role to allow the AWS lambdas to read the credentials file stored on the S3 bucket.
- A Snowflake [api integration](https://docs.snowflake.com/en/sql-reference/sql/create-api-integration.html#usage-notes), that can call the AWS REST API
- Six Snowfalke [external functions](https://docs.snowflake.com/en/sql-reference/external-functions-introduction.html), which will call their respective AWS lambda to retrieve and submit information.

## Installation
The software can be installed in 2 ways:
- [Automatic installation](#automated-install)
- [Manual installation](#install-manually-using-aws-console)

The guide below outlines the steps to be taken for both.
### Prerequisites


**Snowflake Prerequisites:**
For installation a Snowflake, a database and a user account is needed with permissions to create [external functions](https://docs.snowflake.com/en/sql-reference/external-functions-introduction.html) and [api integrations](https://docs.snowflake.com/en/sql-reference/sql/create-api-integration.html#usage-notes).

**AWS Prerequisites:**
1. An AWS account with the following permissions:

    * List buckets
    * Create buckets
    * List AWS Lambda functions
    * Create AWS Lambda functions
    * Create role
2. AWS Profile set up on the machine where the installation is running from

### Automated Install

A Command Line (CLI) tool is provided along with the installation package that takes care of the deployment of these functions both to AWS and Snowflake.

**Features**

* Check the current status of your integration
* Install AWS Lambdas and Snowflake External Functions to your environment
* Get credentials for Venafi Rest API if needed

**Commands**

- *getcreds* - Get access token from Venafi Rest API with vcert-sdk client id

- *install* - Install the External Functions and Lambdas to your environment

- *state* - Show the current state of the integration, check the missing components

**Preqreuisites**

Make sure you have a valid aws credential file in your ~/.aws folder. The credential file has to contain a region config. You should select a region where you would like to install your bucket and Lambdas.
Example aws credential file:

```
[venafi-snowflake-connector]
aws_access_key_id=DEMOKEYID
aws_secret_access_key=DEMOKEY
region=eu-west-1
```

**Usage**

1. Clone the repository with `git clone https://github.com/starschema/snowflake-venafi-connector`
2. `cd connector/cli_tool/main`
3. There is an example_config.yml file which needs to be filled with your AWS, Snowflake and Venafi TPP server. This config file will be used by the installer to create the components of the integration. See the comments in the file.
4. Run `go run . status --file=<path-to-your-config>` to check the current status of your integration. This will give you a detailed description in which components are installed, and which not. Before the first install none of the components should be installed.
5. Run `go run . install --file=<path-to-your-config>` Command to install the external functions to the configured Snowflake database and AWS Lambdas
6. Run `go run . status --file=<path-to-your-config>` once again to check if all the components are available.

**Install steps**
The tool performs the following installation steps:

1. An AWS configuration is created by using your Aws profile.

2. If the bucket you provided does not exist, the installer will create an S3 bucket and upload a json file to it which contains your refresh token, access token and date of the token expiration.

3. The installer creates a Lambda execution role and give permission to it to access the bucket and to write logs in Cloudwatch and to execute the function.

4. The installer creates a role which later will be set to the Rest Api to allow to call the execute API from Snowflake

5. The installer creates an AWS Rest Api and assign the created role to it.

6. The installer uploads to executables from the repository and create a Lambda for each function. The Lambdas are added under a POST method as resources to the Rest Api

7. In Snowflake an api integration is created.

8. The External ID from Snowflake is added to the Rest Api!s role. This allows Snowflake to call the AWS Lambdas through the Rest Api.

9. Snowflake functions are created on the api integration.


### Install Manually Using AWS Console

You can create the components of the integration manually yourself using the AWS Console and Snowflake Management Interface.
This allows you to perform each installation step yourself, which gives you the flexibility to apply stricter IAM control, oblige with naming conventions and other standards.

#### Installation steps

In this walkthrough we will install the "REQUEST_MACHINE_ID" external function.

1. Clone repository and get the executables

    1. `git clone https://github.com/starschema/snowflake-venafi-connector`

    2. Each Lambda function will need a zip file with an executable. These executables can be found under the cli_tool/main/bin/handlers folder. You can use these executables or you can build them for yourself:

        ```
        cd connector/main/lambda/request_machine_id
        GOOS=linux GOARCH=amd64  go build -o /path/to/new/executable/requestmachineid
        ```

2. Log in to AWS Console: https://console.aws.amazon.com

3. Create an empty role for Snowflake. You will add this role to the API Gateway to implement the ability to call the AWS Lambdas from Snowflake
    1. To the iAM console
    2. Open the Roles tab
    3. Create an empty role

4. Create an AWS Lambda. You will have to do this six times for all the lambdas. [Getting started with functions](https://docs.aws.amazon.com/lambda/latest/dg/getting-started-create-function.html).

    1. On the Lambda/functions console, click to "Create Function" button
    2. On the Basic Information page name your function (e.g: request-machine-id), select Go.1 as a runtime, and choose **Create a role with basic lambda permissions**.
    3. Now you arrived to your new function's page. Under the code section, In the cli_tool/main/bin/handlers you can find the executables for the functions. Compress and upload the requestmachineid executable. Lambda only allows zip file format to upload.
    4. **Important:** Under the Runtime Settings click "Edit". Change the handler "hello" to the name of the executable. In this case it is "requestmachineid".
    5. If you click to the "Test" section you can run the new Lambda function, and you should get an error like "Failed to get access token". This is fine for now, all we want to see is the underlying code was triggered.
    6. Go to the configuration page and click on "Environment Variables". You have to set up these 2 environment variables:
        ```
        S3_BUCKET: <name of the S3 bucket where the credential file is tored>
        CREDENTIAL_FILE_NAME: <name of the credential file>
        ZONE: <name of the region where functions and S3 bucket are created>
        ```
5.  Create S3 Bucket with Venafi credentials

    1. On the S3 console create a new bucket to store your Venafi credentials by following this documentation: https://docs.aws.amazon.com/AmazonS3/latest/userguide/creating-bucket.html
    2. Upload a json file which will store your access token and refresh token to Venafi TPP server. The file should look like the following:
        ```
        [
            {
            "Url": <url>
            "AccessToken": <access_token>,
            "AccessTokenExpires": <expiration_date, example: 2022-01-06T11:39:59Z>,
            "RefreshToken": <refresh_token>
            }
        ]
        ```
        **NOTE**: If VCert command line tool is used to get credentials for the first time please make sure to require credentials with the flag --client-id 'vcert-sdk'
6. Change Lambda role to allow access to the bucket

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
        1. Go back to the Lambda function config page
        2. Click on permissions and click on the lambda role. This will take you the the iAM console Roles page.
        3. Assign the new permission to your role.

    **NOTE:** This method will result in 6 different roles for the 6 lambas. Instead, you may want to create a single custom role with access to Cloudwatch and the S3 Bucket, and assign that role to all AWS Lambas.  More information about creating roles for AWS Lambdas: https://docs.aws.amazon.com/lambda/latest/dg/lambda-intro-execution-role.html

7. Create an API Gateway

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



8. Set up Api Integration and External functions in Snowflake

    1. Ceate an api integration with the following query:
        ```
        create or replace api integration venafi_manual
            api_provider = aws_api_gateway
            api_aws_role_arn = '<arn-of-your-role>'
            api_allowed_prefixes = ('https://<id-of-your-rest-api>.execute-api.<region>.amazonaws.com/<stage>/')
            enabled = true;
        ```
        *Advanced:* If you would like to order your endpoints otherwise for example under a resource called "venafi-integration" the query would be:
        ```
        create or replace api integration venafi_manual
            api_provider = aws_api_gateway
            api_aws_role_arn = '<arn-of-your-role>/snowflake'
            api_allowed_prefixes = ('https://<id-of-your-rest-api>.execute-api.<region>.amazonaws.com/<stage>/venafi-integration/')
            enabled = true;
        ```
    2. Create an external function to request machine id with the following query:
        ```
        create external function REQUEST_MACHINE_ID(type varchar, tpp_url varchar, dns array, zone varchar, upn array, common_name varchar)
            returns variant
            api_integration = venafi_manual
            as 'https://<id-of-rest-api>.execute-api.eu-west-1.amazonaws.com/dev/requestmachineid'
        ```
    3. Run `describe api_integration <your_integration_name>`
        From the result set you will need API_AWS_IAM_USER_ARN and API_AWS_EXTERNAL_ID
    4. Go back to your AWS console to the role you created for Snowflake.  Click on 'Edit Trust Relationship' and change the json to this:

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

Installation is done. Make sure that:

 1. Your Lambda execution role has permission to create CloudWatch logs and to access your bucket
 2. You added Invoke permission for your Api Gateway
 3. The API_AWS_IAM_USER_ARN and API_AWS_EXTERNAL_ID are set in the Snowflake role policy.
 4. You created the S3 BUCKET and uploaded a valid credential json file.
 5. You added the 3 needed environment variables to your lambda

After verifying, you can run your first query to request a machine id:

```SELECT
    JSON_REQUEST_CERT:Certificate AS CERT,
    JSON_REQUEST_CERT:PrivateKey AS PRIVATE_KEY
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


## Security and important notes

* If don't want to put your Venafi credentials into the config file, you can create a bucket and upload it manually with the AWS console. This way the cli tool will skip the credential file creation step. You still need to provide the name of your uploaded file and a bucket name created by you. Please check the Install Manually section to see the steps for uploading credentials.
* You can set up additional security to your bucket in the AWS Console. Useful link: https://docs.aws.amazon.com/AmazonS3/latest/userguide/UsingEncryption.html
* The installer will create two roles in AWS. The first role will be the execute role for the AWS Lambdas. This role will contains permissions to invoke the function, to write CloudWatch logs, and to access the S3 bucket where Venafi credentials are stored.
* All Snowflake Functions will use the same role
* The second role will be used by the AWS Rest API and the Snowflake API integration and will give a permission to call the AWS execute api from Snowflake.
* There might be cases when you would like to modify the permission configuration of the Lambdas or re-upload your credentials, etc. Every step from the automatic installation can be done manually from AWS Console and Snowflake.

## Troubleshoot

If the automatic installation fails, run a status command to see which components are missing. You can create them manually by following the Manual Tutorial or you can try to re-run an install. In the bucket there will be a deployment info file, which will provide you information about your existing components. Make sure that you use the proper ID's during manual install.

Make sure that your AWS user has permission to create bucket, list buckets, create lambdas, list lambdas.

If the installation was successful but the Snowflake functions are still not working, check the logs of the function in the AWS Console Cloudwatch.

Please reach out with any question you might have.

## Uninstall

1. To uninstall the integration you have to remove the components manually. First you can simply drop the components in Snowflake. Example:

```
 drop integration venafi_integration

 drop function GET_MACHINE_ID(VARCHAR, VARCHAR, VARCHAR)
 drop function LIST_MACHINE_IDS(VARCHAR, VARCHAR, VARCHAR)
 drop function REVOKE_MACHINE_ID(VARCHAR, VARCHAR, VARCHAR, BOOLEAN)
 drop function RENEW_MACHINE_ID(VARCHAR, VARCHAR, VARCHAR)
 drop function REQUEST_MACHINE_ID(VARCHAR, VARCHAR, ARRAY, VARCHAR,ARRAY,VARCHAR)
 drop function GET_MACHINE_ID_STATUS(VARCHAR, VARCHAR, VARCHAR, VARCHAR)

 drop function GET_MID(VARCHAR, VARCHAR, VARCHAR)
 drop function LIST_MIDS(VARCHAR, VARCHAR, VARCHAR)
 drop function REVOKE_MID(VARCHAR, VARCHAR, VARCHAR, BOOLEAN)
 drop function RENEW_MID(VARCHAR, VARCHAR, VARCHAR)
 drop function REQUEST_MID(VARCHAR, VARCHAR, ARRAY, VARCHAR,ARRAY,VARCHAR)
 drop function GET_MID_STATUS(VARCHAR, VARCHAR, VARCHAR, VARCHAR)
```

2. In your AWS Console remove the deployed AWS Lambdas functions. If you used the automated install the prefix for these functions is "venafi-snowflake-func".
3. In your S3 bucket you can find a deployment.info file. This will contain the IDs of your components, so you can make sure all
4. Remove the created Venafi Rest Api on the API Gateway tab.
5. Remove the two roles which were created (arn / name in deployment.info file)
6. Remove the policy which is attached to your lambda execution role.
7. Remove the files from your S3 bucket, then delete the bucket itself.
