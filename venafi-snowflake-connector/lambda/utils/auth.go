package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/Venafi/vcert/v4/pkg/venafi/tpp"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/palette-software/go-log-targets"
)

type credentialJSON []map[string]string

func uploadRefreshedTokenToS3(credentials credentialJSON, filename, bucket, zone string) {

	data, err := json.MarshalIndent(credentials, "", " ")
	if err != nil {
		log.Errorf("failed to marshal file %v", err)
		return
	}
	// The session the S3 Uploader will use
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(zone),
	}))
	r := bytes.NewReader(data)

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	// Upload the file to S3.
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
		Body:   r,
	})
	if err != nil {
		log.Errorf("Failed to upload file, %v", err)
		return
	}
	log.Info("New Credential File is uploaded")
}

func GetAccessToken(tpp_url string) (string, error) {
	CREDENTIAL_FILE_NAME := os.Getenv("CREDENTIAL_FILE_NAME")
	S3_BUCKET := os.Getenv("S3_BUCKET")
	ZONE := os.Getenv("ZONE")
	credentials, err := getCredentials(CREDENTIAL_FILE_NAME, S3_BUCKET, ZONE)
	if err != nil {
		log.Error("Failed to get credentials from S3")
		return "", fmt.Errorf("Failed to get access token: %v", err.Error())
	}

	credentialArray := parseCredentialData(credentials)
	var access_token string
	for _, x := range credentialArray {
		if x["url"] == tpp_url {
			expiritation_time := x["access_token_expires"]
			layout := "2006-01-02T15:04:05.000Z"
			t, _ := time.Parse(layout, expiritation_time)
			if CheckIfAccessTokenIsValid(t) {
				access_token = x["access_token"]
				break
			} else {
				new_credentials := GetNewAccessToken(x)
				if new_credentials != nil {
					access_token = (*new_credentials)["access_token"]
					x = *new_credentials
					uploadRefreshedTokenToS3(credentialArray, CREDENTIAL_FILE_NAME, S3_BUCKET, ZONE)
					break
				} else {
					log.Errorf("Failed to get new credentials")
					return "", fmt.Errorf("Failed to refresh and get new credentials from S3")
				}
			}
		}
	}
	return access_token, nil
}

func CheckIfAccessTokenIsValid(acces_token_expiration time.Time) bool {
	return time.Now().Before(acces_token_expiration)
}

func GetNewAccessToken(single_credential_for_tpp map[string]string) *map[string]string {

	auth := endpoint.Authentication{
		User:     single_credential_for_tpp["user"],
		Password: single_credential_for_tpp["password"],
	}

	c, err := tpp.NewConnector(single_credential_for_tpp["url"], "", false, nil)
	if err != nil {
		log.Errorf("Failed to create TPP Connector: %v", err.Error())
		return nil
	}

	resp, err := c.GetRefreshToken(&auth)
	if err != nil {
		log.Errorf("Failed to get refresh token: %v", err.Error())
		return nil
	}
	auth.RefreshToken = resp.Refresh_token

	new_creds, err := c.RefreshAccessToken(&auth)
	if err != nil {
		fmt.Printf("err: %v", err.Error())
		return nil
	}
	single_credential_for_tpp["access_token"] = new_creds.Access_token
	single_credential_for_tpp["refresh_token"] = new_creds.Refresh_token
	single_credential_for_tpp["access_token_expires"] = fmt.Sprintf("%d", new_creds.Expires)
	return &single_credential_for_tpp
}

func parseCredentialData(credentialsData []byte) credentialJSON {
	var data credentialJSON
	err := json.Unmarshal(credentialsData, &data)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

func getCredentials(filename, bucket, zone string) ([]byte, error) {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String(zone),
	}))

	// Create a downloader with the session and default options
	downloader := s3manager.NewDownloader(sess)
	buff := &aws.WriteAtBuffer{}

	// Write the contents of S3 Object to the file
	n, err := downloader.Download(buff, &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(filename),
	})
	if err != nil {
		fmt.Printf("Failed to get credentials: %v", err)
		return []byte{}, fmt.Errorf("failed to get credentials, %v", err)
	}
	fmt.Printf("file downloaded, %d bytes\n", n)
	return buff.Bytes(), nil
}
