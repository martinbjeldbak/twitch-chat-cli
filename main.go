package main

import (
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/BourgeoisBear/rasterm"
	"github.com/charmbracelet/lipgloss"
	"github.com/gempir/go-twitch-irc/v3"
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
			// TRY #1 with sixel-go
			// err = sixel.NewEncoder(&buf).Encode(img)
			// if err != nil {
			// 	log.Fatal(err)
			// }

			// TRY #2 with rasterm
			rts := rasterm.Settings{EscapeTmux: false}
			err = rts.ItermWriteImage(&buf, img)
			if err != nil {
				log.Fatal(err)
			}

			msg = strings.ReplaceAll(msg, emote.Name, buf.String())
			// If debugging:
			msg += fmt.Sprintf(" (#%v: %v -> %v)", i+1, emote.Name, buf.String())
		}

		fmt.Println(msg)
	})

	client.Join("esfandtv")

	sixelCapable, err := rasterm.IsSixelCapable()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Started... (iTerm: %v, sixel: %v, tmux: %v, kitty: %v)\n", rasterm.IsTermItermWez(), sixelCapable, rasterm.IsTmuxScreen(), rasterm.IsTermKitty())

	err = client.Connect()
	if err != nil {
		log.Fatal(err)
	}
}
