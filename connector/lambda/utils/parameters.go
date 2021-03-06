package utils

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	log "github.com/palette-software/go-log-targets"
)

type ConfigParameters struct {
	TppURL string
	Zone   string
}

type RequestParameters struct {
	RequestID     string
	UPN           []string
	Disable       bool
	MachineIDType string
	CommonName    string
	DNSName       []string
}

func snowflakeInterfaceToStrArray(snowflakeArr interface{}) []string {
	intefaceStr := fmt.Sprintf("%v", snowflakeArr)
	trimmed := strings.TrimPrefix(intefaceStr, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	resultArr := strings.Split(trimmed, ",")
	return resultArr
}

func ParseSnowflakeParameters(request events.APIGatewayProxyRequest, queryType string) (ConfigParameters, RequestParameters) {
	var snowflakeData SnowFlakeType
	var configParameters ConfigParameters
	var requestParameters RequestParameters
	err := json.Unmarshal([]byte(request.Body), &snowflakeData)
	if err != nil {
		log.Errorf("Failed to unmarshal snowflake parameters: %s", err)
		return configParameters, requestParameters
	}
	snowflakeParams := snowflakeData.Data[0]
	requestParameters.MachineIDType = fmt.Sprintf("%v", snowflakeParams[1])
	if requestParameters.MachineIDType != MachineIDTypeTLS {
		requestParameters.MachineIDType = "TLS" // this is not used yets, probably we will use it to request other machine id types
	}
	configParameters.TppURL = fmt.Sprintf("%v", snowflakeParams[2])
	switch queryType {
	case LIST_MID_TYPE:
		configParameters.Zone = fmt.Sprintf("%v", snowflakeParams[3])
	case GET_MID_TYPE:
		requestParameters.RequestID = strings.Replace(fmt.Sprintf("%v", snowflakeData.Data[0][3]), "\\", "\\\\", -1)
	case GET_STATUS_MID_TYPE:
		configParameters.Zone = fmt.Sprintf("%v", snowflakeParams[3])
		requestParameters.CommonName = fmt.Sprintf("%v", snowflakeParams[4])

	case REQUEST_MID_TYPE:
		requestParameters.DNSName = snowflakeInterfaceToStrArray(snowflakeParams[3])
		configParameters.Zone = fmt.Sprintf("%v", snowflakeParams[4])
		requestParameters.UPN = snowflakeInterfaceToStrArray(snowflakeParams[5])
		requestParameters.CommonName = fmt.Sprintf("%v", snowflakeParams[6])
	case RENEW_MID_TYPE:
		requestParameters.RequestID = strings.Replace(fmt.Sprintf("%v", snowflakeParams[3]), "\\", "\\\\", -1)
	case REVOKE_MID_TYPE:
		requestParameters.RequestID = strings.Replace(fmt.Sprintf("%v", snowflakeParams[3]), "\\", "\\\\", -1)
		shouldDisable, err := strconv.ParseBool(fmt.Sprintf("%v", snowflakeParams[4]))
		if err != nil {
			log.Errorf("Failed to parse disable request property from Snowflake parameters: %s", err)
			requestParameters.Disable = false
		} else {
			requestParameters.Disable = shouldDisable
		}
	}
	return configParameters, requestParameters
}
