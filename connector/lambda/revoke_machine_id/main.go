package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/starschema/snowflake-venafi-connector/lambda/utils"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/certificate"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/palette-software/go-log-targets"
)

func RevokeMachineID(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

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

	machineIDType := fmt.Sprintf("%v", snowflakeData.Data[0][1])
	if machineIDType != utils.MachineIDTypeTLS {
		machineIDType = utils.MachineIDTypeTLS // Currently only TLS requests are supported.
	}
	log.Infof("Type of the machine id to request: %s", machineIDType)

	dataForRequestCert.TppURL = fmt.Sprintf("%v", snowflakeData.Data[0][2])
	escaped_requestID := strings.Replace(fmt.Sprintf("%v", snowflakeData.Data[0][3]), "\\", "\\\\", -1)
	shouldDisable, err := strconv.ParseBool(fmt.Sprintf("%v", snowflakeData.Data[0][4]))
	if err != nil {
		log.Errorf("Failed to parse disable request property from Snowflake parameters: %s", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}

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

	revokeReq := &certificate.RevocationRequest{
		CertificateDN: dataForRequestCert.RequestID,
		Disable:       shouldDisable,
	}

	err = c.RevokeCertificate(revokeReq)
	if err != nil {
		return events.APIGatewayProxyResponse{
			Body:       fmt.Sprintf("{'data': [[0, '%v']]}", err.Error()),
			StatusCode: 500,
		}, err
	}
	log.Infof("Successfully revoked certificate: %s", dataForRequestCert.RequestID)
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       fmt.Sprintf("{'data': [[0, '%s']]}", dataForRequestCert.RequestID),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(RevokeMachineID)
}
