package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/palette-software/go-log-targets"
	"github.com/starschema/snowflake-venafi-connector/lambda/utils"
)

func RevokeMachineID(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	log.AddTarget(os.Stdout, log.LevelDebug)

	connectorParams := utils.ParseSnowflakeParameters(request, utils.REVOKE_MID_TYPE)

	client, err := utils.NewVenafiConnector(connectorParams)
	if err != nil {
		log.Errorf("Failed to create venafi client from snowflake parameters: %v", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}
	snowflakeResponse, err := client.RevokeMachineID(connectorParams.RequestID, connectorParams.Disable)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       snowflakeResponse,
			StatusCode: 500,
		}, err
	}
	log.Infof("Successfully revoked certificate: %s", connectorParams.RequestID)
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       snowflakeResponse,
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(RevokeMachineID)
}
