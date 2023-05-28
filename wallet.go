package main

import (
	"log"

	amqp "github.com/rabbitmq/amqp091-go"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func main() {
	// Create and bind queue to RabbitMQ exchange
	conn := openConnection()
	defer conn.Close()

	ch := createChannel(conn)
	defer ch.Close()

	declareExchange(ch)
	q := declareQueue(ch)
	bindQueue(ch, q)

	// Receive messages
	msgs, err := ch.Consume(
		q.Name, // queue
		"",     // consumer
		true,   // auto-ack
		false,  // exclusive
		false,  // no-local
		false,  // no-wait
		nil,    // args
	)
	failOnError(err, "Failed to register a consumer")

	var forever chan struct{}

	go func() {
		for d := range msgs {
			log.Printf(" [x] received %s", d.Body)
		}
	}()
	log.Print("Ejecutando consumer, para salir presione CTRL+C")

	<-forever
}

func openConnection() *amqp.Connection {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	failOnError(err, "Failed to connect to a RabbitMQ server")

	return conn
}

func createChannel(conn *amqp.Connection) *amqp.Channel {
	ch, err := conn.Channel()
	failOnError(err, "Failed to open a channel")

	return ch
}

func declareExchange(ch *amqp.Channel) {
	err := ch.ExchangeDeclare(
		"RabbitMQ", // name
		"direct",   // type
		true,       // durable
		false,      // auto-deleted
		false,      // internal
		false,      // no-wait
		nil,        // arguments
	)
	failOnError(err, "Failed to declare an exchange")
}

func declareQueue(ch *amqp.Channel) amqp.Queue {
	q, err := ch.QueueDeclare(
		"",    // name
		false, // durable
		false, // delete when unused
		true,  // exclusive
		false, // no-wait
		nil,   // arguments
	)
	failOnError(err, "Failed to declare a queue")

	return q
}

func bindQueue(ch *amqp.Channel, q amqp.Queue) {
	err := ch.QueueBind(
		q.Name,      // queue name
		"trustBank", // routing key
		"RabbitMQ",  // exchange
		false,
		nil,
	)
	failOnError(err, "Failed to bind a queue")
}
