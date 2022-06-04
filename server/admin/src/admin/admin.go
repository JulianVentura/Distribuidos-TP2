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
	Posts         chan mom.Message
	Comments      chan mom.Message
	AverageResult chan mom.Message
	BestMeme      chan mom.Message
	SchoolMemes   chan mom.Message
}

type Admin struct {
	queues              AdminQueues
	skt                 *socket.ServerSocket
	mutex               sync.Mutex
	computationFinished chan bool //TODO: Ver si dejar
	quit                chan bool
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
		queues:              queues,
		skt:                 &skt,
		mutex:               sync.Mutex{},
		computationFinished: make(chan bool, 2),
		quit:                quit,
	}

	go func() {
		<-self.quit
		self.quit <- true
		self.closeServer()
	}()

	return self, nil
}

func (self *Admin) closeServer() {
	self.mutex.Lock()
	if self.skt != nil {
		_ = self.skt.Close()
		self.skt = nil
	}
	self.mutex.Unlock()
}

func (self *Admin) finish() {
	log.Infof("Shuting down Server Admin")
	self.closeServer()
}

func (self *Admin) Run() {
	log.Infof("Server Admin started")
	self.clientWorker()
}

func (self *Admin) clientWorker() {

	//Esperar por un nuevo cliente
	client, err := self.waitForNewClient()
	if err != nil {
		log.Errorf("Error waiting for new client")
		return
	}

	//We must close client connection at finish
	defer client.Close()

	//Recibir el stream de datos
	err = self.receiveStreamFromClient(client) //Problema: Si enviamos quit y no hay datos, nos bloqueamos
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
	err = self.sendResultsToClient(client)
	if err != nil {
		log.Errorf("Error sending results to client: %v", err)
	}
}

func (self *Admin) waitForNewClient() (*socket.TCPConnection, error) {

	defer self.closeServer()

	log.Infof("Waiting for client arrival")
	skt, err := self.skt.Accept()

	if err != nil {
		return nil, err
	}
	log.Infof("New Client has arrived")
	return &skt, nil
}

func (self *Admin) sendResultsToClient(client *socket.TCPConnection) error {
	log.Infof("Sending results to client")
	//Get computation results
	avgMsg := <-self.queues.AverageResult
	bestMemeMsg := <-self.queues.BestMeme
	schoolMemes := make([]string, 0, 100)
	for memeUrl := range self.queues.SchoolMemes {
		schoolMemes = append(schoolMemes, memeUrl.Body)
	}

	avg, err := strconv.ParseFloat(avgMsg.Body, 64)
	if err != nil {
		//TODO: Ver si devolvemos error al cliente
		return fmt.Errorf("Critical error, couldn't parse average result")
	}

	//Send results to client
	protocol.Send(client, &protocol.Response{
		PostScoreAvg:      avg,
		BestSentimentMeme: []byte(bestMemeMsg.Body),
		SchoolMemes:       schoolMemes,
	})

	return nil
}

func (self *Admin) receiveStreamFromClient(client *socket.TCPConnection) error {
	log.Debugf("Receiving streams from client")
	//Recibir un mensaje del cliente, parsearlo, encolarlo en la cola adecuada, repetir
	postStreamFinished := false
	commentStreamFinished := false

	postsReceived := 0
	commentsReceived := 0
Loop:
	for {
		select {
		case <-self.quit:
			return fmt.Errorf("Stream canceled by request")
		default:
			//TODO: Pasar a config
			//TODO: Pasar a un canal directamente desde el propio socket, eso es mas Golang
			message, err := protocol.ReceiveWithTimeout(client, time.Second) //TODO: Esto parecería no estar funcionando
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
				postsReceived += 1
				if postsReceived%10000 == 0 {
					log.Infof("Se recibieron %v posts", postsReceived)
				}
			case *protocol.Comment:
				log.Debugf("Received comment: %v", m.Comment)
				commentsReceived += 1
				if commentsReceived%10000 == 0 {
					log.Infof("Se recibieron %v comments", commentsReceived)
				}
				self.queues.Comments <- mom.Message{Body: m.Comment}
			case *protocol.PostFinished:
				log.Infof("Client poststream has finished")
				close(self.queues.Posts)
				postStreamFinished = true
			case *protocol.CommentFinished:
				log.Infof("Client comment stream has finished")
				close(self.queues.Comments)
				commentStreamFinished = true
			}

			if commentStreamFinished && postStreamFinished {
				break Loop
			}
		}
	}

	log.Infof("Client stream has finished")

	return nil
}
