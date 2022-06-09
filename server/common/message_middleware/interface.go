package message_middleware

type Message struct {
	Body  []byte
	Topic string
}

type QueueConfig struct {
	Name      string
	Class     string
	Topic     string
	Source    string
	Direction string
}
