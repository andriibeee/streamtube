package queue

type AddNewVideoMessage struct {
	AddedBy string `json:"addedBy"`
	VideoID string `json:"videoID"`
}

type VideoRemovedMessage struct {
	VideoID string `json:"videoID"`
}

type MessageEnvelope struct {
	Type string `json:"type"`
	Payload interface{} `json:"payload"`
}