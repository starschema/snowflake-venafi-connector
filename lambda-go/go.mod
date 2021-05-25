module github.com/starschema/snowflake-venafi-connector

go 1.14

replace github.com/snowflake-venafi-connector/go-lambda/src/request_cert => ./request_cert

require (
	github.com/Venafi/vcert v3.18.4+incompatible
	github.com/Venafi/vcert/v4 v4.14.1
	github.com/aws/aws-lambda-go v1.23.0
)
