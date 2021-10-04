# CLI Developer documentation

The CLI tool provides an easy deploy for Venafi Snowflake Integration.
Features:

* Check the current status of your integration
* Install AWS Lambdas and Snowflake External Functions to your environment
* Get credentials for Venafi Rest API if needed



### Commands
- *getcreds* - Get access token from Venafi Rest API with vcert-sdk client id
- *install* - Get the status of the integration components in your environment
- *state* - Show the


### Components

The CLI installs the following components:

**AWS**
- An S3 bucket will be created that store the secure Venafi credentials that the AWS Lamdbas will use. No other connections will be allowed.
-

### Installation steps