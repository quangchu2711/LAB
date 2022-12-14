package main

import (
    "fmt"
    "time"
    "strconv"
    mqtt "github.com/eclipse/paho.mqtt.golang"
)

var nodeMqttClient mqtt.Client
var valueSensor int

var messageNodeDevicePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    sensorDeviceMsg := string(msg.Payload())
    fmt.Printf("Received message: [%s] from topic: %s\n", sensorDeviceMsg, msg.Topic())
    valueSensor += 1
    if sensorDeviceMsg == "HUM1" {
        sendToBot(strconv.Itoa(valueSensor) + "%")
    }
    if sensorDeviceMsg == "TEMP1" {
        sendToBot("26℃")
    }
}

func mqttBegin(broker string, user string, pw string, messagePubHandler *mqtt.MessageHandler) mqtt.Client {
    var opts *mqtt.ClientOptions = new(mqtt.ClientOptions)

    opts = mqtt.NewClientOptions()
    opts.AddBroker(broker)
    opts.SetUsername(user)
    opts.SetPassword(pw) 
    opts.SetDefaultPublishHandler(*messagePubHandler)
    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }

    return client
}

func sendToBot(strMsg string ) {
    nodeMqttClient.Publish("xuong/device/telegram/sensor/status", 0, false, strMsg)
    fmt.Println("Publish: xuong/device/esp1170372/sensor/status" + ": " + strMsg)  
}

func main() {
    valueSensor = 40
    nodeMqttClient = mqttBegin("localhost:1883", "nmtam", "221220", &messageNodeDevicePubHandler)
    nodeMqttClient.Subscribe("xuong/device/espID/sensor/cmd", 1, nil)
    
    fmt.Println("Connected")
    for {
        time.Sleep(2 * time.Second)
    }
}