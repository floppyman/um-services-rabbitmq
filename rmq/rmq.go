package rmq

import (
	"crypto/tls"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"time"
	
	amqp "github.com/rabbitmq/amqp091-go"
	"github.com/umbrella-sh/um-common/ext"
)

//goland:noinspection GoNameStartsWithPackageName
type RmqSession struct {
	name            string
	logReceiver     func(string, error)
	ssl             *tls.Config
	connection      *amqp.Connection
	channel         *amqp.Channel
	done            chan bool
	notifyConnClose chan *amqp.Error
	notifyChanClose chan *amqp.Error
	notifyConfirm   chan amqp.Confirmation
	isReady         bool
}

const (
	// When reconnecting to the server after connection failure
	reconnectDelay = 5 * time.Second
	
	// When setting up the channel after a channel exception
	reInitDelay = 2 * time.Second
	
	// When resending messages, the server didn't confirm
	resendDelay = 5 * time.Second
)

var (
	errNotConnected  = errors.New("not connected to a server")
	errAlreadyClosed = errors.New("already closed: not connected to the server")
	errShutdown      = errors.New("session is shutting down")
)

// New creates a new consumer state instance, and automatically
// attempts to connect to the server.
func New(queueName string, connAddr string) *RmqSession {
	session := RmqSession{
		ssl:  nil,
		name: queueName,
		done: make(chan bool),
	}
	go session.handleReconnect(connAddr)
	return &session
}

// NewTLS creates a new consumer state instance with TLS, and automatically
// attempts to connect to the server.
func NewTLS(queueName string, connAddr string, tlsCommonName string) *RmqSession {
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		InsecureSkipVerify: false,
		ClientAuth:         tls.NoClientCert,
	}
	if tlsCommonName != "" {
		tlsConfig.ServerName = tlsCommonName
	}
	
	session := RmqSession{
		ssl:  tlsConfig,
		name: queueName,
		done: make(chan bool),
	}
	go session.handleReconnect(connAddr)
	return &session
}

// CreateUrl creates a correctly formatted url for the rabbitmq server
func CreateUrl(rmq ConfigRmq) string {
	epi := rand.Intn(len(rmq.Endpoints)-0) + 0
	return fmt.Sprintf("amqp%s://%s:%s@%s.%s:%d/%s",
		ext.Iif(rmq.TlsEnabled, "s", ""),
		rmq.Username,
		rmq.Password,
		rmq.Endpoints[epi],
		rmq.QueueExtension,
		rmq.QueuePort,
		url.QueryEscape(rmq.VirtualHost))
}

// AttachLogReceiver sets the method for receiving logs from this session
func (session *RmqSession) AttachLogReceiver(receiver func(string, error)) {
	session.logReceiver = receiver
}

// handleReconnect will wait for a connection error on
// notifyConnClose, and then continuously attempt to reconnect.
func (session *RmqSession) handleReconnect(addr string) {
	for {
		session.isReady = false
		session.logReceiver("Attempting to connect", nil)
		
		conn, err := session.connect(addr)
		
		if err != nil {
			session.logReceiver("Failed to connect. Retrying...", err)
			
			select {
			case <-session.done:
				return
			case <-time.After(reconnectDelay):
			}
			continue
		}
		
		if done := session.handleReInit(conn); done {
			break
		}
	}
}

// connect will create a new AMQP connection
func (session *RmqSession) connect(addr string) (*amqp.Connection, error) {
	var conn *amqp.Connection
	var err error
	
	if session.ssl != nil {
		conn, err = amqp.DialTLS(addr, session.ssl)
	} else {
		conn, err = amqp.Dial(addr)
	}
	
	if err != nil {
		return nil, err
	}
	
	session.changeConnection(conn)
	session.logReceiver("Connected!", nil)
	return conn, nil
}

