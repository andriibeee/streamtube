package main

import (
	"flag"
	"net/http"
	"os"
	"os/signal"
	"streamtube/app/internal/auth"
	"streamtube/app/internal/queue"
	"strings"
	"syscall"

	"github.com/go-chi/chi"
	"github.com/gorilla/securecookie"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/twitch"
)

func main() {
	var kafkaBrokers string
	var port string
	var clientID string
	var clientSecret string
	var hashKey string
	var blockKey string

	flag.StringVar(&kafkaBrokers, "kafkaBrokers", "127.0.0.1:19092", "kafka cluster addrs (split by comma)")
	flag.StringVar(&port, "port", "3000", "http port")
	flag.StringVar(&clientID, "clientID", "", "twitch oauth client id")
	flag.StringVar(&clientSecret, "clientSecret", "", "twitch oauth secret")
	flag.StringVar(&hashKey, "hashKey", "7qjvrRtsIFschSOMGYR119oYE9Ho034O", "hash key")
	flag.StringVar(&blockKey, "blockKey", "NSX0nnAr1qSOVHFb", "block key")

	flag.Parse()

	if kafkaBrokers == "" {
		panic("kafkaBrokers should be set")
	}

	brokers := strings.Split(kafkaBrokers, ",")
	for i, b := range brokers {
		brokers[i] = strings.TrimSpace(b)
	}

	s := securecookie.New([]byte(hashKey), []byte(blockKey))

	oauth2Config := &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Scopes:       []string{"user:read:email"},
		Endpoint:     twitch.Endpoint,
		RedirectURL:  "http://localhost:3000/auth/callback",
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	ch := queue.NewConnectionsHub()
	pl := queue.NewPlaylistHub()
	es := queue.NewEventStream(brokers, &ch, &pl)
	qe := queue.NewQueue(&ch, &pl, &es)

	go es.Run(c)

	qendp := queue.NewQueueEndpoints(&pl, &qe, &es, s)
	ae := auth.NewAuthEndpoints(s, oauth2Config)

	r := chi.NewRouter()
	r.Mount("/playlist", qendp.GetRouter())
	r.Mount("/auth", ae.GetRouter())

	http.ListenAndServe(":"+port, r)
}
