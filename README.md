# snowflake-venafi-connector
Helper library with samples to make Venafi services available as Snowflake functions

## Prerequisites for Deployment and Usage

* Serverless Tool - `npm install serverless -g`
* Serverless Snowflake Plugin - `npm install serverless-snowflake-external-function-plugin`

The deployment was tested with node version: v14.16.0

### AWS

An AWS account is required with a valid zone configuration. AWS credentials should be stored in ~/.aws/credentials.
An S3 bucket in AWS where a json file stored with Venafi credentials.

A role which has permission both to the Lambda functions and to access S3.
More information: [https://docs.aws.amazon.com/IAM/latest/UserGuide/id_roles_create.html]

Valid format of the credential file:
[
     {
      "access_token": <access_token>,
      "access_token_expires": <expiration_date>,
      "refresh_token": <refresh_token>
      "url": <url>
     }
]
**NOTE**: If VCert command line tool is used to get credentials for the first time please make sure to require credentials with the flag --client-id 'vcert-sdk'

### Environment Variables

Three environment variables should be set for each function. These are:
```
 S3_BUCKET: <name of the S3 bucket where the credential file is tored>
 CREDENTIAL_FILE_NAME: <name of the credential file>
 ZONE: <name of the zone where functions and S3 bucket are created>
 ```

## Snowflake

Snowflake account with a role which grant permission to create, delete, describe, etc. external functions and api integrations.
Add Snowflake credentials for the serverless.yml file:
```service: venafi-snowflake-connector
plugins:
  - serverless-snowflake-external-function-plugin

custom:
  snowflake:
    role: role
    account: account
    username: user
    password: ****
    warehouse: warehouse
    database: testdb
    schema: public
```
### Deployment
#### Example deployment of GET_MACHINE_ID external function
1. `cd venafi-snowflake-connector/lambda/get_machine_id` # cd to source code folder
2. Modify main.go, and save the file.
2. `GOOS=linux GOARCH=amd64  go build -o ../../bin/handlers/getmachineid .` # create an executable to the handlers folder
3. `cd ...` # cd back to serverless.yml
4. `serverless deploy`

### Usage

**NOTE: Currently only TLS type is supported.**
The defined external functions can be called from Snowflake, usually with the tpp url as parameter, and the requestID of the machine identity.
Exmaple:
`select GETMACHINEID('TLS, 'https://example-tpp-url', '\\VED\\Policy\\Example/test.cert');`
`select RENEWMACHINEID('TLS', 'https://example-tpp-url', '\\VED\\Policy\\Example/test.cert');`

To revoke a machine identity with retire:
`select REVOKEMACHINEID('TLS','https://example-tpp-url', '\\VED\\Policy\\Example/test.cert', true);`
To simply revoke a machine identity:
`select REVOKEMACHINEID('TLS','https://example-tpp-url', '\\VED\\Policy\\Example/test.cert', false);`


To list machine identities in a specific zone:
`select LISTMACHINEIDS('TLS', 'https://example-tpp-url', '\Policy\\ExampleZone');`

To request a machine identity:
`select REQUESTMACHINEID(
    'TLS',
    'https://example-tpp-url',
                   'dns',
                   'zone',
                   'upn',
                   'new-md-common-name'
                  );`

To get a status of a machine identity:
`select GETMACHINEIDSTATUS('https://example-tpp-url',
                   'dns',
                   'zone',
                   'upn',
                   'new-md-common-name'
                  );`


### Troubleshoot:
If the deployment fails, check the credentials both in Snowflake and AWS. Make sure you have the valid config and credential file in ~/.aws folder. Make sure you uploaded the credential file, and the environemnt variables are set in AWS.

If execution fails:
Check the monitoring page of the function in AWS Lambda. The logs will provide information about the errors.

If you delete functions manually the stack of lambdas can change. Useful link to resolve: https://stackoverflow.com/questions/58382779/serverless-deploy-function-not-found-sls-deploy