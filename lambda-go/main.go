package main

import (
	"context"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func handleRequest(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	jsonMock := "{'data': [[0, 'test-value']]}"
	return events.APIGatewayProxyResponse{Body: string(jsonMock), StatusCode: 200}, nil
}
func main() {
	lambda.Start(handleRequest)
}
