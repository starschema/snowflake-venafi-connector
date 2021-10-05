package utils

import (
	"context"
	"crypto/x509/pkix"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/certificate"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	log "github.com/palette-software/go-log-targets"
)

type venafiConnector struct {
	client endpoint.Connector
}
type RequestMachineIDResponse struct {
	Certificate string `json:"Certificate"`
	PrivateKey  string `json:"PrivateKey"`
	Passphrase  string `json:"Passphrase"`
	RequestID   string `json:"RequestID"`
}

type VenafiConnector interface {
	RequestMachineID(ctx context.Context, commonName, upn, dns string) (string, error)
	GetMachineID(ctx context.Context, requestID string) (string, error)
	ListMachineIDs(ctx context.Context) (string, error)
	RevokeMachineIDs(ctx context.Context, requestID string, disable bool) (string, error)
	RenewMachineIDs(ctx context.Context, requestID string) (string, error)
	GetMachineIDStatus(ctx context.Context, commonName string) (string, error)
}

func createSnowflakeResponse(data string) string {
	return fmt.Sprintf("{'data': [[0, '%v']]}", data)
}

func createSnowflakeResponseWithEscape(data []byte) string {
	escapedStr := strings.ReplaceAll(string(data), "\\", "\\\\")
	c := fmt.Sprintf("{'data': [[0, '%v']]}", escapedStr) // we can only return one row
	return c
}

func NewVenafiConnector(configParams ConfigParameters) (*venafiConnector, error) {

	accessToken, err := GetAccessToken(configParams.TppURL)
	if err != nil {
		log.Errorf("Failed to get accesss token: %s", err)
		return nil, err
	}
	config := &vcert.Config{
		ConnectorType: endpoint.ConnectorTypeTPP,
		BaseUrl:       configParams.TppURL,
		Zone:          configParams.Zone,
		Credentials: &endpoint.Authentication{
			AccessToken: accessToken},
	}

	client, err := vcert.NewClient(config)
	if err != nil {
		log.Errorf("Failed to create venafi connector")
		return &venafiConnector{}, nil
	}
	return &venafiConnector{
		client: client,
	}, nil
}

func generateRandomKeyPassword() string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZÅÄÖ" +
		"abcdefghijklmnopqrstuvwxyzåäö" +
		"0123456789" +
		"+/?_#$!~>~@#$&*(")
	length := 8
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	str := b.String() // E.g. "ExcbsVQs"
	return str
}

func (c *venafiConnector) RequestMachineID(ctx context.Context, cn string, upn []string, dns []string) (string, error) {
	password := generateRandomKeyPassword()
	enrollReq := &certificate.Request{
		Subject: pkix.Name{
			CommonName: cn,
		},
		UPNs:        upn,
		DNSNames:    dns,
		CsrOrigin:   certificate.LocalGeneratedCSR,
		KeyType:     certificate.KeyTypeRSA,
		KeyLength:   2048,
		KeyPassword: password,
	}
	err := c.client.GenerateRequest(nil, enrollReq)
	if err != nil {
		log.Errorf("Failed to generate request: %v ", err)
		return createSnowflakeResponse(err.Error()), nil // return nil here, because we would like to see the error in Snowflake
	}

	requestID, err := c.client.RequestCertificate(enrollReq)
	if err != nil {
		log.Errorf("Failed to request certificate:: %v ", err)
		return createSnowflakeResponse(err.Error()), nil // return nil here, because we would like to see the error in Snowflake
	}

	enrollReq.PickupID = requestID
	enrollReq.Timeout = 180 * time.Second
	pcc, err := c.client.RetrieveCertificate(enrollReq)
	if err != nil {
		return createSnowflakeResponse(err.Error()), nil // return nil here, because we would like to see the error in Snowflake
	}

	pcc.AddPrivateKey(enrollReq.PrivateKey, []byte(enrollReq.KeyPassword))
	escaped_requestID := strings.Replace(fmt.Sprintf("%v", requestID), "\\", "\\\\", -1)
	responseObject := RequestMachineIDResponse{Certificate: pcc.Certificate, PrivateKey: pcc.PrivateKey, Passphrase: enrollReq.KeyPassword, RequestID: escaped_requestID}
	data, err := json.Marshal(responseObject)
	if err != nil {
		return createSnowflakeResponse(err.Error()), err
	}

	// Transform data to a form which is readable by Snowflake
	return createSnowflakeResponseWithEscape(data), nil
}

func (c *venafiConnector) GetMachineID(ctx context.Context, requestID string) (string, error) {
	pickupReq := &certificate.Request{
		PickupID: requestID,
		Timeout:  180 * time.Second,
	}

	pcc, err := c.client.RetrieveCertificate(pickupReq)
	if err != nil {
		log.Errorf("Could not get certificate: %s", err)
		return createSnowflakeResponse(err.Error()), nil
	}

	bytes, err := json.Marshal(pcc)
	if err != nil {
		log.Errorf("Failed to serialize certificate: %v", err)
		return createSnowflakeResponse(err.Error()), err
	}
	return createSnowflakeResponseWithEscape(bytes), nil
}

func (c *venafiConnector) ListMachineIDs(ctx context.Context) (string, error) {
	certList, err := c.client.ListCertificates(endpoint.Filter{})
	if err != nil {
		log.Errorf("Failed to list certificates: %s", err)
		return createSnowflakeResponse(err.Error()), nil
	}
	bytes, err := json.Marshal(certList)
	if err != nil {
		log.Errorf("Failed to serialize certificate: %v", err)
		return createSnowflakeResponse(err.Error()), err
	}
	log.Info("Sucessfully called List Certificates")
	// Transform data to a form which is readable by Snowflake
	return createSnowflakeResponseWithEscape(bytes), nil
}

func (c *venafiConnector) RevokeMachineID(ctx context.Context, requestID string, disable bool) (string, error) {
	revokeReq := &certificate.RevocationRequest{
		CertificateDN: requestID,
		Disable:       disable,
	}

	err := c.client.RevokeCertificate(revokeReq)
	if err != nil {
		log.Errorf("Failed to revoke cert: %v", err)
		return createSnowflakeResponse(err.Error()), nil
	}
	return createSnowflakeResponse(requestID), nil
}

func (c *venafiConnector) RenewMachineID(ctx context.Context, requestID string) (string, error) {
	renewReq := &certificate.RenewalRequest{
		CertificateDN: requestID,
	}

	requestID, err := c.client.RenewCertificate(renewReq)
	if err != nil {
		log.Errorf("Failed to renew certificate: %v", err)
		return createSnowflakeResponse(err.Error()), nil
	}
	return createSnowflakeResponse(requestID), nil
}

func (c *venafiConnector) GetMachineIDStatus(ctx context.Context, cn string) (string, error) {
	enrollReq := &certificate.Request{
		Subject: pkix.Name{
			CommonName: cn},
	}
	// Request a new certificate using Venafi API
	err := c.client.GenerateRequest(nil, enrollReq)
	if err != nil {
		log.Errorf("Failed to generate request: %v ", err)
		return createSnowflakeResponse(err.Error()), nil
	}

	log.Info("Generate request was successful")
	// Request a new certificate using Venafi API
	_, err = c.client.RequestCertificate(enrollReq)
	if err != nil {
		if strings.Contains(err.Error(), "disabled") {
			return createSnowflakeResponse("Certificate is disabled"), nil
		} else {
			log.Errorf("Failed to get status of certificate: %v ", err)
			return createSnowflakeResponse(err.Error()), nil
		}
	}
	return createSnowflakeResponse("Certificate is enabled"), nil
}
