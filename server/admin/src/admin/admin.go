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

type AdminQueues struct {
	Posts          chan mom.Message
	Comments       chan mom.Message
	Average_result chan mom.Message
	Best_meme      chan mom.Message
}

type Admin struct {
	queues               AdminQueues
	skt                  *socket.ServerSocket
	mutex                sync.Mutex
	computation_finished chan bool
	quit                 chan bool
}

func New(
	address string,
	queues AdminQueues,
	quit chan bool,
) (*Admin, error) {

	skt, err := socket.NewServer(address)
	if err != nil {
		return nil, Err.Ctx("Couldn't create server", err)
	}

	self := &Admin{
		queues:               queues,
		skt:                  &skt,
		mutex:                sync.Mutex{},
		computation_finished: make(chan bool, 2),
		quit:                 quit,
	}

	go func() {
		<-self.quit
		self.quit <- true
		self.close_server()
	}()

	return self, nil
}

func (self *Admin) close_server() {
	self.mutex.Lock()
	if self.skt != nil {
		_ = self.skt.Close()
		self.skt = nil
	}
	self.mutex.Unlock()
}

func (self *Admin) finish() {
	log.Infof("Shuting down Server Admin")
	self.close_server()
}

func (self *Admin) Run() {
	log.Infof("Server Admin started")
	self.client_worker()
}

func (self *Admin) client_worker() {

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
		log.Errorf("Error receiving stream from client: %v", err)
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

	defer self.close_server()

	log.Infof("Waiting for client arrival")
	skt, err := self.skt.Accept()

	if err != nil {
		return nil, err
	}
	log.Infof("New Client has arrived")
	return &skt, nil
}

func (self *Admin) send_results_to_client(client *socket.TCPConnection) error {
	log.Infof("Sending results to client")
	//Get computation results
	avg_r_msg := <-self.queues.Average_result
	log.Infof("AVG Result: %v", avg_r_msg.Body)
	best_meme_msg := <-self.queues.Best_meme
	log.Infof("Best Meme: %v", best_meme_msg.Body)
	avg, err := strconv.ParseFloat(avg_r_msg.Body, 64)
	if err != nil {
		//TODO: Ver si devolvemos error al cliente
		return fmt.Errorf("Critical error, couldn't parse average result")
	}

	//Send results to client
	protocol.Send(client, &protocol.Response{
		Post_score_average:  avg,
		Best_sentiment_meme: best_meme_msg.Body,
	})

	return nil
}

func (self *Admin) receive_stream_from_client(client *socket.TCPConnection) error {
	log.Debugf("Receiving streams from client")
	//Recibir un mensaje del cliente, parsearlo, encolarlo en la cola adecuada, repetir
	post_stream_finished := false
	comment_stream_finished := false

	posts_received := 0
	comments_received := 0
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
				// log.Debugf("Received post: %v", m.Post)
				self.queues.Posts <- mom.Message{Body: m.Post}
				posts_received += 1
				if posts_received%10000 == 0 {
					log.Infof("Se recibieron %v posts", posts_received)
				}
			case *protocol.Comment:
				log.Debugf("Received comment: %v", m.Comment)
				comments_received += 1
				if comments_received%10000 == 0 {
					log.Infof("Se recibieron %v comments", comments_received)
				}
				self.queues.Comments <- mom.Message{Body: m.Comment}
			case *protocol.PostFinished:
				log.Infof("Client poststream has finished")
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

	log.Infof("Client stream has finished")

	return nil
}
