package main

import (
	"fmt"
	"log"
)

var Verbose bool = true

func Log(isVerbose bool, message string, tabIndex int, params ...interface{}) {
	if isVerbose && !Verbose {
		return
	}
	ret := ""
	for i := 0; i < tabIndex; i++ {
		ret += "\t"
	}
	ret += message
	log.Printf(ret, params...)
}

func LogFatal(message string, params ...interface{}) {
	log.Fatalf(message, params...)
}

func main() {
	c := ConfigOptions{
		Aws: AwsOptions{
			AccessKeyID: "",
			AccessKey:   "",
			Zone:        "",
			Bucket:      "",
		},
		Snowflake: []SnowflakeOptions{
			{
				Role:      "",
				Account:   "",
				Username:  "",
				Password:  "",
				Warehouse: "",
				Database:  "",
				Schema:    "",
			},
		},
		Venafi: []VenafiOptions{
			{
				AccessToken:        "",
				AccessTokenExpires: "",
				RefreshToken:       "",
				Url:                "",
			},
		},
	}
	config, _, s3Client, lambdaClient, iamClient, gatewayClient, stsClient := bootstrapOperation(0)
	accountID, err := GetCallerIdentity(stsClient)
	if err != nil {
		log.Fatal("Failed to get account id")
	}
	SetConfig(c)
	PrintStatus(GetStatus(0, config, s3Client, lambdaClient, iamClient, gatewayClient, accountID))

	Install(config, s3Client, lambdaClient, iamClient, gatewayClient, accountID)
	fmt.Print("over")
}

// type AWSConfig struct {
// 	Zone       string
// 	Bucket     string
// 	AccesKeyID string
// 	AccesKey   string
// }

// var CLI struct {
// 	Install struct {
// 		Verbose bool
// 		// Manual bool `arg name:"manuel" help:"PLease config the tool first." type:"path"`
// 	} `cmd help:"Manual install."`
// 	Status struct {
// 	} `cmd help:"Status of venafi snowflake connector"`
// 	Config struct {
// 		Manual      bool
// 		AccessKeyID string `arg optional name:"aws_access_key_id" help:"AWS Access Key."`
// 		AccessKey   string `arg optional name:"aws_access_key" help:"AWS Access Key."`
// 		Zone        string `arg optional name:"aws_zone" help:"AWS Access Key."`
// 		Bucket      string `arg optional name:"s3_bucket" help:"AWS Access Key."`
// 		Role        string `arg optional name:"snowflake_role" help:"AWS Access Key."`
// 		Account     string `arg optional name:"snowflake_account" help:"AWS Access Key."`
// 		Username    string `arg optional name:"snowflake_username" help:"AWS Access Key."`
// 		Password    string `arg optional name:"snowflake_password" help:"AWS Access Key."`
// 		Warehouse   string `arg optional name:"snowflake_warehouse" help:"AWS Access Key."`
// 		Database    string `arg optional name:"snowflake_database" help:"AWS Access Key."`
// 		Schema      string `arg optional name:"snowflake_schema" help:"AWS Access Key."`
// 	} `cmd help:"Create config file for venafi connector"`
// }

// func checkIfConfigFileExists() error {
// 	if _, err := os.Stat("./config.yml"); os.IsNotExist(err) {
// 		return err
// 	}
// 	return nil
// }

// func createConfigFromManualInput() cli_tool.Config {
// 	var accessKeyID string
// 	var accessKey string
// 	var zone string
// 	var s3Bucket string
// 	var account string
// 	var role string
// 	var warehouse string
// 	var database string
// 	var schema string
// 	var username string
// 	var password string

// 	fmt.Println("Enter AWS Access Key Id ")
// 	fmt.Scanln(&accessKeyID)

// 	fmt.Println("Enter AWS Access Key: ")
// 	fmt.Scanln(&accessKey)

