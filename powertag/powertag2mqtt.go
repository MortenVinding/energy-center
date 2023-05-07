package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"strings"
	"time"
)

const ProgNameMqtt string = "powertag2mqtt"

func main() {
	topic := flag.String("topic", "powertag/", "The topic name to/from which to publish/subscribe")
	broker := flag.String("broker", "tcp://192.168.0.20:1883", "The broker URI. ex: tcp://10.10.1.1:1883")
	password := flag.String("password", "", "The password (optional)")
	user := flag.String("user", "", "The User (optional)")
	id := flag.String("id", "powertag2mqtt", "The ClientID (optional)")
	cleansess := flag.Bool("clean", false, "Set Clean Session (default false)")
	qos := flag.Int("qos", 0, "The Quality of Service 0,1,2 (default 0)")
	store := flag.String("store", ":memory:", "The Store Directory (default use memory store)")
	flag.Parse()

	stat, _ := os.Stdin.Stat()
	if stat.Mode()&os.ModeCharDevice != 0 {
		fmt.Fprintf(os.Stderr, "%s: no data on stdin\n", ProgNameMqtt)
		fmt.Fprintf(os.Stderr, "%s expects data to be piped to stdin, i.e.:\n", ProgNameMqtt)
		fmt.Fprintf(os.Stderr, "    powertagd | powertag2influx\n")
		os.Exit(2)
	}

	MQTT.DEBUG = log.New(os.Stdout, "", 0)
	MQTT.ERROR = log.New(os.Stdout, "", 0)

	opts := MQTT.NewClientOptions()
	opts.AddBroker(*broker)
	opts.SetClientID(*id)
	opts.SetUsername(*user)
	opts.SetPassword(*password)
	opts.SetCleanSession(*cleansess)
	if *store != ":memory:" {
		opts.SetStore(MQTT.NewFileStore(*store))
	}
	opts.SetKeepAlive(60 * time.Second)
	opts.SetPingTimeout(1 * time.Second)

	client := MQTT.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	fmt.Printf("%s: connected to %s\n", ProgNameMqtt, *broker)

	lnscan := bufio.NewScanner(os.Stdin)
	for lnscan.Scan() {
		line := lnscan.Text()
		if strings.HasPrefix(line, "powertag,") {
			sanitized := strings.Replace(line, "powertag,", "", -1)
			splitted := strings.Split(sanitized, " ")
			if len(splitted) == 3 {
				tags := asMap(splitted[0])
				measures := asMap(splitted[1])
				// ts := splitted[2]

				_, idExist := tags["id"]
				if idExist {
					jsonStr, _ := json.Marshal(measures)

					token := client.Publish(*topic+tags["id"], byte(*qos), false, jsonStr)
					token.Wait()
				}

			}
		}
	}
}

func asMap(inputString string) map[string]string {
	m := make(map[string]string)

	entry := strings.Split(inputString, ",")
	for _, s := range entry {
		split := strings.Split(s, "=")
		m[split[0]] = split[1]
	}

	return m
}
