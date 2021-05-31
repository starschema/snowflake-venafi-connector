package main

import (
	"context"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"strings"

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
	CommonName  string `json:"commonName,omitempty"`
	RequestID   string `json:"commonName,omitempty"`
}

type SnowFlakeType struct {
	Data [][]interface{} `json:"data,omitempty"`
}

func RequestCert(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {

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
		log.Errorf("Failed to connect to endpoint: %s", err)
		return events.APIGatewayProxyResponse{
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
		log.Errorf("Failed to generate request: %v ", err)
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
		log.Errorf("Failed to request certificate:: %v ", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}
	log.Infof("Certificate request was successful. RequestID is: %s", requestID)
	escaped_requestID := strings.Replace(fmt.Sprintf("%v", requestID), "\\", "\\\\", -1)
	return events.APIGatewayProxyResponse{ // Success HTTP response
		Body:       fmt.Sprintf("{'data': [[0, '%v']]}", escaped_requestID),
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(RequestCert)
}
