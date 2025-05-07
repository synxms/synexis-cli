package synexis

import (
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/synexism/synexis/pkg/storage"
	"github.com/synexism/synexis/src/service"
	"log"
)

func setAccessToken(_ *cobra.Command, args []string) error {
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()
	if err := store.Set("access_token", args[0]); err != nil {
		log.Fatalln("Failed to store access token:", err)
	}
	fmt.Println("Access token saved.")
	return nil
}

func setRefreshToken(_ *cobra.Command, args []string) error {
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()
	if err := store.Set("refresh_token", args[0]); err != nil {
		log.Fatalln("Failed to store refresh token:", err)
	}
	fmt.Println("Refresh token saved.")
	return nil
}

func getAccessToken(_ *cobra.Command, _ []string) error {
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()
	result, err := store.Get("access_token")
	if err != nil {
		log.Fatalln("Failed to store access token:", err)
	}
	fmt.Println(result)
	return nil
}

func getRefreshToken(_ *cobra.Command, _ []string) error {
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()
	result, err := store.Get("refresh_token")
	if err != nil {
		log.Fatalln("Failed to store access token:", err)
	}
	fmt.Println(result)
	return nil
}

func refreshToken(_ *cobra.Command, _ []string) error {
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()
	rt, err := store.Get("refresh_token")
	if err != nil {
		log.Fatalln("Failed to store access token:", err)
	}
	authenticationService := service.NewAuthentication()
	result, err := authenticationService.GenerateAccessAndRefreshToken(rt)
	if err != nil {
		log.Fatalln(err.Error())
	}
	if result != nil {
		if result.ResponseCode == "00" {
			if err := store.Set("refresh_token", result.Refresh); err != nil {
				log.Fatalln("Failed to store refresh token:", err)
			}
			if err := store.Set("access_token", result.Access); err != nil {
				log.Fatalln("Failed to store access token:", err)
			}
			fmt.Println("Renewed Refresh token saved.")
			fmt.Println("Renewed Access token saved.")
		} else {
			fmt.Println("Refresh Token failed.")
		}
	}
	return nil
}

func checkRefreshToken(_ *cobra.Command, _ []string) error {
	store := storage.NewStorage()
	if err := store.Init(); err != nil {
		log.Fatalln("Failed to init storage:", err)
	}
	defer store.Close()
	rt, err := store.Get("refresh_token")
	if err != nil {
		log.Fatalln("Failed to store refresh token:", err)
	}
	at, err := store.Get("access_token")
	if err != nil {
		log.Fatalln("Failed to store access token:", err)
	}
	authenticationService := service.NewAuthentication()
	refreshTokenRemaining, refreshTokenExpiredAt, err := authenticationService.IsExpired(rt)
	if err != nil {
		if errors.Is(err, errors.New("token is expired")) {
			fmt.Println("Refresh token expired")
		}
		if !errors.Is(err, errors.New("token is expired")) {
			fmt.Println("Refresh token checking error")
		}
	}
	if refreshTokenRemaining != nil && refreshTokenExpiredAt != nil {
		fmt.Println("Refresh token remaining: " + *refreshTokenRemaining)
		fmt.Println("Refresh token expired at: " + *refreshTokenExpiredAt)
	}
	accessTokenRemaining, accessTokenExpiredAt, err := authenticationService.IsExpired(at)
	if err != nil {
		if errors.Is(err, errors.New("token is expired")) {
			fmt.Println("Access token expired")
		}
		if !errors.Is(err, errors.New("token is expired")) {
			fmt.Println("Access token checking error")
		}
	}
	if accessTokenRemaining != nil && accessTokenExpiredAt != nil {
		fmt.Println("Access token remaining: " + *accessTokenRemaining)
		fmt.Println("Access token expired at: " + *accessTokenExpiredAt)
	}
	return nil
}

func InitializeTokenCmd(tokenCmd *cobra.Command) {
	tokenSetCmd := &cobra.Command{
		Use:   "set",
		Short: "Setter commands",
		Long:  `Setter commands`,
	}
	tokenGetCmd := &cobra.Command{
		Use:   "get",
		Short: "Getter commands",
		Long:  `Getter commands`,
	}
	tokenSetCmd.AddCommand(&cobra.Command{
		Use:   "at [token]",
		Short: "Set new access token to local synexis command line tool",
		Long:  `Set new access token to local synexis command line tool`,
		Args:  cobra.ExactArgs(1),
		RunE:  setAccessToken,
	})
	tokenSetCmd.AddCommand(&cobra.Command{
		Use:   "rt [token]",
		Short: "Set new refresh token to local synexis command line tool",
		Long:  `Set new refresh token to local synexis command line tool`,
		Args:  cobra.ExactArgs(1),
		RunE:  setRefreshToken,
	})
	tokenGetCmd.AddCommand(&cobra.Command{
		Use:   "at",
		Short: "Get existing access token to local synexis command line tool",
		Long:  `Get existing access token to local synexis command line tool`,
		RunE:  getAccessToken,
	})
	tokenGetCmd.AddCommand(&cobra.Command{
		Use:   "rt",
		Short: "Get existing refresh token to local synexis command line tool",
		Long:  `Get existing refresh token to local synexis command line tool`,
		RunE:  getRefreshToken,
	})
	tokenCmd.AddCommand(tokenSetCmd)
	tokenCmd.AddCommand(tokenGetCmd)
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "refresh",
		Short: "Refresh access token and refresh token",
		Long:  `Refresh access token and refresh token`,
		RunE:  refreshToken,
	})
	tokenCmd.AddCommand(&cobra.Command{
		Use:   "check",
		Short: "Refresh access token and Refresh token expired check",
		Long:  `Refresh access token and Refresh token expired check`,
		RunE:  checkRefreshToken,
	})
}