// handleReconnect will wait for a channel error
// and then continuously attempt to re-initialize both channels
func (session *RmqSession) handleReInit(conn *amqp.Connection) bool {
	for {
		session.isReady = false
		
		err := session.init(conn)
		
		if err != nil {
			session.logReceiver("Failed to initialize channel. Retrying...", err)
			
			select {
			case <-session.done:
				return true
			case <-time.After(reInitDelay):
			}
			continue
		}
		
		select {
		case <-session.done:
			return true
		case <-session.notifyConnClose:
			session.logReceiver("Connection closed. Reconnecting...", nil)
			return false
		case <-session.notifyChanClose:
			session.logReceiver("Channel closed. Re-running init...", nil)
		}
	}
}

// init will initialize channel & declare queue
func (session *RmqSession) init(conn *amqp.Connection) error {
	ch, err := conn.Channel()
	
	if err != nil {
		return err
	}
	
	err = ch.Confirm(false)
	if err != nil {
		return err
	}
	
	session.changeChannel(ch)
	session.isReady = true
	session.logReceiver("Setup!", nil)
	
	return nil
}

// changeConnection takes a new connection to the queue,
// and updates the close listener to reflect this.
func (session *RmqSession) changeConnection(connection *amqp.Connection) {
	session.connection = connection
	session.notifyConnClose = make(chan *amqp.Error)
	session.connection.NotifyClose(session.notifyConnClose)
}

// changeChannel takes a new channel to the queue,
// and updates the channel listeners to reflect this.
func (session *RmqSession) changeChannel(channel *amqp.Channel) {
	session.channel = channel
	session.notifyChanClose = make(chan *amqp.Error)
	session.notifyConfirm = make(chan amqp.Confirmation, 1)
	session.channel.NotifyClose(session.notifyChanClose)
	session.channel.NotifyPublish(session.notifyConfirm)
}

// Push will push data onto the queue, and wait for a confirmation.
// If no confirmations are received until within the resendTimeout,
// it continuously re-sends messages until a confirmation is received.
// This will block until the server sends a confirmation. Errors are
// only returned if the push action itself fails, see UnsafePush.
func (session *RmqSession) Push(data []byte) error {
	if !session.isReady {
		return errors.New("failed to push push: not connected")
	}
	for {
		err := session.UnsafePush(data)
		if err != nil {
			session.logReceiver("Push failed. Retrying...", err)
			select {
			case <-session.done:
				return errShutdown
			case <-time.After(resendDelay):
			}
			continue
		}
		select {
		case confirm := <-session.notifyConfirm:
			if confirm.Ack {
				session.logReceiver("Push confirmed!", nil)
				return nil
			}
		case <-time.After(resendDelay):
		}
		session.logReceiver("Push didn't confirm. Retrying...", nil)
	}
}

// UnsafePush will push to the queue without checking for
// confirmation. It returns an error if it fails to connect.
// No guarantees are provided for whether the server will
// receive the message.
func (session *RmqSession) UnsafePush(data []byte) error {
	if !session.isReady {
		return errNotConnected
	}
	return session.channel.Publish(
		"",           // Exchange
		session.name, // Routing key
		false,        // Mandatory
		false,        // Immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        data,
		},
	)
}

// Stream will continuously put queue items on the channel.
// It is required to call delivery.Ack when it has been
// successfully processed, or delivery.Nack when it fails.
// Ignoring this will cause data to build up on the server.
func (session *RmqSession) Stream() (<-chan amqp.Delivery, error) {
	if !session.isReady {
		return nil, errNotConnected
	}
	return session.channel.Consume(
		session.name,
		"",    // Consumer
		false, // Auto-Ack
		false, // Exclusive
		false, // No-local
		false, // No-Wait
		nil,   // Args
	)
}

// Close will cleanly shut down the channel and connection.
func (session *RmqSession) Close() error {
	if !session.isReady {
		return errAlreadyClosed
	}
	err := session.channel.Close()
	if err != nil {
		return err
	}
	err = session.connection.Close()
	if err != nil {
		return err
	}
	close(session.done)
	session.isReady = false
	return nil
}
