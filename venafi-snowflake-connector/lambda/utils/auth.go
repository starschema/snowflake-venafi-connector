package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/Venafi/vcert/v4/pkg/endpoint"
	"github.com/Venafi/vcert/v4/pkg/venafi/tpp"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

const CREDENTIAL_FILE_NAME = "credentials.json"

func uploadRefreshedTokenToS3(credentials []map[string]string) {
	file, err := json.MarshalIndent(credentials, "", " ")
	if err != nil {
		fmt.Printf("failed to marshal file %q, %v", file, err)
		return
	}
	err = ioutil.WriteFile(CREDENTIAL_FILE_NAME, file, 0644)
	if err != nil {
		fmt.Printf("failed to write file %q, %v", file, err)
		return
	}
	// The session the S3 Uploader will use
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"),
	}))

	// Create an uploader with the session and default options
	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(CREDENTIAL_FILE_NAME)
	if err != nil {
		fmt.Errorf("failed to open file %q, %v", CREDENTIAL_FILE_NAME, err)
		return
	}

	// Upload the file to S3.
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("venafi-credentials"),
		Key:    aws.String(CREDENTIAL_FILE_NAME),
		Body:   f,
	})
	if err != nil {
		fmt.Printf("failed to upload file, %v", err)
		return
	}
	fmt.Printf("file uploaded")
}

func GetAccessToken(tpp_url string) string {
	if !fileExists(CREDENTIAL_FILE_NAME) {
		getCredentialFile()
	}

	credentialArray := parseJson()
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
				fmt.Printf("creds: %v", new_credentials)
				if new_credentials != nil {
					access_token = (*new_credentials)["access_token"]
					x = *new_credentials
					uploadRefreshedTokenToS3(credentialArray)
					break
				}
			}
		}
	}
	fmt.Printf("access_token: %s", access_token)
	return access_token
}

func CheckIfAccessTokenIsValid(acces_token_expiration time.Time) bool {
	return time.Now().Before(acces_token_expiration)
}

func GetNewAccessToken(credentials map[string]string) *map[string]string {

	auth := endpoint.Authentication{
		User:     "partneradmin",
		Password: "Password123!",
	}

	c, err := tpp.NewConnector(credentials["url"], "", false, nil)
	if err != nil {
		fmt.Printf("err: %v", err.Error())
		return nil
	}

	resp, err := c.GetRefreshToken(&auth)
	if err != nil {
		panic(err)
	}
	auth.RefreshToken = resp.Refresh_token

	new_creds, err := c.RefreshAccessToken(&auth)
	if err != nil {
		fmt.Printf("err: %v", err.Error())
		return nil
	}
	credentials["access_token"] = new_creds.Access_token
	credentials["refresh_token"] = new_creds.Refresh_token
	credentials["access_token_expires"] = fmt.Sprintf("%d", new_creds.Expires)
	return &credentials
}

type credentalArray []map[string]string

func parseJson() credentalArray {
	var data credentalArray
	file, err := ioutil.ReadFile(CREDENTIAL_FILE_NAME)
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(file, &data)
	if err != nil {
		log.Fatal(err)
	}
	return data
}

// fileExists checks if a file exists and is not a directory before we
// try using it to prevent further errors.
func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func getCredentialFile() error {
	sess := session.Must(session.NewSession(&aws.Config{
		Region: aws.String("eu-west-1"),
	}))

	// Create a downloader with the session and default options
	downloader := s3manager.NewDownloader(sess)

	// Create a file to write the S3 Object contents to.
	f, err := os.Create(CREDENTIAL_FILE_NAME)
	if err != nil {
		return fmt.Errorf("failed to create file %q, %v", CREDENTIAL_FILE_NAME, err)
	}

	// Write the contents of S3 Object to the file
	n, err := downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String("venafi-credentials"),
		Key:    aws.String(CREDENTIAL_FILE_NAME),
	})
	if err != nil {
		fmt.Printf("error: %v", err)
		return fmt.Errorf("failed to download file, %v", err)
	}
	fmt.Printf("file downloaded, %d bytes\n", n)
	return nil

}

// The session the S3 Downloader will use
