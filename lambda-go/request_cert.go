package main

import (
	"context"
	"crypto/x509/pkix"
	"fmt"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/certificate"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

type StuffNeededForRequestCerts struct {
	AccessToken string `json:"token,omitempty"`
	TppURL      string `json:"tppUrl,omitempty"`
	Zone        string `json:"zone,omitempty"`
	UPN         string `json:"upn,omitempty"`
	DNSName     string `json:"dnsName,omitempty"`
	CommonName  string `json:"	,omitempty"`
}


func RequestCert(ctx context.Context, request StuffNeededForRequestCerts) (events.APIGatewayProxyResponse, error) {

	dataForRequestCert := request
	//
	// 0. Get client instance based on connection config
	//
	config := &vcert.Config{
		ConnectorType: endpoint.ConnectorTypeTPP,
		BaseUrl:       dataForRequestCert.TppURL,
		Zone:          dataForRequestCert.Zone,
		Credentials: &endpoint.Authentication{
			AccessToken: dataForRequestCert.AccessToken},
	}
	fmt.Printf("TPP URL: %s", config.BaseUrl)

	//config := cloudConfig
	//config := mockConfig
	c, err := vcert.NewClient(config)
	if err != nil {
		fmt.Printf("Failed to connect to endpoint: %v ", err)
		return events.APIGatewayProxyResponse{ // Error HTTP response
			Body:       err.Error(),
			StatusCode: 500,
		}, nil
	}

	//
	// 1.1. Compose request object
	//
	//Not all Venafi Cloud providers support IPAddress and EmailAddresses extensions.
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
		Body:       requestID,
		StatusCode: 200,
	}, nil
}

func main() {
	lambda.Start(RequestCert)
}
