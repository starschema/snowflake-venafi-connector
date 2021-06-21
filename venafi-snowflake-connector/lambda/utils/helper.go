package utils

import (
	"fmt"

	"github.com/Venafi/vcert/v4"
	"github.com/Venafi/vcert/v4/pkg/endpoint"
	log "github.com/palette-software/go-log-targets"
)

func CreateVenafiConnectorFromParameters(dataFromSnowflake SnowFlakeType, token string) (*endpoint.Connector, error) {
	config := &vcert.Config{
		ConnectorType: endpoint.ConnectorTypeTPP,
		BaseUrl:       fmt.Sprintf("%v", dataFromSnowflake.Data[0][1]),
		Zone:          fmt.Sprintf("%v", dataFromSnowflake.Data[0][3]),
		Credentials: &endpoint.Authentication{
			AccessToken: token},
	}
	// Create a new Connector for Venafi API calls
	c, err := vcert.NewClient(config)
	if err != nil {
		log.Errorf("Failed to create vcert client: %v", err)
		return nil, err
	}
	return &c, nil
}
