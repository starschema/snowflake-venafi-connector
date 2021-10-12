package utils

import (
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/assert"
)

func TestCreateConnectorSuccess(t *testing.T) {
	e := events.APIGatewayProxyRequest{
		HTTPMethod: "POST",
		Path:       "/getmachineid",
		Body:       `{"data": [[0,"TLS","https://test-venafi-tpp-server-url.com","\\example\\requestID"]]}`,
	}
	configParams, requestParams := ParseSnowflakeParameters(e, GET_MID_TYPE)
	assert.Equal(t, MachineIDTypeTLS, requestParams.MachineIDType)
	assert.Equal(t, "\\\\example\\\\requestID", requestParams.RequestID)
	assert.Equal(t, "https://test-venafi-tpp-server-url.com", configParams.TppURL)
	_, err := NewVenafiConnector(configParams)
	assert.Nil(t, err)
}
