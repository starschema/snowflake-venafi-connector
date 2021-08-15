package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	log "github.com/palette-software/go-log-targets"
)

type SnowflakeParameters struct {
	MachineIDType string
	TppURL        string
	RequestID     string
	Zone          string
	UPN           string
	CommonName    string
	DNSName       string
	Disable       bool
}

func ParseSnowflakeParameters(request events.APIGatewayProxyRequest, queryType string) SnowflakeParameters {
	var snowflakeData SnowFlakeType
	var connectorParameters SnowflakeParameters
	err := json.Unmarshal([]byte(request.Body), &snowflakeData)
	if err != nil {
		log.Errorf("Failed to unmarshal snowflake parameters: %s", err)
		return connectorParameters
	}
	snowflakeParams := snowflakeData.Data[0]
	connectorParameters.MachineIDType = fmt.Sprintf("%v", snowflakeParams[1])
	if connectorParameters.MachineIDType != MachineIDTypeTLS {
		connectorParameters.MachineIDType = "TLS" // this is not used yets, probably we will use it to request other machine id types
	}
	connectorParameters.TppURL = fmt.Sprintf("%v", snowflakeParams[2])
	switch queryType {
	case LIST_MID_TYPE:
		connectorParameters.Zone = fmt.Sprintf("%v", snowflakeParams[3]) // TODO: UPN, DNS should allow multiple values
	case GET_MID_TYPE:
		connectorParameters.RequestID = strings.Replace(fmt.Sprintf("%v", snowflakeData.Data[0][3]), "\\", "\\\\", -1)
	case GET_STATUS_MID_TYPE:
		connectorParameters.Zone = fmt.Sprintf("%v", snowflakeParams[3])
		connectorParameters.CommonName = fmt.Sprintf("%v", snowflakeParams[4])

	case REQUEST_MID_TYPE:
		connectorParameters.DNSName = fmt.Sprintf("%v", snowflakeParams[3])
		connectorParameters.Zone = fmt.Sprintf("%v", snowflakeParams[4])
		connectorParameters.UPN = fmt.Sprintf("%v", snowflakeParams[5])
		connectorParameters.CommonName = fmt.Sprintf("%v", snowflakeParams[6])
	case RENEW_MID_TYPE:
		connectorParameters.RequestID = strings.Replace(fmt.Sprintf("%v", snowflakeParams[3]), "\\", "\\\\", -1)
	case REVOKE_MID_TYPE:
		connectorParameters.RequestID = strings.Replace(fmt.Sprintf("%v", snowflakeParams[3]), "\\", "\\\\", -1)
		shouldDisable, err := strconv.ParseBool(fmt.Sprintf("%v", snowflakeParams[4]))
		if err != nil {
			log.Errorf("Failed to parse disable request property from Snowflake parameters: %s", err)
			connectorParameters.Disable = false
		} else {
			connectorParameters.Disable = shouldDisable
		}
	}
	return connectorParameters
}