// 	fmt.Println("Enter AWS Zone config: ")
// 	fmt.Scanln(&zone)

// 	fmt.Println("Enter AWS S3 Bucket name: ")
// 	fmt.Scanln(&s3Bucket)

// 	fmt.Println("Enter Snowflake Account: ")
// 	fmt.Scanln(&account)

// 	fmt.Println("Enter Snowflake Role: ")
// 	fmt.Scanln(&role)

// 	fmt.Println("Enter Snowflakee warehouse: ")
// 	fmt.Scanln(&warehouse)

// 	fmt.Println("Enter Snowflake database: ")
// 	fmt.Scanln(&database)

// 	fmt.Println("Enter Snowflake schema: ")
// 	fmt.Scanln(&schema)

// 	fmt.Println("Enter Snowflake user: ")
// 	fmt.Scanln(&username)

// 	fmt.Println("Enter snowflake password: ")
// 	fmt.Scanln(&password)

// 	return cli_tool.Config{
// 		cli_tool.AWSConfig{AccessKeyID: accessKeyID, AccessKey: accessKey, Zone: zone, S3Bucket: s3Bucket},
// 		cli_tool.SnowflakeConfig{Role: role, Account: account, Warehouse: warehouse, Database: database, Schema: schema, Username: username},
// 	}
// }

// func createConfigFromArguments(ctx *kong.Context) cli_tool.Config {
// 	args := ctx.Args

// 	return cli_tool.Config{
// 		cli_tool.AWSConfig{
// 			AccessKeyID: args[1],
// 			AccessKey:   args[2],
// 			Zone:        args[3],
// 			S3Bucket:    args[4],
// 		},
// 		cli_tool.SnowflakeConfig{
// 			Role:      args[5],
// 			Account:   args[6],
// 			Username:  args[7],
// 			Password:  args[8],
// 			Warehouse: args[9],
// 			Database:  args[10],
// 			Schema:    args[11],
// 		},
// 	}
// }

// func createAWSConfigFromArguments(args ctx.Args) cli_tool.AWSConfig {

// }

// func main() {

// ctx := kong.Parse(&CLI)
// if strings.Contains(ctx.Command(), "config") && len(ctx.Args) < 12 {
// 	manualFlag := ctx.FlagValue(ctx.Flags()[1])
// 	if !manualFlag.(bool) {
// 		print("Please add config arguments or choose manual config option. Description about config arguments is available by command snowflake-connector config --help")
// 		return
// 	} else {
// 		config := createConfigFromManualInput()
// 		cli_tool.CreateConfigFile(config)
// 		RunConfigServerlessAWSCredentials(config.AWS.AccessKeyID, config.AWS.AccessKey)
// 		print("Finished create configuration for snowflake connector. Run ./snowflake-connector install")
// 		return
// 	}
// }

// switch ctx.Command() {
// case "install":
// 	print("Check if configuration exists...\n")
// 	err := checkIfConfigFileExists()
// 	if err != nil {
// 		print("Please run `snowflake-connector config` to configurate the install before run cli install\n")
// 		return
// 	}
// 	print("Configuration exists, start install...\n")
// 	print("Install Venafi Snowflake Functions\n")
// 	RunServerlessDeployment()
// 	print("Finished install of Venafi Snowflake Functions\n")
// case "config <aws_access_key_id> <aws_access_key> <aws_zone> <s3_bucket> <snowflake_role> <snowflake_account> <snowflake_username> <snowflake_password> <snowflake_warehouse> <snowflake_database> <snowflake_schema>":

// 	config := createConfigFromArguments(ctx)
// 	cli_tool.CreateConfigFile(config)
// 	RunConfigServerlessAWSCredentials(config.AWS.AccessKeyID, config.AWS.AccessKey)
// 	print("Finished create configuration for snowflake connector. Run ./snowflake-connector install")

// case "status":
// 	print("status ok")
// default:
// 	panic(ctx.Command())
// }
//}
