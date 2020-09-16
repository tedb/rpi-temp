package main

import (
	"encoding/json"
	"errors"
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

		gobot.Every(60*time.Second, func() {
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

	err := triggerBulkRead(path)
	if err != nil {
		return nil, err
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

// triggerBulkRead accelerates reading the probes
// https://www.kernel.org/doc/html/latest/w1/slaves/w1_therm.html
func triggerBulkRead(path string) error {
	thermBulkRead := filepath.Join(path, "therm_bulk_read")
	err := ioutil.WriteFile(thermBulkRead, []byte("trigger\n"), 0)
	if err != nil {
		return err
	}

	interval := 100 * time.Millisecond
	counter := 30 * time.Second / interval

	// Wait for bulk reads to complete
Loop:
	for {
		data, err := ioutil.ReadFile(thermBulkRead)
		if err != nil {
			return err
		}

		// From docs: Reading therm_bulk_read will
		// return 0 if no bulk conversion pending,
		// -1 if at least one sensor still in conversion,
		// 1 if conversion is complete but at least one sensor
		// value has not been read yet.
		switch strings.TrimSpace(string(data)) {
		case "0":
			return errors.New("no bulk read pending, but it was triggered")
		case "-1":
			if counter == 0 {
				return errors.New("timed out waiting for bulk read")
			}
			log.Print("Waiting for temperature probes...")
			time.Sleep(interval)
			counter--
		case "1":
			break Loop
		}
	}

	return nil
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
