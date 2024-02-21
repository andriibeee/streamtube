package queue

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/mitchellh/mapstructure"
	"github.com/segmentio/kafka-go"
)

var ErrUnknownEventType = errors.New("unknown type")
var ErrInvalidPayload = errors.New("invalid payload")

type Event struct {
	Type    string      `json:"type"`
	Payload interface{} `json:"payload"`
}

type AddNewVideoPayload struct {
	Username string `mapstructure:"username"`
	AddedBy  string `mapstructure:"addedBy"`
	VideoID  string `mapstructure:"videoID"`
}

type RemoveVideoPayload struct {
	Username string `mapstructure:"username"`
	VideoID  string `mapstructure:"videoID"`
}

type EventStream struct {
	hub      *ConnectionsHub
	reader   *kafka.Reader
	playlist *PlaylistHub
	writer   *kafka.Writer
}

func NewEventStream(brokers []string, hub *ConnectionsHub, playlist *PlaylistHub) EventStream {
	return EventStream{
		hub:      hub,
		playlist: playlist,
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:  brokers,
			Topic:    "events",
			MaxBytes: 10e6,
		}),
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        "events",
			RequiredAcks: kafka.RequireNone,
			Async:        true,
		},
	}
}

func (exc EventStream) OnEvent(event Event) error {
	if event.Type == "AddNewVideo" {
		var payload AddNewVideoPayload
		err := mapstructure.Decode(event.Payload, &payload)
		if err != nil {
			return ErrInvalidPayload
		}
		log.Println("message received")

		err = exc.hub.SendMessage(payload.Username, MessageEnvelope{
			Type: event.Type,
			Payload: AddNewVideoMessage{
				AddedBy: payload.AddedBy,
				VideoID: payload.VideoID,
			},
		})

		if err != nil && errors.Is(err, ErrNoConnectionFound) {
			// no-op
			// the connected user might be on another instance
		} else if err != nil {
			return err
		}

		log.Println("message sent to socket")

		err = exc.playlist.AddVideo(payload.Username, Video{
			AddedBy: payload.AddedBy,
			VideoID: payload.VideoID,
		})
		if err == nil {
			log.Println("video added to playlist")
		}
		return err
	} else if event.Type == "RemoveVideo" {
		var payload RemoveVideoPayload
		err := mapstructure.Decode(event.Payload, &payload)
		if err != nil {
			return ErrInvalidPayload
		}
		log.Println("message received")

		err = exc.hub.SendMessage(payload.Username, MessageEnvelope{
			Type: event.Type,
			Payload: VideoRemovedMessage{
				VideoID: payload.VideoID,
			},
		})

		if err != nil && errors.Is(err, ErrNoConnectionFound) {
			// no-op
			// the connected user might be on another instance
		} else if err != nil {
			return err
		}

		log.Println("message sent to socket")

		err = exc.playlist.RemoveVideo(payload.Username, payload.VideoID)
		if err == nil {
			log.Println("video removed from playlist")
		}
		return err
	}
	return ErrUnknownEventType
}

func (exc EventStream) readMessage(ctx context.Context) error {
	m, err := exc.reader.ReadMessage(ctx)
	if err != nil {
		return err
	}

	var data Event

	err = json.Unmarshal(m.Value, &data)
	if err != nil {
		return err
	}

	return exc.OnEvent(data)
}

func (exc EventStream) writeMessage(ctx context.Context, msg interface{}) error {
	b, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return exc.writer.WriteMessages(context.Background(), kafka.Message{
		Value: b,
	})
}

func (exc EventStream) Run(sigint chan os.Signal) error {
loop:
	for {
		select {
		case <-sigint:
			break loop
		default:
			err := exc.readMessage(context.Background())
			if err != nil {
				log.Println(err)
			}
		}
	}
	return exc.reader.Close()
}
