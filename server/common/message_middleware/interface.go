package message_middleware

type Message struct {
	Body  string
	Topic string
}

type QueueConfig struct {
	Name      string
	Class     string
	Topic     string
	Source    string
	Direction string
}
