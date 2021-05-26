package main

import (
	"context"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/certificate"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type VenafiConnectorConfig struct {
	AccessToken string `json:"token,omitempty"`
	TppURL      string `json:"tppUrl,omitempty"`
	Zone        string `json:"zone,omitempty"`
	UPN         string `json:"upn,omitempty"`
	DNSName     string `json:"dnsName,omitempty"`
	CommonName  string `json:"	,omitempty"`
}

type SnowFlakeType struct {
	Data [][]interface{} `json:"data,omitempty"`
}

func RequestCert(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

	var dataForRequestCert VenafiConnectorConfig
	var snowflakeData SnowFlakeType
	err := json.Unmarshal([]byte(request.Body), &snowflakeData)
	if err != nil {
		fmt.Printf("Failed to unmarshal snowflake value: %v ", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	dataForRequestCert.TppURL = fmt.Sprintf("%v", snowflakeData.Data[0][1])
	dataForRequestCert.AccessToken = fmt.Sprintf("%v", snowflakeData.Data[0][2])
	dataForRequestCert.DNSName = fmt.Sprintf("%v", snowflakeData.Data[0][3]) // TODO: UPN, DNS should allow multiple values
	dataForRequestCert.Zone = fmt.Sprintf("%v", snowflakeData.Data[0][4])
	dataForRequestCert.UPN = fmt.Sprintf("%v", snowflakeData.Data[0][5])
	dataForRequestCert.CommonName = fmt.Sprintf("%v", snowflakeData.Data[0][6])

	config := &vcert.Config{
		ConnectorType: endpoint.ConnectorTypeTPP,
		BaseUrl:       dataForRequestCert.TppURL,
		Zone:          dataForRequestCert.Zone,
		Credentials: &endpoint.Authentication{
			AccessToken: dataForRequestCert.AccessToken},
	}

	c, err := vcert.NewClient(config)
	if err != nil {
		fmt.Printf("Failed to connect to endpoint: %v ", err) // TODO: use logger
		return events.APIGatewayProxyResponse{                // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	var enrollReq = &certificate.Request{}

	enrollReq = &certificate.Request{
		Subject: pkix.Name{
			CommonName:         dataForRequestCert.CommonName,
			Organization:       []string{"Starschema"},
			OrganizationalUnit: []string{"Team Software Dev"},
			Locality:           []string{"Salt Lake"},
			Province:           []string{"Salt Lake"},
			Country:            []string{"US"},
		},
		UPNs:        []string{dataForRequestCert.UPN},
		DNSNames:    []string{dataForRequestCert.DNSName},
		CsrOrigin:   certificate.LocalGeneratedCSR,
		KeyType:     certificate.KeyTypeRSA,
		KeyLength:   2048,
		ChainOption: certificate.ChainOptionRootLast,
		KeyPassword: "newPassw0rd!",
	}
	//
	// 1.2. Generate private key and certificate request (CSR) based on request's options
	//
	err = c.GenerateRequest(nil, enrollReq)
	if err != nil {
		fmt.Printf("Failed to generate request: %v ", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	//
	// 1.3. Submit certificate request, get request ID as a response
	//
	requestID, err := c.RequestCertificate(enrollReq)
	if err != nil {
		fmt.Printf("Failed to request certificate: %v ", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}
	// fmt.Printf("Successfully submitted certificate request. Will pickup certificate by ID: %s", requestID)
	// body, err := json.Marshal(requestID)
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       fmt.Sprintf("{'data': [[0, '%v']]}", requestID),
		StatusCode: 200,
	}, nil
}

func main() {
	// lambda.Start(GetCertificate)
	lambda.Start(RequestCert)
}
