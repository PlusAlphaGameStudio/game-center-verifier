package rabbitmq

import (
	"context"
	amqp "github.com/rabbitmq/amqp091-go"
)

func (client *Client) ConsumeRepeat(handler func(body []byte, corrId string, replyTo string)bool, done chan struct{}) {
	ctx := context.Background()

	deliveries, err := client.Consume()
	if err != nil {
		client.errLog.Printf("could not start consuming from %v: %v\n", client.QueueName(), err)
		return
	}

	// This channel will receive a notification when a channel closed event
	// happens. This must be different from Client.notifyChanClose because the
	// library sends only one notification and Client.notifyChanClose already has
	// a receiver in handleReconnect().
	// Recommended to make it buffered to avoid deadlocks
	chClosedCh := make(chan *amqp.Error, 1)
	client.channel.NotifyClose(chClosedCh)

loop:
	for {
		select {
		case <-ctx.Done():
			err := client.Close()
			if err != nil {
				client.errLog.Printf("close failed: %s\n", err)
			}
			break loop

		case amqErr := <-chClosedCh:
			// This case handles the event of closed channel e.g. abnormal shutdown
			client.errLog.Printf("AMQP Channel closed due to: %s\n", amqErr)

			deliveries, err = client.Consume()
			if err != nil {
				// If the AMQP channel is not ready, it will continue the loop. Next
				// iteration will enter this case because chClosedCh is closed by the
				// library
				client.errLog.Println("error trying to consume, will try again")
				continue
			}

			// Re-set channel to receive notifications
			// The library closes this channel after abnormal shutdown
			chClosedCh = make(chan *amqp.Error, 1)
			client.channel.NotifyClose(chClosedCh)

		case delivery := <-deliveries:
			bodyStr := string(delivery.Body)
			client.infoLog.Printf("received message from %v: %v\n", client.QueueName(), bodyStr)
			if handler(delivery.Body, delivery.CorrelationId, delivery.ReplyTo) {
				_ = delivery.Ack(false)
			} else {
				_ = delivery.Nack(false, false)
			}
		}
	}

	close(done)
}

