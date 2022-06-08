package client

import (
	"fmt"
	"time"

	mom "distribuidos/tp2/server/common/message_middleware"
	"distribuidos/tp2/server/common/message_middleware/client/admin_proxy"
	rq "distribuidos/tp2/server/common/message_middleware/client/read_queue"
	workers "distribuidos/tp2/server/common/message_middleware/client/workers"
	wq "distribuidos/tp2/server/common/message_middleware/client/write_queue"

	amqp "github.com/streadway/amqp"
)

type MessageMiddlewareConfig struct {
	Address                string
	NotifyAdmin            bool
	ChannelBufferSize      uint
	MessageBatchSizeTarget uint
	MessageBatchTimeout    time.Duration
}

type MessageMiddleware struct {
	connection *amqp.Connection
	writers    []*workers.WriteWorker
	readers    []*workers.ReadWorker
	config     MessageMiddlewareConfig
}

const WRITER_WORKERS_INIT_SIZE = 1
const READER_WORKERS_INIT_SIZE = 1

func Start(config MessageMiddlewareConfig) (*MessageMiddleware, error) {

	self := &MessageMiddleware{
		writers: make([]*workers.WriteWorker, 0, WRITER_WORKERS_INIT_SIZE),
		readers: make([]*workers.ReadWorker, 0, READER_WORKERS_INIT_SIZE),
		config:  config,
	}

	conn, err := amqp.Dial(config.Address)
	if err != nil {
		return nil, fmt.Errorf("Failed to connect to RabbitMQ: %v", err)
	}

	self.connection = conn

	return self, nil
}

func (self *MessageMiddleware) newChannel() (*amqp.Channel, error) {
	ch, err := self.connection.Channel()
	if err != nil {
		return nil, fmt.Errorf("Failed to open a channel: %v", err)
	}

	err = ch.Qos(1, 0, true)

	if err != nil {
		return nil, err
	}

	return ch, nil
}

func (self *MessageMiddleware) Finish() error {

	for _, reader := range self.readers {
		reader.Finish()
	}

	for _, writer := range self.writers {
		writer.Finish()
	}

	self.connection.Close()

	return nil
}

func (self *MessageMiddleware) NewQueue(config mom.QueueConfig) (chan mom.Message, error) {
	switch config.Direction {
	case "read":
		return self.ReadQueue(config)
	case "write":
		return self.WriteQueue(config)
	default:
		return nil, fmt.Errorf("Error trying to create a queue: Direction %v is not recognized", config.Direction)
	}
}

func (self *MessageMiddleware) ReadQueue(config mom.QueueConfig) (chan mom.Message, error) {
	var err error = nil
	var queue rq.ReadQueue = nil

	channel := make(chan mom.Message, self.config.ChannelBufferSize)
	ch, err := self.newChannel()
	if err != nil {
		return nil, err
	}
	switch config.Class {
	case "worker":
		admin, err := admin_proxy.NewAdminProxy(ch, self.config.NotifyAdmin)
		if err != nil {
			return nil, err
		}
		queue, err = rq.NewReadWorkerQueue(&config, ch, &admin)
	case "topic":
		queue, err = rq.NewReadTopicQueue(&config, ch)
	case "fanout":
		queue, err = rq.NewReadFanoutQueue(&config, ch)
	default:
		return nil, fmt.Errorf("Error trying to create a read queue: Class %v is not recognized", config.Class)
	}

	worker := workers.StartReadWorker(queue, channel)

	self.readers = append(self.readers, worker)

	return channel, nil
}

func (self *MessageMiddleware) WriteQueue(config mom.QueueConfig) (chan mom.Message, error) {
	var err error = nil
	var queue wq.WriteQueue = nil

	channel := make(chan mom.Message, self.config.ChannelBufferSize)
	ch, err := self.newChannel()
	if err != nil {
		return nil, err
	}
	admin, err := admin_proxy.NewAdminProxy(ch, self.config.NotifyAdmin)
	if err != nil {
		return nil, err
	}
	switch config.Class {
	case "worker":
		queue, err = wq.NewWriteWorkerQueue(&config, ch, &admin)
	case "topic":
		queue, err = wq.NewWriteTopicQueue(&config, ch, &admin)
	case "fanout":
		queue, err = wq.NewWriteFanoutQueue(&config, ch, &admin)
	default:
		return nil, fmt.Errorf("Error trying to create a write queue: Class %v is not recognized", config.Class)
	}

	wConfig := workers.WriteWorkerConfig{
		MessageBatchSizeTarget: self.config.MessageBatchSizeTarget,
		MessageBatchTimeout:    self.config.MessageBatchTimeout,
	}

	worker := workers.StartWriteWorker(wConfig, channel, queue)

	self.writers = append(self.writers, worker)

	return channel, nil
}

