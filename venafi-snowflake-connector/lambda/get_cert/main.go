package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/starschema/snowflake-venafi-connector/lambda/utils"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/certificate"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/palette-software/go-log-targets"
)

func GetCert(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	log.AddTarget(os.Stdout, log.LevelDebug)

	var dataForRequestCert utils.VenafiConnectorConfig
	var snowflakeData utils.SnowFlakeType
	err := json.Unmarshal([]byte(request.Body), &snowflakeData)
	if err != nil {
		log.Errorf("Failed to unmarshal snowflake parameters: %s", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	dataForRequestCert.TppURL = fmt.Sprintf("%v", snowflakeData.Data[0][1])
	dataForRequestCert.AccessToken = fmt.Sprintf("%v", snowflakeData.Data[0][2])
	escaped_pickupID := strings.Replace(fmt.Sprintf("%v", snowflakeData.Data[0][3]), "\\", "\\\\", -1)
	dataForRequestCert.RequestID = escaped_pickupID

	config := &vcert.Config{
		ConnectorType: endpoint.ConnectorTypeTPP,
		BaseUrl:       dataForRequestCert.TppURL,
		Credentials: &endpoint.Authentication{
			AccessToken: dataForRequestCert.AccessToken},
	}

	c, err := vcert.NewClient(config)
	if err != nil {
		log.Errorf("Failed to connect to endpoint: %s", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("{'data': [[0, '%v']]}", err.Error()),
			StatusCode: 500,
		}, nil
	}

	pickupReq := &certificate.Request{
		PickupID: dataForRequestCert.RequestID,
		Timeout:  180 * time.Second,
	}

	pcc, err := c.RetrieveCertificate(pickupReq)
	if err != nil {
		log.Errorf("Could not get certificate: %s", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("{'data': [[0, '%v']]}", err.Error()),
			StatusCode: 500,
		}, nil
	}

	bytes, err := json.Marshal(pcc)
	if err != nil {
		log.Errorf("Failed to serialize certificate: %v", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("{'data': [[0, '%v']]}", err.Error()),
			StatusCode: 500,
		}, nil
	}

	log.Infof("Retrieving certificate was succesful")
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       fmt.Sprintf("{'data': [[0, '%v']]}", string(bytes)),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(GetCert)
}
