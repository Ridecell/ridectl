package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sso"
	"github.com/aws/aws-sdk-go-v2/service/sso/types"
	"github.com/aws/aws-sdk-go-v2/service/ssooidc"
	"github.com/pterm/pterm"
	"gopkg.in/ini.v1"
)

const (
	AWSRegion  = "us-west-2"
	ClientName = "ridectl-aws-sso"
)

type SSOCache struct {
	AccessToken string `json:"accessToken"`
	ExpiresAt   string `json:"expiresAt"`
	Region      string `json:"region"`
	StartURL    string `json:"startUrl"`
}

func isoTimeNow() time.Time {
	return time.Now().UTC()
}

func loadSSOCache(filePath string) (*SSOCache, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var cache SSOCache
	err = json.Unmarshal(data, &cache)
	return &cache, err
}

func openBrowser(url string) {
	var err error

	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		pterm.Error.Printf("Failed to open browser: %v\n", err)
	}
}

func getSSOCachedAccessToken(awsSSOCachePath string) (string, error) {
	cacheFileName := awsSSOCachePath + "/" + ClientName + ".json"

	cache, err := loadSSOCache(cacheFileName)
	if err == nil && cache != nil {
		// Validate if token is not expired
		if isoTimeNow().Before(parseTimestamp(cache.ExpiresAt)) {
			return cache.AccessToken, nil
		}
	}
	return "", fmt.Errorf("cached SSO login expired or invalid")
}

func parseTimestamp(value string) time.Time {
	t, _ := time.Parse(time.RFC3339, value)
	return t
}

func isCredentialValid(awsCredentialPath, roleName string) bool {
	cfg, err := ini.LooseLoad(awsCredentialPath)
	if err == nil {
		// Read profile section named as roleName
		section := cfg.Section(roleName)
		value, err := section.GetKey("expiration")
		if err == nil {
			expiryTime, err := value.Int64()
			if err == nil {
				// Get the current time in milliseconds
				currentTime := time.Now().UnixNano() / int64(time.Millisecond)
				if expiryTime > currentTime {
					return true
				}
			}
		}
	}
	return false
}

func LoadAWSAccountInfo(ridectlConfigFile string) (string, string) {

	cfg, err := ini.LooseLoad(ridectlConfigFile)

	if err == nil {
		// Read profile section named as aws
		section := cfg.Section("aws")
		if section.HasKey("start_url") && section.HasKey("account_id") {
			startUrl, _ := section.GetKey("start_url")
			accountId, _ := section.GetKey("account_id")
			return startUrl.String(), accountId.String()
		}
	}
	return "", ""
}

func UpdateAWSAccountInfo(ridectlConfigFile, startUrl, accountId string) error {

	cfg, err := ini.LooseLoad(ridectlConfigFile)
	if err != nil {
		cfg = ini.Empty()
	}
	// Read profile section named as aws
	section := cfg.Section("aws")

	// Update the keys
	section.Key("start_url").SetValue(startUrl)
	section.Key("account_id").SetValue(accountId)

	// Save the updated configuration back to the file
	return cfg.SaveTo(ridectlConfigFile)
}

func updateAWSCredentials(awsCredentialPath string, roleName string, newCredential *types.RoleCredentials) error {

	cfg, err := ini.LooseLoad(awsCredentialPath)
	if err != nil {
		return err
	}
	// Read profile section named as roleName
	section := cfg.Section(roleName)

	// Update the keys
	section.Key("aws_access_key_id").SetValue(*newCredential.AccessKeyId)
	section.Key("aws_secret_access_key").SetValue(*newCredential.SecretAccessKey)
	section.Key("aws_session_token").SetValue(*newCredential.SessionToken)
	section.Key("expiration").SetValue(fmt.Sprintf("%v", newCredential.Expiration))

	// Save the updated configuration back to the file
	return cfg.SaveTo(awsCredentialPath)
}

