module github.com/starschema/snowflake-venafi-connector

go 1.16

replace github.com/snowflake-venafi-connector/go-lambda/src/request_cert => ./request_cert

require (
	github.com/Venafi/vcert/v4 v4.14.1
	github.com/aws/aws-lambda-go v1.23.0
	github.com/palette-software/go-log-targets v0.0.0-20200609204140-16fbfda0867a
	github.com/zfjagann/golang-ring v0.0.0-20210116075443-7c86fdb43134 // indirect
)
