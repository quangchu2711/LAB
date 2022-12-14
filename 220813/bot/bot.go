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

var  serialRXChannel chan string

type Mqtt struct {
    Broker string

    SerialSrcTopic string
    SerialDstTopic string

    TeleSrcTopic string  
    TeleDstTopic string   
}


type LedControlCode struct {
    TokenCode []string
    Cmd string
    ResponseMap map[string]string
}

type Command struct {
    ControlLed []LedControlCode

    DefaultRespMsg string
    TimeoutRespMsgVN string
    TimeoutRespMsgEN string
    TickTimeout time.Duration
}

type FileConfig struct {
    MqttConfig Mqtt
    CmdConfig Command
}

var cfg FileConfig
var cmdListMap map[string]*LedControlCode

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

func readSerialRXChannel(timeOut time.Duration) string {
    var msg string

    select {
        case msg =  <-serialRXChannel:
            return msg;
        case <-time.After(timeOut * time.Second):
            msg = "TIMEOUT"
            return msg
    }
}


func handleTeleScript(script *LedControlCode) {

    sendToSerial(script.Cmd)

    resRxChan := readSerialRXChannel(cfg.CmdConfig.TickTimeout)

    resDataTele, checkKeyExists := script.ResponseMap[resRxChan];

    switch checkKeyExists {
        case true:
            sendToTelegram(resDataTele)

        default:
            sendToTelegram(script.ResponseMap["ERROR CMD"])

    }
} 

func cmdListMapInit(cmdListMap map[string]*LedControlCode) {
    script := cfg.CmdConfig.ControlLed
    for i := 0 ; i < len(script); i++ {
        fmt.Println("======================") 
        //fmt.Println(script)
        for j := 0; j < len(script[i].TokenCode); j++ {
            cmdListMap[script[i].TokenCode[j]] = &script[i]  
        }
        // // for _, msgTele := range script.TokenCode {
        // //     fmt.Println("-------------------") 
        // //     fmt.Println(msgTele, script) 
        // //     var pCopy *LedControlCode
        // //     pCopy =  &script   
        // //     cmdListMap[msgTele] = pCopy
        // // } 
    } 
    fmt.Println("+++++++++++++++++++++\n") 

    for key, value := range cmdListMap {
        fmt.Println(key, value)
    }  
}

func handleTeleCmd(tokenCode string) {

    script, checkKeyExists := cmdListMap[tokenCode];

    switch checkKeyExists {
        case true:
            handleTeleScript(script)

        default:
            sendToTelegram(cfg.CmdConfig.DefaultRespMsg)

    }

}

func handleSerialCmd(cmd string) {
    serialRXChannel <-cmd
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

    serialRXChannel = make(chan string, 1)

    yamlFileHandle()

    cmdListMap = map[string]*LedControlCode{}

    cmdListMapInit(cmdListMap)

    mqttClientHandleTele = mqttBegin(cfg.MqttConfig.Broker, &messageTelePubHandler)

    mqttClientHandleTele.Subscribe(cfg.MqttConfig.TeleSrcTopic, 1, nil)

    mqttClientHandleSerial = mqttBegin(cfg.MqttConfig.Broker, &messageSerialPubHandler)

    mqttClientHandleSerial.Subscribe(cfg.MqttConfig.SerialSrcTopic, 1, nil)
    
    fmt.Println("Connected")

    for {

        time.Sleep(2 * time.Second)
    }
}