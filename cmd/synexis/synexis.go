package synexis

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/synxms/synexis/pkg/storage"
	"github.com/synxms/synexis/pkg/utility"
	"github.com/synxms/synexis/src/service"
	"log"
)

func synexisAuthenticate(_ *cobra.Command, _ []string) error {
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
	result, err := authenticationService.GenerateLoginWithGoogle()
	if err != nil {
		log.Fatalln(err.Error())
	}
	if result != nil {
		if result.ResponseCode == "00" {
			err := authenticationService.OpenDefaultBrowser(result.RedirectURL)
			if err != nil {
				log.Fatalln(err.Error())
			}
		} else {
			fmt.Println("Authentication failed.")
		}
	}
	return nil
}

func synexisServerBaseURL(_ *cobra.Command, args []string) error {
	if !utility.IsValidURL(args[0]) {
		log.Fatalln("please provide valid base url before continue")
	}
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()
	if err := store.Set("base_url", args[0]); err != nil {
		log.Fatalln("Failed to store server base url:", err)
	}
	return nil
}

var (
	rootCmd = &cobra.Command{
		Use:   "synexis",
		Short: "Authentication tools for synexis",
		Long:  `Authentication tools for synexis`,
	}
	authenticateCmd = &cobra.Command{
		Use:   "authenticate",
		Short: "Authentication to register or login into synexis account",
		Long:  `Authentication to register or login into synexis account`,
		RunE:  synexisAuthenticate,
	}
	serverCmd = &cobra.Command{
		Use:   "server-base-url",
		Short: "First command should be executed to start using this cli tool",
		Long:  `First command should be executed to start using this cli tool`,
		Args:  cobra.ExactArgs(1),
		RunE:  synexisServerBaseURL,
	}
	tokenCmd = &cobra.Command{
		Use:   "token",
		Short: "Token management after authentication",
		Long:  `Token management after authentication`,
	}
	serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "Sub command for holds synexis services",
		Long:  `Sub command for holds synexis services`,
	}
)

func Initialize() {
	InitializeTokenCmd(tokenCmd)
	InitializeServiceCmd(serviceCmd)
	rootCmd.AddCommand(authenticateCmd)
	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(serviceCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
