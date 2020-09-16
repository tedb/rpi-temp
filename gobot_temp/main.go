package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"path/filepath"
	"strconv"
	"strings"
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
const oneWireBasePath = "/sys/bus/w1/devices/w1_bus_master1/"

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
			data, err := ReadAllTemps(oneWireBasePath)
			if err != nil {
				log.Fatal("Error reading temperatures", err)
			}

			mqttJSON, err := json.Marshal(FeedUpdate{Feeds: data})
			if err != nil {
				log.Fatal("JSON did not marshal", err)
			}
			log.Printf("%s", mqttJSON)

			topic := config.Username + "/groups/" + config.Group
			mqttAdaptor.Publish(topic, mqttJSON)
		})
	}

	robot := gobot.NewRobot("RPiTemp",
		[]gobot.Connection{mqttAdaptor},
		work,
	)

	robot.Start()
}

// ReadAllTemps uses bulk read to get all temperature probes
func ReadAllTemps(path string) (map[string]float64, error) {
	temps := make(map[string]float64)

	thermBulkRead := filepath.Join(path, "therm_bulk_read")
	err := ioutil.WriteFile(thermBulkRead, []byte("trigger\n"), 0)
	if err != nil {
		return nil, err
	}

	// Wait for bulk reads to complete
	for {
		data, err := ioutil.ReadFile(thermBulkRead)
		if err != nil {
			return nil, err
		}
		if strings.TrimSpace(string(data)) == "1" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}

	probeDirs, err := filepath.Glob(path + "28-*")
	if err != nil {
		return nil, err
	}

	for _, dirName := range probeDirs {
		dirName := dirName

		name, degC, degF, err := readTemp(dirName)
		if err != nil {
			return nil, err
		}

		temps["temp-"+name+"-c"] = degC
		temps["temp-"+name+"-f"] = degF
	}

	return temps, nil
}

func readTemp(dirName string) (name string, degC, degF float64, err error) {
	rawName, err := ioutil.ReadFile(filepath.Join(dirName, "name"))
	if err != nil {
		return
	}

	name = strings.TrimPrefix(strings.TrimSpace(string(rawName)), "28-")

	rawMilliDegC, err := ioutil.ReadFile(filepath.Join(dirName, "temperature"))
	if err != nil {
		return
	}

	milliDegC, err := strconv.ParseFloat(strings.TrimSpace(string(rawMilliDegC)), 32)
	if err != nil {
		return
	}

	degC = milliDegC / 1000.0 //nolint:gomnd
	degF = degC*1.8 + 32      //nolint:gomnd

	return
}