// func (self *MessageMiddleware) notifyNewWriteQueueToAdmin(config *QueueConfig) error {
// 	message := fmt.Sprintf("new_write,%v,%v,%v,%v,%v",
// 		config.Name,
// 		config.Class,
// 		config.Topic,
// 		config.Source,
// 		config.Direction)

// 	return self.notifyToAdmin(message)
// }

// func (self *MessageMiddleware) notifyNewReadQueueToAdmin(queueOrigin string) error {
// 	message := fmt.Sprintf("new_read,%v", queueOrigin)
// 	return self.notifyToAdmin(message)
// }

// func (self *MessageMiddleware) notifyToAdmin(message string) error {
// 	if !self.config.NotifyAdmin {
// 		return nil
// 	}
// 	targetQueue := "admin_control"

// 	err := publish([]string{message}, "", targetQueue, self.channel)

// 	return err
// }

/*

Refactorización del middleware

Requerimientos:

- Mantener la misma API.
- Lograr un código más comprensible

Idea:

- Crear una nueva cola consiste de:

1. Parsear el objeto QueueConfig recibido y determinar si es válido.
	Al haber hecho esto se sabrá exactamente el tipo de cola a crear, por lo que podrá agregarse
	un nuevo campo en la config (privado) que lo indique (tipo enum)
2. Una vez vuelto de la config queremos crear la cola en sí, que consistirá en:
	a. Crear una nueva conexión (channel)
	b. Si la cola es de lectura y se bindea a un exchange entonces bindearla al topic 'finish' del admin
	c. Si la cola es de escritura notificar al admin
	d. Si la cola es de lectura y tipo worker, notificar al admin de un nuevo lector (Qué pasa si no es worker? Se notifica el fin?)
3.

Abstracciones:

// - Parseador: Determina si la config es válida y completa campo __type para poder parsearla desde el middle
- WriteQueue: A partir de una QueueConfig de la cuál se haya detectado direction "write" parsea la clase
	e invoca a alguna de las siguientes, pasandole la config:
		* WriteWorkerQueue: Implementa interfaz WriteQueue

- middleware: Al recibir la QueueConfig parsea la clase y direction para invocar a las funciones correctas:
	* Primero crea un nuevo channel
	* Luego creará una instancia de AdminProxy, a partir de notifyAdmin y el channel
	* Luego determina si es read o work, invocando a cada método según corresponda
		- Si es read, parseará la clase y creará alguna de las siguientes estructuras:
			* ReadWorkerQueue
			* ReadTopicQueue
			* ReadFanoutQueue

		  # Cada una de ellas implementa la interfaz ReadQueue con los siguientes métodos:
		  	* read(): Retorna un nuevo canal amqp desde donde se recibirán los mensajes
			* close(): Finaliza la cola, cerrando la conexión del channel
		  # Cada una de ellas para ser construidas recibirá la QueueConfig, el channel y el adminProxy

		- Si es write, parseará la clase y creará alguna de las siguientes estructuras:
			* WriteWorkerQueue
			* WriteTopicQueue
			* WriteFanoutQueue

		  # Cada una de ellas implementa la interfaz ReadQueue con los siguientes métodos:
		  	* write(message []byte, topic string): Escribe los mensajes en la cola
		  	* close(): Cierra la cola y notifica al admin (de ser necesario)
		  # Cada una de ellas para ser construidas recibirá la QueueConfig, el channel y el adminProxy

- Si todo sale bien habremos creado un nuevo channel, una nueva cola y una instancia de adminProxy la cual habremos
utilizado para comunicarle al admin la nueva cola creada
- Restará invocar a un Writer o a un Reader pasándole la cola recién creada y la instancia de admin

Tanto el reader como el admin deberán encargarse de cumplir el protocolo al enviar los mensajes
El writer además tendrá la lógica del batch de mensajes

Al finalizar el middleware, será responsabilidad del writer cerrar la cola de escritura que posee, la cual a su vez
notificará al admin. Una vez hecho esto la propia cola deberá cerrar la conexión, ya que es su propiedad


Orden de los módulos:

en /server/common

message_middleware:
	- interface.go: Define QueueConfig y Message
	- client/
		- client.go (lo que ahora es middleware)
		- write_queue/
			- write_worker_queue.go
			- write_topic_queue.go
			- write_fanout_queue.go
		- read_queue/
			- read_worker_queue.go
			- read_topic_queue.go
			- read_fanout_queue.go
		- workers/
			- reader.go
			- writer.go
			- batch_table.go
		- protocol/
			- protocol.go
		- admin_proxy/
			- admin_proxy.go


mom_admin:
	- main.go
	- admin.go : Importa y utiliza message_middleware/client
	- Dockerfile
*/
