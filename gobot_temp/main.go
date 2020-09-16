package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/kelseyhightower/envconfig"
	"gobot.io/x/gobot"
	"gobot.io/x/gobot/platforms/mqtt"
)

// Config captures environment variables
type Config struct {
	Username string `required:"true"`
	Key      string `required:"true"`
	Group    string `required:"true"`
}

// FeedUpdate will be marshaled to JSON for MQTT
type FeedUpdate struct {
	Feeds map[string]float64 `json:"feeds"`
}

const mqttURL = "ssl://io.adafruit.com:8883"
const mqttClientID = "gobot"

func main() {
	var config Config
	err := envconfig.Process("ADAFRUIT_IO", &config)
	if err != nil {
		log.Fatal(err.Error())
	}

	mqttAdaptor := mqtt.NewAdaptorWithAuth(mqttURL, mqttClientID, config.Username, config.Key)
	mqttAdaptor.SetAutoReconnect(true)

	work := func() {
		mqttAdaptor.On(config.Username+"/errors", func(msg mqtt.Message) {
			log.Printf("Adafruit IO error: %s\n", msg.Payload())
		})

		mqttAdaptor.On(config.Username+"/throttle", func(msg mqtt.Message) {
			log.Printf("Adafruit IO throttled: %s\n", msg.Payload())
		})

		gobot.Every(1*time.Second, func() {
			// TODO: use real data
			data := FeedUpdate{
				Feeds: map[string]float64{
					"temp-011850aecaff": 99.1,
					"temp-0218508170ff": 98.2,
				},
			}
			mqttJSON, err := json.Marshal(data)
			if err != nil {
				log.Fatal("JSON did not marshal", err)
			}
			log.Printf("%s", mqttJSON)

			topic := config.Username + "/groups/" + config.Group
			log.Println(topic)
			mqttAdaptor.Publish(topic, mqttJSON)
		})
	}

	robot := gobot.NewRobot("RPiTemp",
		[]gobot.Connection{mqttAdaptor},
		work,
	)

	robot.Start()
}
