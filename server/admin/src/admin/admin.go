package admin

import (
	Err "distribuidos/tp2/common/errors"
	"distribuidos/tp2/common/protocol"
	"distribuidos/tp2/common/socket"
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"fmt"
	"strconv"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

type AdminConfig struct {
	//Logging
	Log_level string
	//Server Connection
	Server_address string
	//Mom Connection
	Mom_address string
	//Number of workers
}

type AdminQueues struct {
	Posts          chan mom.Message
	Comments       chan mom.Message
	Average_result chan mom.Message
}

type Admin struct {
	queues               AdminQueues
	config               AdminConfig
	skt                  *socket.ServerSocket
	computation_finished chan bool
	quit                 chan bool
	finished             sync.WaitGroup
}

func New(
	config AdminConfig,
	queues AdminQueues,
	quit chan bool,
) (*Admin, error) {

	skt, err := socket.NewServer(config.Server_address)
	if err != nil {
		return nil, Err.Ctx("Couldn't create server", err)
	}

	self := &Admin{
		queues:               queues,
		config:               config,
		skt:                  &skt,
		computation_finished: make(chan bool, 2),
		quit:                 quit,
	}

	//self.warmup_wait()

	return self, nil
}

func (self *Admin) Finish() {
	if self.skt != nil {
		_ = self.skt.Close()
		self.skt = nil
	}
	self.quit <- true
	self.finished.Wait()
}

func (self *Admin) Run() {
	self.finished.Add(1)
	go self.client_worker()

	log.Infof("Server Admin started")
	<-self.quit
	log.Infof("Shuting down Server Admin")
	self.Finish()
}

func (self *Admin) client_worker() {

	//We must free resources at finish
	defer func() {
		log.Debugf("Client worker has finished")
		self.finished.Done()
		self.Finish()
	}()

	//Esperar por un nuevo cliente
	client, err := self.wait_for_new_client()
	if err != nil {
		log.Errorf("Error waiting for new client")
		return
	}

	//We must close client connection at finish
	defer client.Close()

	//Recibir el stream de datos
	err = self.receive_stream_from_client(client) //Problema: Si enviamos quit y no hay datos, nos bloqueamos
	if err != nil {
		return
	}
	//Esperar a que finalice el cómputo de datos
	// select {
	// case <-self.computation_finished:
	// 	break
	// case <-self.quit:
	// 	return
	// }
	//Enviar los resultados
	err = self.send_results_to_client(client)
	if err != nil {
		log.Errorf("Error sending results to client: %v", err)
	}
}

func (self *Admin) wait_for_new_client() (*socket.TCPConnection, error) {
	log.Infof("Waiting for client arrival")
	skt, err := self.skt.Accept()

	_ = self.skt.Close()
	self.skt = nil

	if err != nil {
		return nil, err
	}
	log.Infof("New Client has arrived")
	return &skt, nil
}

func (self *Admin) send_results_to_client(client *socket.TCPConnection) error {
	log.Debugf("Sending results to client")
	//Get computation results
	msg := <-self.queues.Average_result
	log.Infof("Resultado de AVG: %v", msg.Body)
	avg, err := strconv.ParseFloat(msg.Body, 64)
	if err != nil {
		//TODO: Ver si devolvemos error al cliente
		return fmt.Errorf("Critical error, couldn't parse average result")
	}

	//Send results to client
	protocol.Send(client, &protocol.Response{
		Post_score_average: avg,
	})

	return nil
}

func (self *Admin) receive_stream_from_client(client *socket.TCPConnection) error {
	log.Debugf("Receiving streams from client")
	//Recibir un mensaje del cliente, parsearlo, encolarlo en la cola adecuada, repetir
	post_stream_finished := false
	comment_stream_finished := true

	posts_received := 0
Loop:
	for {
		select {
		case <-self.quit:
			return fmt.Errorf("Stream canceled by request")
		default:
			//TODO: Pasar a config
			//TODO: Pasar a un canal directamente desde el propio socket, eso es mas Golang
			message, err := protocol.Receive_with_timeout(client, time.Second) //TODO: Esto parecería no estar funcionando
			if err == socket.TimeoutError {
				continue
			}
			if err != nil {
				log.Errorf("Closing client connection because of error %v", err)
				return err
			}
			switch m := message.(type) {
			case *protocol.Post:
				log.Debugf("Received post: %v", m.Post)
				self.queues.Posts <- mom.Message{Body: m.Post}
				posts_received += 1
				if posts_received%10000 == 0 {
					log.Infof("Se recibieron %v posts", posts_received)
				}
			case *protocol.Comment:
				log.Debugf("Received comment: %v", m.Comment)
				self.queues.Comments <- mom.Message{Body: m.Comment}
			case *protocol.PostFinished:
				log.Infof("Client post stream has finished")
				close(self.queues.Posts)
				post_stream_finished = true
			case *protocol.CommentFinished:
				log.Infof("Client comment stream has finished")
				close(self.queues.Comments)
				comment_stream_finished = true
			}

			if comment_stream_finished && post_stream_finished {
				break Loop
			}
		}
	}

	log.Debugf("Client stream has finished")

	return nil
}
