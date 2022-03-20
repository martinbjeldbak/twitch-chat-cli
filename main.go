package main

import (
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gempir/go-twitch-irc/v3"
	"github.com/mattn/go-sixel"
)

const cdnFmtString = "https://static-cdn.jtvnw.net/emoticons/v2/%v/%v/%v/%v"

func emoteUrl(id string) string {
	return fmt.Sprintf(cdnFmtString, id, "static", "dark", "1.0")
}

func main() {
	client := twitch.NewAnonymousClient()

	client.OnPrivateMessage(func(message twitch.PrivateMessage) {
		msg := lipgloss.NewStyle().
			Inline(true).
			Bold(true).
			Foreground(lipgloss.Color(message.User.Color)).
			Render(message.User.DisplayName)

		msg += fmt.Sprintf(": %v", message.Message)

		for i, emote := range message.Emotes {
			emoteUrl := emoteUrl(emote.ID)

			resp, err := http.Get(emoteUrl)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			emoteImageData, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Fatal(err)
			}

			img, err := png.Decode(strings.NewReader(string(emoteImageData)))
			if err != nil {
				log.Fatal(err)
			}

			var buf strings.Builder
			err = sixel.NewEncoder(&buf).Encode(img)
			if err != nil {
				log.Fatal(err)
			}

			//msg = strings.ReplaceAll(msg, emote.Name, buf.String())
			emoteString := buf.String()
			msg += fmt.Sprintf(" (emote #%v: %v)", i+1, emoteString)
		}

		fmt.Println(msg)
	})

	client.Join("esfandtv")

	fmt.Println("Started...")

	err := client.Connect()
	if err != nil {
		log.Fatal(err)
	}
}
