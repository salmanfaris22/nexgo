package main

import (
	"context"
	"log"
	"os"
	"os/signal"

	"github.com/salmanfaris22/nexgo/pkg/config"
	"github.com/salmanfaris22/nexgo/pkg/server"
)

func main() {
	cfg, err := config.Load(".")
	if err != nil {
		log.Fatal(err)
	}

	srv, err := server.New(cfg)
	if err != nil {
		log.Fatal(err)
	}

	// Register data loaders (like getServerSideProps in Next.js)
	// srv.RegisterDataLoader("/blog/[slug]", func(req *http.Request, params map[string]string) (map[string]interface{}, error) {
	//     return map[string]interface{}{"slug": params["slug"]}, nil
	// })

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := srv.Start(ctx); err != nil {
		log.Fatal(err)
	}
}
