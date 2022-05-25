package main

import (
	mom "distribuidos/tp2/server/common/message_middleware/message_middleware"
	"fmt"
	"strings"

	log "github.com/sirupsen/logrus"
)

type MiddlewareAdmin struct {
	mom     *mom.MessageMiddleware
	control chan mom.Message
	table   ControlTable
	quit    chan bool
}

func StartAdmin(
	mom_address string,
	quit chan bool,
) (*MiddlewareAdmin, error) {

	msg_middleware, err := mom.Start(mom_address, false)
	if err != nil {
		return nil, fmt.Errorf("Couldn't connect to mom: %v", err)
	}

	control, err := msg_middleware.Read_queue(mom.QueueConfig{
		Name:  "admin_control",
		Class: "worker",
	})
	if err != nil {
		msg_middleware.Finish()
		return nil, fmt.Errorf("Couldn't declare control queue: %v", err)
	}

	log.Infof("Message Middleware Admin started")

	return &MiddlewareAdmin{
		mom:     msg_middleware,
		control: control,
		table:   New_control_table(),
		quit:    quit,
	}, nil
}

func (self *MiddlewareAdmin) Run() {

Loop:
	for {
		select {
		case <-self.quit:
			break Loop
		case m := <-self.control:
			params := strings.SplitN(m.Body, ",", 2)
			switch params[0] {
			case "new_write":
				self.handle_new_write_queue(params[1])
			case "new_read":
				self.handle_new_read_queue(params[1])
			case "finish":
				self.handle_queue_finish(params[1])
			default:
				log.Errorf("Received invalid message. Topic: %v, Body: %v", m.Topic, m.Body)
			}
		}
	}

	self.mom.Finish()
	log.Infof("Middleware Admin has finished")
}

func (self *MiddlewareAdmin) handle_new_write_queue(queue_config string) {
	log.Debugf("New Queue Writer: %v", queue_config)
	//Parsear
	params := strings.Split(queue_config, ",")
	if len(params) != 5 {
		log.Errorf("Received new queue with invalid params: %v", queue_config)
		return
	}
	q_config := mom.QueueConfig{
		Name:      params[0],
		Class:     params[1],
		Topic:     params[2],
		Source:    params[3],
		Direction: params[4],
	}
	//Declarar la cola con el middleware
	queue, err := self.mom.Write_queue(q_config)
	if err != nil {
		log.Errorf("Couldn't initialize queue with config %v: %v", queue_config, err)
		return
	}
	//Introducir en table, con el callback
	self.table.Add_new_writer(q_config.Name, func(reader_count uint) {
		log.Debugf("Callback of %v has been called for %v readers", q_config.Name, reader_count)
		for i := uint(0); i < reader_count; i++ {
			queue <- mom.Message{
				Topic: "finish",
				Body:  "finish",
			}
		}
	})
}

func (self *MiddlewareAdmin) handle_new_read_queue(queue_name string) {
	log.Debugf("New Queue Reader: %v", queue_name)
	self.table.Add_new_reader(queue_name)
}

func (self *MiddlewareAdmin) handle_queue_finish(queue_name string) {
	log.Debugf("Queue finish: %v", queue_name)
	self.table.Decrease_writer(queue_name)
}
