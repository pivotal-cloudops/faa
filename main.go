package main

import (
	"errors"
	"log"
	"net/http"
	"os"

	"fmt"
	"strings"

	"github.com/pivotal-cloudops/faa/postfacto"
	"github.com/pivotal-cloudops/faa/slackcommand"
)

func main() {
	var (
		port string
		ok   bool
	)
	port, ok = os.LookupEnv("PORT")
	if !ok {
		port = "8080"
	}

	vToken, ok := os.LookupEnv("SLACK_VERIFICATION_TOKEN")
	if !ok {
		panic(errors.New("must provide SLACK_VERIFICATION_TOKEN"))
	}

	postfactoAPI, ok := os.LookupEnv("POSTFACTO_API")
	if !ok {
		postfactoAPI = "https://retro-api.cfapps.io"
	}

	retroID, ok := os.LookupEnv("POSTFACTO_RETRO_ID")
	if !ok {
		panic(errors.New("must provide POSTFACTO_RETRO_ID"))
	}

	retroPassword, ok := os.LookupEnv("POSTFACTO_RETRO_PASSWORD")
	if !ok {
		panic(errors.New("must provide POSTFACTO_RETRO_PASSWORD"))
	}

	c := &postfacto.RetroClient{
		Host: postfactoAPI,
		ID:   retroID,
		Password: retroPassword,
	}

	server := slackcommand.Server{
		VerificationToken: vToken,
		Delegate: &PostfactoSlackDelegate{
			RetroClient: c,
		},
	}

	http.Handle("/", server)

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

type PostfactoSlackDelegate struct {
	RetroClient *postfacto.RetroClient
}

type Command string

const (
	CommandHappy Command = "happy"
	CommandMeh   Command = "meh"
	CommandSad   Command = "sad"
)

func (d *PostfactoSlackDelegate) Handle(r slackcommand.Command) (string, error) {
	parts := strings.SplitN(r.Text, " ", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("must be in the form of '%s [happy/meh/sad] [message]'", r.Command)
	}

	c := parts[0]
	description := parts[1]

	var category postfacto.Category

	client := d.RetroClient

	switch Command(c) {
	case CommandHappy:
		category = postfacto.CategoryHappy
	case CommandMeh:
		category = postfacto.CategoryMeh
	case CommandSad:
		category = postfacto.CategorySad
	default:
		return "", errors.New("unknown command: must provide one of 'happy', 'meh' or 'sad'")
	}

	retroItem := postfacto.RetroItem{
		Category:    category,
		Description: fmt.Sprintf("%s [%s]", description, r.UserName),
	}

	err := client.Login()
	if err != nil {
		return "", err
	}

	err = client.Add(retroItem)
	if err != nil {
		return "", err
	}

	return "retro item added", nil
}
