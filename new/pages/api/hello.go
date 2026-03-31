package api

import (
	"encoding/json"
	"net/http"

	"github.com/salmanfaris22/nexgo/pkg/router"
)

func init() { router.RegisterAPI("/api/hello", Hello) }

func Hello(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Hello from NexGo! 👋",
		"method":  r.Method,
	})
}
