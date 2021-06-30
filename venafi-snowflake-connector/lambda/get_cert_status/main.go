package main

import (
	"context"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/certificate"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/palette-software/go-log-targets"
	"github.com/starschema/snowflake-venafi-connector/lambda/utils"
)

func GetCertStatus(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	log.AddTarget(os.Stdout, log.LevelDebug)

	var dataForRequestCert utils.VenafiConnectorConfig
	var snowflakeData utils.SnowFlakeType
	err := json.Unmarshal([]byte(request.Body), &snowflakeData)
	if err != nil {
		log.Errorf("Failed to unmarshal snowflake parameters: %s", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}

	dataForRequestCert.TppURL = fmt.Sprintf("%v", snowflakeData.Data[0][1])
	dataForRequestCert.Zone = fmt.Sprintf("%v", snowflakeData.Data[0][2])
	dataForRequestCert.CommonName = fmt.Sprintf("%v", snowflakeData.Data[0][3])

	accessToken, err := utils.GetAccessToken(dataForRequestCert.TppURL)
	if err != nil {
		log.Errorf("Failed to get accesss token: %s", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}

	config := &vcert.Config{
		ConnectorType: endpoint.ConnectorTypeTPP,
		BaseUrl:       dataForRequestCert.TppURL,
		Zone:          dataForRequestCert.Zone,
		Credentials: &endpoint.Authentication{
			AccessToken: accessToken},
	}

	c, err := vcert.NewClient(config)
	if err != nil {
		fmt.Printf("Failed to connect to endpoint: %s", err)
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("{'data': [[0, '%v']]}", err.Error()),
			StatusCode: 500,
		}, err
	}

	enrollReq := &certificate.Request{
		Subject: pkix.Name{
			CommonName: dataForRequestCert.CommonName},
	}
	// Request a new certificate using Venafi API
	err = c.GenerateRequest(nil, enrollReq)
	if err != nil {
		log.Errorf("Failed to generate request: %v ", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}

	log.Info("Generate request was successful")
	// Request a new certificate using Venafi API
	_, err = c.RequestCertificate(enrollReq)
	if err != nil {
		if strings.Contains(err.Error(), "disabled") {
			return events.APIGatewayProxyResponse{ // Success HTTP response
				Body:       fmt.Sprintf("{'data': [[0, '%v']]}", "Certificate is disabled"),
				StatusCode: 200,
			}, nil
		} else {
			log.Errorf("Failed to get status of certificate: %v ", err)
			return events.APIGatewayProxyResponse{ // Error HTTP response
				Body:       err.Error(),
				StatusCode: 500,
			}, err
		}
	}
	// Transform data to a form which is readable by Snowflake
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       fmt.Sprintf("{'data': [[0, '%v']]}", "Certificate is enabled"),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(GetCertStatus)
}
