package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/certificate"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/palette-software/go-log-targets"
)

type VenafiConnectorConfig struct {
	AccessToken string `json:"token,omitempty"`
	TppURL      string `json:"tppUrl,omitempty"`
	Zone        string `json:"zone,omitempty"`
	UPN         string `json:"upn,omitempty"`
	DNSName     string `json:"dnsName,omitempty"`
	RequestID   string `json:"requestID,omitempty"`
}

type SnowFlakeType struct {
	Data [][]interface{} `json:"data,omitempty"`
}

func GetCert(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	var dataForRequestCert VenafiConnectorConfig
	var snowflakeData SnowFlakeType
	err := json.Unmarshal([]byte(request.Body), &snowflakeData)
	if err != nil {
		log.Errorf("Failed to unmarshal snowflake parameters: %s", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	escaped_zone := strings.Replace(fmt.Sprintf("%v", snowflakeData.Data[0][3]), "\\", "\\\\", -1)
	escaped_pickupID := strings.Replace(fmt.Sprintf("%v", snowflakeData.Data[0][4]), "\\", "\\\\", -1)

	dataForRequestCert.TppURL = fmt.Sprintf("%v", snowflakeData.Data[0][1])
	dataForRequestCert.AccessToken = fmt.Sprintf("%v", snowflakeData.Data[0][2])
	dataForRequestCert.Zone = escaped_zone
	dataForRequestCert.RequestID = escaped_pickupID

	config := &vcert.Config{
		ConnectorType: endpoint.ConnectorTypeTPP,
		BaseUrl:       dataForRequestCert.TppURL,
		Zone:          dataForRequestCert.Zone,
		Credentials: &endpoint.Authentication{
			AccessToken: dataForRequestCert.AccessToken},
	}

	c, err := vcert.NewClient(config)
	if err != nil {
		fmt.Printf("Failed to connect to endpoint: %s", err)
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
		fmt.Printf("Could not get certificate: %s", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("{'data': [[0, '%v']]}", err.Error()),
			StatusCode: 500,
		}, nil
	}

	log.Infof("Retrieving certificate was succesfull")
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       fmt.Sprintf("{'data': [[0, '%v']]}", pcc),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(GetCert)
}