func renewAccessToken(startUrl, awsSSOCachePath string) (string, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(AWSRegion))
	if err != nil {
		return "", fmt.Errorf("unable to load SDK config: %v", err)
	}

	ssooidcClient := ssooidc.NewFromConfig(cfg)
	registerClientResponse, err := ssooidcClient.RegisterClient(context.TODO(), &ssooidc.RegisterClientInput{
		ClientName: aws.String(ClientName),
		ClientType: aws.String("public"),
	})
	if err != nil {
		return "", fmt.Errorf("error registering client: %v", err)
	}

	startAuthorizationResponse, err := ssooidcClient.StartDeviceAuthorization(context.TODO(), &ssooidc.StartDeviceAuthorizationInput{
		ClientId:     registerClientResponse.ClientId,
		ClientSecret: registerClientResponse.ClientSecret,
		StartUrl:     aws.String(startUrl),
	})
	if err != nil {
		return "", fmt.Errorf("error starting device authorization: %v", err)
	}

	//fmt.Printf("Please open %s to complete authorization\n", *startAuthorizationResponse.VerificationUriComplete)
	fmt.Println("Attempting to automatically open the SSO authorization page in your default browser.")
	fmt.Println("\nIf the browser does not open or you wish to use a different device to authorize this request, open the following URL:")
	fmt.Printf("%s\n", *startAuthorizationResponse.VerificationUriComplete)

	openBrowser(*startAuthorizationResponse.VerificationUriComplete)

	// Wait for device authorization
	var tokenResponse *ssooidc.CreateTokenOutput
	for {
		tokenResponse, err = ssooidcClient.CreateToken(context.TODO(), &ssooidc.CreateTokenInput{
			ClientId:     registerClientResponse.ClientId,
			ClientSecret: registerClientResponse.ClientSecret,
			GrantType:    aws.String("urn:ietf:params:oauth:grant-type:device_code"),
			DeviceCode:   startAuthorizationResponse.DeviceCode,
		})
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	expirationDate := isoTimeNow().Add(time.Duration(tokenResponse.ExpiresIn) * time.Second)
	accessToken := *tokenResponse.AccessToken

	// Cache the token
	cache := SSOCache{
		AccessToken: accessToken,
		ExpiresAt:   expirationDate.Format(time.RFC3339),
		Region:      AWSRegion,
		StartURL:    startUrl,
	}
	cacheData, _ := json.Marshal(cache)

	return accessToken, os.WriteFile(filepath.Join(awsSSOCachePath, ClientName+".json"), cacheData, 0640)
}

func fetchRoleCredentials(accessToken, roleName, accountId string) (*types.RoleCredentials, error) {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion(AWSRegion))
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config: %v", err)
	}

	ssoClient := sso.NewFromConfig(cfg)
	roleCredentials, err := ssoClient.GetRoleCredentials(context.TODO(), &sso.GetRoleCredentialsInput{
		RoleName:    aws.String(roleName),
		AccountId:   aws.String(accountId),
		AccessToken: aws.String(accessToken),
	})
	if err != nil {
		return nil, fmt.Errorf("error fetching role credentials: %v", err)
	}

	return roleCredentials.RoleCredentials, nil
}

func RetriveAWSSSOCredsPath(ridectlHomeDir, startUrl, accountId, roleName string) string {

	// Check if existing credentials are valid
	awsCredentialPath := ridectlHomeDir + "/credentials"
	if isCredentialValid(awsCredentialPath, roleName) {
		return awsCredentialPath
	}

	// Unset AWS_PROFILE env var if set.
	_ = os.Setenv("AWS_PROFILE", "")

	accessToken, err := getSSOCachedAccessToken(ridectlHomeDir)
	if err != nil {
		pterm.Warning.Printf("%v, renewing access token.\n", err)
		accessToken, err = renewAccessToken(startUrl, ridectlHomeDir)
		if err != nil {
			pterm.Error.Printf("error renewing access token: %v", err)
			os.Exit(1)
		}
	}

	credentials, err := fetchRoleCredentials(accessToken, roleName, accountId)
	if err != nil {
		pterm.Error.Printf("error fetching account credentials: %v", err)
		os.Exit(1)
	}

	if err = updateAWSCredentials(awsCredentialPath, roleName, credentials); err != nil {
		pterm.Error.Printf("error updating AWS credentials: %v", err)
		os.Exit(1)
	}
	return awsCredentialPath
}
