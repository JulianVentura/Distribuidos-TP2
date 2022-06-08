package write_queue

type WriteQueue interface {
	Write([]byte, string) error
	Close()
}
