package queue

import (
	"context"
	"encoding/json"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/gobwas/ws"
	"github.com/gorilla/securecookie"
)

type QueueEndpoints struct {
	ph    *PlaylistHub
	queue *Queue
	es    *EventStream
	sc    *securecookie.SecureCookie
}

func NewQueueEndpoints(ph *PlaylistHub, queue *Queue, es *EventStream, sc *securecookie.SecureCookie) QueueEndpoints {
	return QueueEndpoints{
		ph:    ph,
		queue: queue,
		es:    es,
		sc:    sc,
	}
}
func (endpoints *QueueEndpoints) protect(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ctx := r.Context()

		cookie, err := r.Cookie("session")
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		value := make(map[string]string)
		err = endpoints.sc.Decode("session", cookie.Value, &value)
		if err != nil {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		ctx = context.WithValue(ctx, "username", value["username"])

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (endpoints *QueueEndpoints) playlist(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	playlist := endpoints.ph.GetPlaylist(username)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(playlist)
}

func (endpoints *QueueEndpoints) stream(w http.ResponseWriter, r *http.Request) {
	conn, _, _, err := ws.UpgradeHTTP(r, w)
	if err != nil {
		log.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	username := r.Context().Value("username").(string)
	go endpoints.queue.ProcessConnection(
		username,
		conn,
	)
}

func (endpoints *QueueEndpoints) skip(w http.ResponseWriter, r *http.Request) {
	username := r.Context().Value("username").(string)
	pl := endpoints.ph.GetPlaylist(username)
	if len(pl) > 0 {
		err := endpoints.es.writeMessage(r.Context(), Event{
			Type: "RemoveVideo",
			Payload: RemoveVideoPayload{
				Username: username,
				VideoID:  pl[0].VideoID,
			},
		})
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Println(err)
			return
		}
	}
	w.WriteHeader(http.StatusNoContent)
}

func (endpoints *QueueEndpoints) GetRouter() *chi.Mux {
	r := chi.NewRouter()
	r.Use(endpoints.protect)
	r.Get("/", endpoints.playlist)
	r.Get("/stream", endpoints.stream)
	r.Post("/skip", endpoints.skip)
	return r
}
