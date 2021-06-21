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

func RequestCert(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

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

	// Parse parameters sent by Snowflake from Lambda Event
	dataForRequestCert.DNSName = fmt.Sprintf("%v", snowflakeData.Data[0][2]) // TODO: UPN, DNS should allow multiple values
	dataForRequestCert.UPN = fmt.Sprintf("%v", snowflakeData.Data[0][3])
	dataForRequestCert.CommonName = fmt.Sprintf("%v", snowflakeData.Data[0][4])

	log.Infof("Finished parse parameters from event object")

	// Get access token from S3. If access token is expired, generate a new one.
	accessToken, err := utils.GetAccessToken(dataForRequestCert.TppURL)
	if err != nil {
		log.Errorf("Failed to get accesss token: %s", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}

	log.Info("Got valid access token from S3")

	config := &vcert.Config{
		ConnectorType: endpoint.ConnectorTypeTPP,
		BaseUrl:       dataForRequestCert.TppURL,
		Zone:          dataForRequestCert.Zone,
		Credentials: &endpoint.Authentication{
			AccessToken: accessToken},
	}
	// Create a new Connector for Venafi API calls
	c, err := utils.CreateVenafiConnectorFromParameters(snowflakeData, accessToken)
	if err != nil {
		log.Errorf("Failed to create venafi client %s", err)
		return events.APIGatewayProxyResponse{
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}

	var enrollReq = &certificate.Request{}

	enrollReq = &certificate.Request{
		Subject: pkix.Name{
			CommonName: dataForRequestCert.CommonName,
		},
		UPNs:     []string{dataForRequestCert.UPN},
		DNSNames: []string{dataForRequestCert.DNSName},
	}
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
	requestID, err := c.RequestCertificate(enrollReq)
	if err != nil {
		log.Errorf("Failed to request certificate:: %v ", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}
	log.Infof("Certificate request was successful. RequestID is: %s", requestID)

	escaped_requestID := strings.Replace(fmt.Sprintf("%v", requestID), "\\", "\\\\", -1)
	// Transform data to a form which is readable by Snowflake
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       fmt.Sprintf("{'data': [[0, '%v']]}", escaped_requestID),
		StatusCode: 200,
	}, err
}

func main() {
	lambda.Start(RequestCert)
}
