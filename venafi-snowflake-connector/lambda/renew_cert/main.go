package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/starschema/snowflake-venafi-connector/lambda/utils"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/certificate"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/palette-software/go-log-targets"
)

func RenewCertificate(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

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
	escaped_requestID := strings.Replace(fmt.Sprintf("%v", snowflakeData.Data[0][2]), "\\", "\\\\", -1)
	dataForRequestCert.RequestID = escaped_requestID

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

	renewReq := &certificate.RenewalRequest{
		CertificateDN: dataForRequestCert.RequestID,
	}

	requestID, err := c.RenewCertificate(renewReq)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("{'data': [[0, '%v']]}", err.Error()),
			StatusCode: 500,
		}, err
	}
	log.Infof("Successfully renewed certificate %s", requestID)
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       fmt.Sprintf("{'data': [[0, '%v']]}", requestID),
		StatusCode: 200,
	}, err
}

func main() {
	lambda.Start(RenewCertificate)
}
