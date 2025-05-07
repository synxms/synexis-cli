package synexis

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/synexism/synexis/src/service"
	"log"
)

func synexisAuthenticate(_ *cobra.Command, _ []string) error {
	authenticationService := service.NewAuthentication()
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
	InitializeTokenCmd(serviceCmd)
	rootCmd.AddCommand(authenticateCmd)
	rootCmd.AddCommand(tokenCmd)
	rootCmd.AddCommand(serviceCmd)
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		panic(err)
	}
}
