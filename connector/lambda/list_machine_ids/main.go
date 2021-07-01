package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	log "github.com/palette-software/go-log-targets"
	"github.com/starschema/snowflake-venafi-connector/lambda/utils"
)

func ListMachineIDs(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

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

	// Parse parameters sent by Snowflake from Lambda Event
	dataForRequestCert.TppURL = fmt.Sprintf("%v", snowflakeData.Data[0][2])
	dataForRequestCert.Zone = fmt.Sprintf("%v", snowflakeData.Data[0][3]) // TODO: UPN, DNS should allow multiple values

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
	c, err := vcert.NewClient(config)
	if err != nil {
		log.Errorf("Failed to create new client")
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}

	certList, err := c.ListCertificates(endpoint.Filter{})
	if err != nil {
		log.Errorf("Failed to list certificates: %s", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, err
	}
	log.Info("Sucessfully called List Certificates")
	// Transform data to a form which is readable by Snowflake
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       fmt.Sprintf("{'data': [[0, '%v']]}", certList),
		StatusCode: 200,
	}, err
}

func main() {
	lambda.Start(ListMachineIDs)
}
