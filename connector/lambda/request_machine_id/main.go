package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/palette-software/go-log-targets"
	"github.com/starschema/snowflake-venafi-connector/lambda/utils"
)

func RequestMachineID(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	log.AddTarget(os.Stdout, log.LevelDebug)

	connectorParams := utils.ParseSnowflakeParameters(request, utils.REQUEST_MID_TYPE)
	client, err := utils.NewVenafiConnector(connectorParams)
	if err != nil {
		log.Errorf("Failed to create venafi client from snowflake parameters: %v", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}
	snowflakeResponse, err := client.RequestMachineID(connectorParams.CommonName, connectorParams.UPN, connectorParams.DNSName)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       snowflakeResponse,
			StatusCode: 500,
		}, err
	}
	log.Infof("Successfully requested certificate: %s", connectorParams.RequestID)
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       snowflakeResponse,
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(RequestMachineID)
}
