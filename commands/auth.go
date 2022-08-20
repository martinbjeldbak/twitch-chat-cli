package commands

import (
	"bufio"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(authCmd)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Twitch",
	Long:  "Authenticate with Twitch using OIDC implicit grant flow via local server",
	Run: func(cmd *cobra.Command, args []string) {
		clientId := "j1i10vfts1iy5p43v8pipr6brg2u3q"
		rand.Seed(time.Now().UnixNano())
		nonce := rand.Int()
		state := rand.Int()
		u, err := url.Parse("https://id.twitch.tv/oauth2/authorize")
		if err != nil {
			log.Fatal(err)
		}

		localCallbackPort := "8090"

		q := u.Query()
		q.Set("response_type", "token id_token")
		q.Set("client_id", clientId)
		q.Set("redirect_uri", fmt.Sprintf("http://localhost:%v/callback", localCallbackPort))
		q.Set("scope", "openid chat:read chat:edit whispers:read whispers:edit")
		q.Set("nonce", fmt.Sprint(nonce))
		q.Set("state", fmt.Sprint(state))
		u.RawQuery = q.Encode()

		go func() {
			http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
				http.ServeFile(w, r, "./static/callback.html")
			})

			if err := http.ListenAndServe(fmt.Sprintf(":%v", localCallbackPort), nil); err != nil {
				log.Fatal(err)
			}
		}()

		fmt.Println(u.String())
		fmt.Printf("\nClick on the above link to authenticate with Twitch.tv. This happens safely client side.\n")
		fmt.Print("Press Ctrl+C when access token copied from above URL")

		// Pause for user input
		input := bufio.NewScanner(os.Stdin)
		input.Scan()
	},
}
