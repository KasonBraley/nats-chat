package main

import (
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nats-io/nats.go"
)

func main() {
	if len(os.Getenv("DEBUG")) > 0 {
		f, err := tea.LogToFile("debug.log", "debug")
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = nats.DefaultURL
	}

	nc, err := nats.Connect(natsURL,
		nats.ErrorHandler(func(_ *nats.Conn, s *nats.Subscription, err error) {
			if s != nil {
				log.Printf("Async error in %q/%q: %v", s.Subject, s.Queue, err)
			} else {
				log.Printf("Async error outside subscription: %v", err)
			}
		}))
	if err != nil {
		log.Fatal(err)
	}
	defer nc.Close()

	// EncodedConn can Publish any raw Go type using the registered Encoder
	ec, err := nats.NewEncodedConn(nc, nats.JSON_ENCODER)
	if err != nil {
		log.Fatal(err)
	}
	defer ec.Close()

	// could later pass an entire configuration struct to initialModel
	p := tea.NewProgram(initialModel(ec))

	// listen for all chats
	if _, err := ec.Subscribe("chat", func(c *chat) {
		p.Send(c) // send the message to bubbletea to update the UI
	}); err != nil {
		log.Fatal(err)
	}

	// listen for new users
	if _, err := ec.Subscribe("user", func(u *user) {
		p.Send(u) // send the message to bubbletea to update the UI
	}); err != nil {
		log.Fatal(err)
	}

	// Start TUI
	if err := p.Start(); err != nil {
		log.Fatal(err)
	}
}

type chat struct {
	Name string `json:"name"`
	Msg  string `json:"msg"`
}

type user struct {
	Name     string
	LoggedIn bool
	// Room
}

func (u user) FilterValue() string { return "" }
