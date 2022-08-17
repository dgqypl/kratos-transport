package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-kratos/kratos/v2/encoding"
	api "github.com/tx7do/kratos-transport/_example/api/manual"
	"github.com/tx7do/kratos-transport/broker"
	"github.com/tx7do/kratos-transport/broker/kafka"
)

const (
	testBrokers = "localhost:9092"
	testTopic   = "test_topic"
	testGroupId = "a-group"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Printf("Headers: %+v, Humidity: %.2f Temperature: %.2f\n", headers, msg.Humidity, msg.Temperature)
	return nil
}

func main() {
	ctx := context.Background()

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	b := kafka.NewBroker(
		broker.OptionContext(ctx),
		broker.Addrs(testBrokers),
		broker.Codec(encoding.GetCodec("json")),
	)

	_, err := b.Subscribe(testTopic,
		api.RegisterHygrothermographHandler(handleHygrothermograph),
		func() broker.Any {
			return &api.Hygrothermograph{}
		},
		broker.SubscribeContext(ctx),
		broker.Queue(testGroupId),
	)
	if err != nil {
		fmt.Println(err)
	}

	<-interrupt
}
