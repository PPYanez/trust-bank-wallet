package main

import (
	"context"
	"log"
	"strconv"
	"strings"
	"trust-bank/wallet/transactions"

	amqp "github.com/rabbitmq/amqp091-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func failOnError(err error, msg string) {
	if err != nil {
		log.Panicf("%s: %s", msg, err)
	}
}

func deposit(ctx context.Context, client transactions.TransactionClient, ammount string, userNumber string) {
	floatAmmount, _ := strconv.ParseFloat(ammount, 32)

	request := transactions.TransactionRequest{
		NroClienteOrigen: userNumber,
		NroClienteDestino: userNumber,
		Monto: float32(floatAmmount),
		Divisa: "USD",
		TipoOperacion: "deposito",
	}

	_, err := client.PersistTransaction(ctx, &request)
	if err != nil {
		log.Fatalf("Cannot send grpc transaction: %v", err)
	}
}

func withdraw(ctx context.Context, client transactions.TransactionClient, ammount string, userNumber string) {
	floatAmmount, _ := strconv.ParseFloat(ammount, 32)

	request := transactions.TransactionRequest{
		NroClienteOrigen: userNumber,
		NroClienteDestino: userNumber,
		Monto: float32(floatAmmount),
		Divisa: "USD",
		TipoOperacion: "giro",
	}

	_, err := client.PersistTransaction(ctx, &request)
	if err != nil {
		log.Fatalf("Cannot send grpc transaction: %v", err)
	}
}

func transfer(ctx context.Context, client transactions.TransactionClient, ammount string, originUserNumber string, destinationUserNumber string) {
	floatAmmount, _ := strconv.ParseFloat(ammount, 32)

	request := transactions.TransactionRequest{
		NroClienteOrigen: originUserNumber,
		NroClienteDestino: destinationUserNumber,
		Monto: float32(floatAmmount),
		Divisa: "USD",
		TipoOperacion: "transferencia",
	}

	_, err := client.PersistTransaction(ctx, &request)
	if err != nil {
		log.Fatalf("Cannot send grpc transaction: %v", err)
	}
}

func main() {
	ctx := context.Context(context.Background())

	// Open grpc connection
	grpcConn, err := grpc.Dial("localhost:8081", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Cannot dial: %v", err)
	}
	defer grpcConn.Close()

	// Define client stub
	client := transactions.NewTransactionClient(grpcConn)

	// Create and bind queue to RabbitMQ exchange
	rabbitMQconn := openConnection()
	defer rabbitMQconn.Close()

	ch := createChannel(rabbitMQconn)
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
			splittedMessage := strings.Split(string(d.Body), " ")

			if (splittedMessage[0] == "deposit") {
				deposit(ctx, client, splittedMessage[1], splittedMessage[2])
			} else if (splittedMessage[0] == "withdraw") {
				withdraw(ctx, client, splittedMessage[1], splittedMessage[2])
			} else if (splittedMessage[0] == "transfer") {
				transfer(ctx, client, splittedMessage[1], splittedMessage[2], splittedMessage[3])
			}
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
		"TrustBank",    // name
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
