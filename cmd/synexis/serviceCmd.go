package synexis

import (
	"errors"
	"fmt"
	"github.com/golang-jwt/jwt/v4"
	"github.com/spf13/cobra"
	"github.com/synxms/synexis/pkg/storage"
	"github.com/synxms/synexis/pkg/utility"
	"github.com/synxms/synexis/src/service"
	"log"
	"os"
)

func generateAPIKey(_ *cobra.Command, _ []string) error {
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()

	// get base url
	baseUrl, err := store.Get("base_url")
	if err != nil {
		log.Fatalln("Failed to get access token:", err)
	}
	authenticationService := service.NewAuthentication(baseUrl)

	// get access token
	accessToken, err := store.Get("access_token")
	if err != nil {
		log.Fatalln("Failed to get access token:", err)
	}

	// retrieve company id
	parser := jwt.NewParser()
	claims := jwt.MapClaims{}
	_, _, err = parser.ParseUnverified(accessToken, claims)
	if err != nil {
		return errors.New("invalid token")
	}
	companyId, ok := claims["companyId"].(string)
	if !ok {
		return errors.New("no company id field in token")
	}

	const apiKeyFormat = "SYX%s-%s-%s-%s"
	prefix := utility.RandomStringUpperCase(3)
	validationLayerOne := utility.RandomString(5)
	validationLayerTwo := utility.RandomString(10)
	result, err := authenticationService.GenerateAPIKeySentinel("SYX"+prefix, validationLayerOne, validationLayerTwo, accessToken)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if result != nil {
		if result.ResponseCode == "00" {
			fmt.Println(fmt.Sprintf(apiKeyFormat, prefix, validationLayerOne, validationLayerTwo, companyId))
		} else {
			fmt.Println("API Key generate failed failed.")
		}
	}
	return nil
}

func uploadDatasetFile(cmd *cobra.Command, args []string) error {
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()
	// get base url
	baseUrl, err := store.Get("base_url")
	if err != nil {
		log.Fatalln("Failed to get base url:", err)
	}
	// get access token
	accessToken, err := store.Get("access_token")
	if err != nil {
		log.Fatalln("Failed to get access token:", err)
	}
	authenticationService := service.NewAuthentication(baseUrl)
	result, err := authenticationService.UploadFileDatasetSentinel(args[0], accessToken)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if result != nil {
		if result.ResponseCode == "00" {
			outputPath, _ := cmd.Flags().GetString("output")
			if outputPath != "" {
				err := os.WriteFile(outputPath, []byte(result.Data.DatasetID), 0644)
				if err != nil {
					log.Fatalln("Failed to write dataset ID to file:", err)
				}
				fmt.Println("Dataset ID saved to", outputPath)
			} else {
				log.Fatalln("Use at the end '-o' to specify output file path")
			}
		} else {
			fmt.Println("Upload dataset failed.")
			fmt.Println("Upload dataset failed, reason : ", result.ResponseMessage)
		}
	}
	return nil
}

func uploadSensoryFile(cmd *cobra.Command, args []string) error {
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()
	// get base url
	baseUrl, err := store.Get("base_url")
	if err != nil {
		log.Fatalln("Failed to get base url:", err)
	}
	// get access token
	accessToken, err := store.Get("access_token")
	if err != nil {
		log.Fatalln("Failed to get access token:", err)
	}
	authenticationService := service.NewAuthentication(baseUrl)
	result, err := authenticationService.UploadFileSensorySentinel(args[0], accessToken)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if result != nil {
		if result.ResponseCode == "00" {
			outputPath, _ := cmd.Flags().GetString("output")
			if outputPath != "" {
				err := os.WriteFile(outputPath, []byte(result.Data.SensoryID), 0644)
				if err != nil {
					log.Fatalln("Failed to write sensory ID to file:", err)
				}
				fmt.Println("Sensory ID saved to", outputPath)
			} else {
				log.Fatalln("Use at the end '-o' to specify output file path")
			}
		} else {
			fmt.Println("Upload sensory failed.")
			fmt.Println("Upload sensory failed, reason : ", result.ResponseMessage)
		}
	}
	return nil
}

func InitializeServiceCmd(serviceCmd *cobra.Command) {
	sentinelCmd := &cobra.Command{
		Use:   "sentinel",
		Short: "Sentinel synexis service command console",
		Long:  `Sentinel synexis service command console`,
	}
	sentinelCmd.AddCommand(&cobra.Command{
		Use:   "apikey",
		Short: "Sentinel API Key generate be careful with this command",
		Long:  `Sentinel API Key generate be careful with this command`,
		RunE:  generateAPIKey,
	})
	datasetCmd := &cobra.Command{
		Use:   "dataset",
		Short: "Sentinel upload dataset for custom training",
		Long:  `Sentinel upload dataset for custom training`,
		Args:  cobra.ExactArgs(1),
		RunE:  uploadDatasetFile,
	}
	datasetCmd.Flags().StringP("output", "o", "", "Path to output file for saving DatasetID")
	sentinelCmd.AddCommand(datasetCmd)
	sensoryCmd := &cobra.Command{
		Use:   "sensory",
		Short: "Sentinel upload sensory configuration for custom training",
		Long:  `Sentinel upload sensory configuration for custom training`,
		Args:  cobra.ExactArgs(1),
		RunE:  uploadSensoryFile,
	}
	sensoryCmd.Flags().StringP("output", "o", "", "Path to output file for saving SensoryID")
	sentinelCmd.AddCommand(sensoryCmd)

	serviceCmd.AddCommand(sentinelCmd)
}
