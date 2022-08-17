package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v2/encoding"
	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	api "github.com/tx7do/kratos-transport/_example/api/manual"
	"github.com/tx7do/kratos-transport/broker"
)

var addrTestCases = []struct {
	name        string
	description string
	addrs       map[string]string // expected address : set address
}{
	{
		"commonOpts",
		"set common addresses through a common.Option in constructor",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"commonInit",
		"set common addresses through a common.Option in common.Init()",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"natsOpts",
		"set common addresses through the nats.Option in constructor",
		map[string]string{
			"nats://192.168.10.1:5222": "192.168.10.1:5222",
			"nats://10.20.10.0:4222":   "10.20.10.0:4222"},
	},
	{
		"default",
		"check if default Address is set correctly",
		map[string]string{
			"nats://127.0.0.1:4222": "",
		},
	},
}

// TestInitAddrs tests issue #100. Ensures that if the addrs is set by an option in init it will be used.
func TestInitAddrs(t *testing.T) {

	for _, tc := range addrTestCases {
		t.Run(fmt.Sprintf("%s: %s", tc.name, tc.description), func(t *testing.T) {

			var br broker.Broker
			var addrs []string

			for _, addr := range tc.addrs {
				addrs = append(addrs, addr)
			}

			switch tc.name {
			case "commonOpts":
				// we know that there are just two addrs in the dict
				br = NewBroker(broker.WithAddress(addrs[0], addrs[1]))
				_ = br.Init()
			case "commonInit":
				br = NewBroker()
				// we know that there are just two addrs in the dict
				_ = br.Init(broker.WithAddress(addrs[0], addrs[1]))
			case "natsOpts":
				nopts := nats.GetDefaultOptions()
				nopts.Servers = addrs
				br = NewBroker(Options(nopts))
				_ = br.Init()
			case "default":
				br = NewBroker()
				_ = br.Init()
			}

			natsBroker, ok := br.(*natsBroker)
			if !ok {
				t.Fatal("Expected common to be of types *natsBroker")
			}
			// check if the same amount of addrs we set has actually been set, default
			// have only 1 address nats://127.0.0.1:4222 (current nats code) or
			// nats://localhost:4222 (older code version)
			if len(natsBroker.opts.Addrs) != len(tc.addrs) && tc.name != "default" {
				t.Errorf("Expected Addr count = %d, Actual Addr count = %d",
					len(natsBroker.opts.Addrs), len(tc.addrs))
			}

			for _, addr := range natsBroker.opts.Addrs {
				_, ok := tc.addrs[addr]
				if !ok {
					t.Errorf("Expected '%s' has not been set", addr)
				}
			}
		})
	}
}

const (
	localBroker = "nats://127.0.0.1:4222"
	testTopic   = "test"
)

func handleHygrothermograph(_ context.Context, topic string, headers broker.Headers, msg *api.Hygrothermograph) error {
	log.Printf("Humidity: %.2f Temperature: %.2f\n", msg.Humidity, msg.Temperature)
	return nil
}

func Test_Publish_WithRawData(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(localBroker),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}

	var msg api.Hygrothermograph
	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		buf, _ := json.Marshal(&msg)
		err := b.Publish(testTopic, buf)
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		fmt.Printf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	fmt.Printf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithRawData(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(localBroker),
	)
	defer b.Disconnect()

	_ = b.Connect()

	_, err := b.Subscribe(testTopic,
		api.RegisterHygrothermographRawHandler(handleHygrothermograph),
		nil,
		broker.WithSubscribeContext(ctx),
	)
	assert.Nil(t, err)

	<-interrupt
}

func Test_Publish_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(localBroker),
		broker.WithCodec(encoding.GetCodec("json")),
	)

	_ = b.Init()

	if err := b.Connect(); err != nil {
		t.Logf("cant connect to broker, skip: %v", err)
		t.Skip()
	}

	var msg api.Hygrothermograph
	const count = 10
	for i := 0; i < count; i++ {
		startTime := time.Now()
		msg.Humidity = float64(rand.Intn(100))
		msg.Temperature = float64(rand.Intn(100))
		err := b.Publish(testTopic, msg)
		assert.Nil(t, err)
		elapsedTime := time.Since(startTime) / time.Millisecond
		fmt.Printf("Publish %d, elapsed time: %dms, Humidity: %.2f Temperature: %.2f\n",
			i, elapsedTime, msg.Humidity, msg.Temperature)
	}

	fmt.Printf("total send %d messages\n", count)

	<-interrupt
}

func Test_Subscribe_WithJsonCodec(t *testing.T) {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	ctx := context.Background()

	b := NewBroker(
		broker.WithOptionContext(ctx),
		broker.WithAddress(localBroker),
		broker.WithCodec(encoding.GetCodec("json")),
	)
	defer b.Disconnect()

	_ = b.Connect()

	_, err := b.Subscribe(testTopic,
		api.RegisterHygrothermographJsonHandler(handleHygrothermograph),
		func() broker.Any {
			return &api.Hygrothermograph{}
		},
		broker.WithSubscribeContext(ctx),
	)
	assert.Nil(t, err)

	<-interrupt
}
