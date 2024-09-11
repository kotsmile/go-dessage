package server

type MessageType string

const (
	ConnectType    MessageType = "connect"
	SendType       MessageType = "send"
	DisconnectType MessageType = "disconnect"
)

type Message struct {
	Address   string      `json:"address"`
	User      string      `json:"user"`
	Text      string      `json:"text"`
	Timestamp int64       `json:"timestamp"`
	Type      MessageType `json:"type"`
	internal  bool
}
