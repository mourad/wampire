package wampire

import (
	"fmt"
	"log"
	"sync"
	"time"
)

type Router struct {
	peers  map[ID]Peer
	broker Broker
	dealer Dealer
	exit   chan struct{}
	wg     *sync.WaitGroup
}

func NewRouter() *Router {
	return &Router{
		peers:  make(map[ID]Peer),
		broker: NewBroker(),
		dealer: NewDealer(),
		exit:   make(chan struct{}),
		wg:     &sync.WaitGroup{},
	}
}

func (r *Router) Accept(p Peer) error {
	timeout := time.NewTimer(time.Second * 1)
	select {
	case rcvMessage := <-p.Receive():
		timeout.Stop()
		h, ok := rcvMessage.(*Hello)
		if !ok {
			err := fmt.Sprintf("Unexpected type on Accept: %d", rcvMessage.MsgType())
			log.Println(err)

			return fmt.Errorf(err)
		}

		var response Message
		if r.authenticate(h) {
			response = &Abort{
				Id: h.Id,
			}
		}

		//answer welcome
		response = &Welcome{
			Id: h.Id,
		}
		p.Send(response)
		//if all goes fine
		go r.handleSession(p)

		return nil
	case <-timeout.C:
		log.Println("Timeout error waiting Hello Message")
		return fmt.Errorf("Timeout error waiting Hello Message")
	}
}

func (r *Router) Terminate() {
	log.Println("Invoked Router terminated!")
	close(r.exit)
	r.wg.Wait()
	log.Println("Router terminated!")
}

func (r *Router) handleSession(p Peer) {
	defer log.Println("Exit session handler from peer ", p.ID())
	defer r.wg.Done()
	defer p.Terminate()

	r.wg.Add(1)

	for {
		select {
		case msg, open := <-p.Receive():
			if !open {
				log.Println("Closing handled session from closed receive chan")
				return
			}

			switch msg.(type) {
			case *Publish:
				log.Println("Received Publish")
				response := r.broker.Publish(msg, p)
				p.Send(response)
			case *Subscribe:
				log.Println("Received Subscribe")
				response := r.broker.Subscribe(msg, p)
				p.Send(response)
			default:
				log.Println("Unhandled message")
				p.Send(&Error{Request: ID("123")})
			}

		case <-r.exit:
			log.Println("Shutting down session handler from peer ", p.ID())
			return
		}
	}
}

func (r *Router) authenticate(msg Message) bool {
	return true
}
