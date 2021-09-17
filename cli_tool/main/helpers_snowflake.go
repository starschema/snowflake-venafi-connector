package main

import (
	"database/sql"
	"fmt"

	_ "github.com/snowflakedb/gosnowflake"
)

const SNOFLAKE_FUNCTION_NAME_GETMACHINEID = "GET_MACHINE_ID"
const SNOFLAKE_FUNCTION_NAME_REQUESTMACHINEID = "REQUEST_MACHINE_ID"
const SNOFLAKE_FUNCTION_NAME_LISTMACHINEIDS = "LIST_MACHINEI_DS"
const SNOFLAKE_FUNCTION_NAME_RENEWMACHINEID = "RENEW_MACHINE_ID"
const SNOFLAKE_FUNCTION_NAME_REVOKEMACHINEID = "REVOKE_MACHINE_ID"
const SNOFLAKE_FUNCTION_NAME_GETMACHINEIDSTATUS = "GET_MACHINE_ID_STATUS"

func CheckSnowflakeConnection() error {
	db, err := sql.Open("snowflake", "")
	if err != nil {
		return err
	}
	defer db.Close()
	return nil
}

func CreateSnowflakeApiIntegration(integrationName string, awsRoleARN string, endpointUrl string) (externalID string, iamARN string, err error) {
	db, err := sql.Open("snowflake", "")
	if err != nil {
		return "", "", err
	}
	defer db.Close()
	sql := fmt.Sprintf(`create or replace api integration cica api_provider = aws_api_gateway api_aws_role_arn = '%s' enabled = true api_allowed_prefixes = ('%s')`, awsRoleARN, endpointUrl)
	result, err := db.Exec(sql)
	fmt.Printf("result: %v", result)
	if err != nil {
		return "", "", err
	}

	return "", "", nil //TODO qUERY return values
}

func CreateSnowflakeFunction(functionName string, endpoint string, path string) error {
	db, err := sql.Open("snowflake", "")
	if err != nil {
		return err
	}
	defer db.Close()
	sql := fmt.Sprintf(`
			create or replace external function <functionname> <parameters>
			returns variant
			api_integration = <integrationName>
			COMPRESSION = none
			as <endpoint>/<path>`)
	result, err := db.Exec(sql)
	fmt.Printf("result: %v", result)
	if err != nil {
		return err
	}

	return nil
}

func GetSnowflakeFunction(functionName string) (notFoundError, generalError error) {
	db, err := sql.Open("snowflake", "")
	if err != nil {
		return nil, err
	}
	defer db.Close()
	sqlStatement := `SHOW EXTERNAL FUNCTIONS` //todo
	_, err = db.Query(sqlStatement)
	switch err {
	case sql.ErrNoRows:
		fmt.Println("No rows were returned!")
		return err, nil
	case nil:
		return nil, nil
	default:
		return nil, err
	}
}

func manageSnowflakeFunction(functionName string, status StatusResult) {
	if status.State < 2 {
		return
	}
}

func getSnowflakeFunctionStatus(conf SnowflakeOptions, functionName string, state *FunctionCheckState) StatusResult {
	notFoundErr, generalErr := GetSnowflakeFunction(functionName)
	if notFoundErr != nil {
		state.AnyMissing = true
		return StatusResult{State: 3, Error: notFoundErr}
	} else if generalErr != nil {
		state.AnyError = true
		return StatusResult{State: 2, Error: generalErr}
	} else {
		return StatusResult{State: 1, Error: nil}
	}
}
