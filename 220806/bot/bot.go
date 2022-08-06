package main

import (
    "log"
    "fmt"
    "time"
    mqtt "github.com/eclipse/paho.mqtt.golang"
    "github.com/ghodss/yaml"
    "io/ioutil"    
)


var mqttClientHandleTele mqtt.Client
var mqttClientHandleSerial mqtt.Client

var  channel chan string

type Mqtt struct {
    Broker string

    SerialSrcTopic string
    SerialDstTopic string

    TeleSrcTopic string  
    TeleDstTopic string   
}

type OnCode struct {
    OnCode1 string
    OnCode2 string
    OnCode3 string
}

type OffCode struct {
    OffCode1 string
    OffCode2 string
    OffCode3 string
}

type LedControlCode struct {
    TokenCode []string
    Cmd string
    StatusCode string
    StatusMsg string
}

type Command struct {
    Led1OnMsgVN LedControlCode
    Led1OnMsgEN LedControlCode
    
    Led1OffMsgVN LedControlCode
    Led1OffMsgEN LedControlCode

    Led2OnMsgVN LedControlCode
    Led2OnMsgEN LedControlCode
    
    Led2OffMsgVN LedControlCode
    Led2OffMsgEN LedControlCode

    DefaultRespMsg string
    ConnectionLostResMsg string
}

type FileConfig struct {
    MqttConfig Mqtt
    CmdConfig Command
}

var cfg FileConfig

func yamlFileHandle() {
    yfile, err := ioutil.ReadFile("config.yaml")

    if err != nil {

      log.Fatal(err)
    }

    err2 := yaml.Unmarshal(yfile, &cfg)

    if err2 != nil {
        fmt.Println("Error file yaml")

      log.Fatal(err2)
    }
}


func sendToSerial(msg string) {
     mqttClientHandleTele.Publish(cfg.MqttConfig.SerialDstTopic, 0, false, msg)
}

func sendToTelegram(msg string) {
     mqttClientHandleSerial.Publish(cfg.MqttConfig.TeleDstTopic, 0, false, msg)
}

func chanSendRes() {
    channel <-"Led control feedback"
}

func chanWaitResAndSendMsgTele(msg string, timeOut time.Duration) {
     select {
        case <-channel:
            sendToTelegram(msg) 
        case <-time.After(timeOut * time.Second):
            sendToTelegram(cfg.CmdConfig.ConnectionLostResMsg)
    }
}

func handleTeleCmd(cmd string) {

    switch cmd {
        case cfg.CmdConfig.Led1OnMsgVN.TokenCode[0], 
             cfg.CmdConfig.Led1OnMsgVN.TokenCode[1]:

             sendToSerial(cfg.CmdConfig.Led1OnMsgVN.Cmd)
             chanWaitResAndSendMsgTele(cfg.CmdConfig.Led1OnMsgVN.StatusMsg, 3)

        case cfg.CmdConfig.Led1OnMsgEN.TokenCode[0], 
             cfg.CmdConfig.Led1OnMsgEN.TokenCode[1]:

             sendToSerial(cfg.CmdConfig.Led1OnMsgEN.Cmd)
             chanWaitResAndSendMsgTele(cfg.CmdConfig.Led1OnMsgEN.StatusMsg, 3)

        case cfg.CmdConfig.Led1OffMsgVN.TokenCode[0], 
             cfg.CmdConfig.Led1OffMsgVN.TokenCode[1]:

             sendToSerial(cfg.CmdConfig.Led1OffMsgVN.Cmd)
             chanWaitResAndSendMsgTele(cfg.CmdConfig.Led1OffMsgVN.StatusMsg, 3)

        case cfg.CmdConfig.Led1OffMsgEN.TokenCode[0], 
             cfg.CmdConfig.Led1OffMsgEN.TokenCode[1]:

             sendToSerial(cfg.CmdConfig.Led1OffMsgEN.Cmd)
             chanWaitResAndSendMsgTele(cfg.CmdConfig.Led1OffMsgEN.StatusMsg, 3)

        case cfg.CmdConfig.Led2OnMsgVN.TokenCode[0], 
             cfg.CmdConfig.Led2OnMsgVN.TokenCode[1]:

             sendToSerial(cfg.CmdConfig.Led2OnMsgVN.Cmd)
             chanWaitResAndSendMsgTele(cfg.CmdConfig.Led2OnMsgVN.StatusMsg, 3)

        case cfg.CmdConfig.Led2OnMsgEN.TokenCode[0], 
             cfg.CmdConfig.Led2OnMsgEN.TokenCode[1]:

             sendToSerial(cfg.CmdConfig.Led2OnMsgEN.Cmd)
             chanWaitResAndSendMsgTele(cfg.CmdConfig.Led2OnMsgEN.StatusMsg, 3)

        case cfg.CmdConfig.Led2OffMsgVN.TokenCode[0], 
             cfg.CmdConfig.Led2OffMsgVN.TokenCode[1]:

             sendToSerial(cfg.CmdConfig.Led2OffMsgVN.Cmd)
             chanWaitResAndSendMsgTele(cfg.CmdConfig.Led2OffMsgVN.StatusMsg, 3)

        case cfg.CmdConfig.Led2OffMsgEN.TokenCode[0], 
             cfg.CmdConfig.Led2OffMsgEN.TokenCode[1]:

             sendToSerial(cfg.CmdConfig.Led2OffMsgEN.Cmd)
             chanWaitResAndSendMsgTele(cfg.CmdConfig.Led2OffMsgEN.StatusMsg, 3)
        default:
            sendToTelegram(cfg.CmdConfig.DefaultRespMsg)
    }
}

func handleSerialCmd(cmd string) {

    switch cmd {
        case cfg.CmdConfig.Led1OnMsgVN.StatusCode:
            chanSendRes()
        case cfg.CmdConfig.Led1OffMsgVN.StatusCode:
            chanSendRes()
    }
}

var messageTelePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {

    teleMsg := string(msg.Payload())

    fmt.Printf("Received message: [%s] from topic: %s\n", teleMsg, msg.Topic())

    handleTeleCmd(teleMsg)
}

var messageSerialPubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {

    serialMsg := string(msg.Payload())

    fmt.Printf("Received message: [%s] from topic: %s\n", serialMsg, msg.Topic())

    handleSerialCmd(serialMsg)
}

func mqttBegin(broker string, messagePubHandler *mqtt.MessageHandler) mqtt.Client {

    var opts *mqtt.ClientOptions = new(mqtt.ClientOptions)

    opts = mqtt.NewClientOptions()
    opts.AddBroker(broker)

    opts.SetDefaultPublishHandler(*messagePubHandler)

    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }
    return client
}

func main() {

    channel = make(chan string, 1)

    yamlFileHandle()

    mqttClientHandleTele = mqttBegin(cfg.MqttConfig.Broker, &messageTelePubHandler)

    mqttClientHandleTele.Subscribe(cfg.MqttConfig.TeleSrcTopic, 1, nil)

    mqttClientHandleSerial = mqttBegin(cfg.MqttConfig.Broker, &messageSerialPubHandler)

    mqttClientHandleSerial.Subscribe(cfg.MqttConfig.SerialSrcTopic, 1, nil)
    
    fmt.Println("Connected")

    for {

        time.Sleep(2 * time.Second)
    }
}