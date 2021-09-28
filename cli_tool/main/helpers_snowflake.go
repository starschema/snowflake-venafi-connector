package main

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/snowflakedb/gosnowflake"
)

const SNOWFLAKE_FUNCTION_NAME_GETMACHINEID = "getmachineid"
const SNOWFLAKE_FUNCTION_NAME_REQUESTMACHINEID = "requestmachineid"
const SNOWFLAKE_FUNCTION_NAME_LISTMACHINEIDS = "listmachineids"
const SNOWFLAKE_FUNCTION_NAME_RENEWMACHINEID = "renewmachineid"
const SNOWFLAKE_FUNCTION_NAME_REVOKEMACHINEID = "revokemachineid"
const SNOWFLAKE_FUNCTION_NAME_GETMACHINEIDSTATUS = "getmachineidstatus"

func getConnectionStringFromParams(username, password, account, warehouse, database, schema, role string) string {
	return fmt.Sprintf("%s:%s@%s-%s/%s/%s?my_warehouse=%s&role=%s", username, "7^kJuS!$QLVzPy~_", account, account, database, schema, warehouse, role)
}

func CheckSnowflakeConnection(conf SnowflakeOptions) error {
	connStr := getConnectionStringFromParams(conf.Username, conf.Password, conf.Account, conf.Warehouse, conf.Database, conf.Schema, conf.Role)

	db, err := sql.Open("snowflake", connStr)
	if err != nil {
		return err
	}
	defer db.Close()
	return nil
}

func CreateSnowflakeApiIntegration(integrationName string, awsRoleARN string, endpointUrl string, conf SnowflakeOptions) (externalID string, iamARN string, err error) {
	connStr := getConnectionStringFromParams(conf.Username, conf.Password, conf.Account, conf.Warehouse, conf.Database, conf.Schema, conf.Role)
	db, err := sql.Open("snowflake", connStr)
	if err != nil {
		return "", "", err
	}
	defer db.Close()
	sql := fmt.Sprintf(`create or replace api integration venafi_integration api_provider = aws_api_gateway api_aws_role_arn = '%s' enabled = true api_allowed_prefixes = ('%s')`, awsRoleARN, endpointUrl)
	_, err = db.Exec(sql)
	if err != nil {
		return "", "", err
	}
	sql = fmt.Sprintf(`describe integration venafi_integration`)
	rows, err := db.Query(sql)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	final := make(map[string]string)
	for rows.Next() {
		var a string
		var p string
		var e string
		var b string
		if err := rows.Scan(&a, &p, &e, &b); err != nil {
			log.Fatal(err)
		}
		final[a] = e
	}
	if err != nil {
		return "", "", err
	}

	return final["API_AWS_EXTERNAL_ID"], final["API_AWS_IAM_USER_ARN"], nil //TODO qUERY return values
}

func CreateSnowflakeFunction(functionName string, endpoint string, conf SnowflakeOptions) {
	path := endpoint + functionName
	connStr := getConnectionStringFromParams(conf.Username, conf.Password, conf.Account, conf.Warehouse, conf.Database, conf.Schema, conf.Role)
	var paramStr string
	switch functionName {
	case SNOWFLAKE_FUNCTION_NAME_GETMACHINEID:
		paramStr = "(type varchar, tpp_url varchar, request_id varchar)"
	case SNOWFLAKE_FUNCTION_NAME_RENEWMACHINEID:
		paramStr = "(type varchar, tpp_url varchar, request_id varchar)"
	case SNOWFLAKE_FUNCTION_NAME_REVOKEMACHINEID:
		paramStr = "(type varchar, tpp_url varchar, request_id varchar)"
	case SNOWFLAKE_FUNCTION_NAME_REQUESTMACHINEID:
		paramStr = "(type varchar, tpp_url varchar, dns varchar, zone varchar, upn varchar, common_name varchar)"
	case SNOWFLAKE_FUNCTION_NAME_LISTMACHINEIDS:
		paramStr = "(type varchar, tpp_url varchar, zone varchar)"
	case SNOWFLAKE_FUNCTION_NAME_GETMACHINEIDSTATUS:
		paramStr = "(type varchar, tpp_url varchar, zone varchar, common_name varchar)"
	default:
		fmt.Printf("############## invalid function name: %v", functionName)
	}

	db, err := sql.Open("snowflake", connStr)
	if err != nil {
		log.Fatal("Failed to create function: " + err.Error())
		return
	}
	defer db.Close()
	sql := fmt.Sprintf(`
			create or replace external function %s %s
			returns variant
			api_integration = venafi_integration
			COMPRESSION = none
			as '%s'`, functionName, paramStr, path)
	_, err = db.Exec(sql)
	if err != nil {
		log.Fatal("Failed to create function: " + err.Error())
	}
}

func GetSnowflakeFunction(functionName string, conf SnowflakeOptions) (notFoundError, generalError error) {
	functionName = strings.ToUpper(functionName)
	connStr := getConnectionStringFromParams(conf.Username, conf.Password, conf.Account, conf.Warehouse, conf.Database, conf.Schema, conf.Role)
	db, err := sql.Open("snowflake", connStr)
	if err != nil {
		return nil, err
	}
	defer db.Close()
	sqlStatement := fmt.Sprintf(`select function_name from information_schema.functions where function_name like '%s'`, functionName)
	var resultInterface interface{}
	err = db.QueryRow(sqlStatement).Scan(&resultInterface)
	if err == sql.ErrNoRows || resultInterface == nil {
		return err, nil
	}

	if err != nil {
		return nil, err
	}
	return nil, nil
}

func manageSnowflakeFunction(functionName string, status StatusResult) {
	if status.State < 2 {
		return
	}
}

func getSnowflakeFunctionStatus(conf SnowflakeOptions, functionName string, state *FunctionCheckState) StatusResult {
	notFoundErr, generalErr := GetSnowflakeFunction(functionName, conf)
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
