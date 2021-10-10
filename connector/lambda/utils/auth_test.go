package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_shouldRequestNewToken(t *testing.T) {
	testTPPUrl := "http://test-tpp-url.com"

	tokenListWithValidToken := []credentialJSON{
		map[string]string{
			"Url":                testTPPUrl,
			"AccessToken":        "test-access-token",
			"RefreshToken":       "test-refresh-token",
			"AccessTokenExpires": "2021-12-28T16:17:19Z",
		},
	}

	mapWithExpiredToken := []credentialJSON{
		map[string]string{

			"Url":                testTPPUrl,
			"AccessToken":        "test-access-token2",
			"RefreshToken":       "test-refresh-token2",
			"AccessTokenExpires": "2020-12-28T16:17:19Z",
		},
	}

	mapWithInvalidTPPUrl := []credentialJSON{
		map[string]string{
			"Url":                "",
			"AccessToken":        "test-access-token3",
			"RefreshToken":       "test-refresh-token3",
			"AccessTokenExpires": "2021-12-28T16:17:19Z",
		},
	}

	_, shouldRequestNewtoken, err := shouldRequestNewToken(tokenListWithValidToken, testTPPUrl)
	assert.Nil(t, err)
	assert.False(t, shouldRequestNewtoken)

	_, shouldRequestNewtoken, err = shouldRequestNewToken(mapWithExpiredToken, testTPPUrl)
	assert.Nil(t, err)
	assert.True(t, shouldRequestNewtoken)

	_, shouldRequestNewtoken, err = shouldRequestNewToken(mapWithInvalidTPPUrl, testTPPUrl)
	assert.NotNil(t, err)
	assert.False(t, shouldRequestNewtoken)
	assert.True(t, strings.Contains(err.Error(), "TPP"))
}
