package main

import (
    "log"
    "fmt"
    "time"
    mqtt "github.com/eclipse/paho.mqtt.golang"
    "github.com/ghodss/yaml"
    "io/ioutil"   
)

type Mqtt struct {
    Broker string

    SerialSrcTopic string
    SerialDstTopic string

    TeleSrcTopic string  
    TeleDstTopic string   
}


type LedControlCode struct {
    ChatCmd []string
    DeviceCmd string
    ChatResponseMap map[string]string
}

type Command struct {
    ControlLedVN []LedControlCode
    ControlLedEN []LedControlCode

    DefaultRespMsg map[string]string

    TickTimeout time.Duration
}

type FileConfig struct {
    MqttConfig Mqtt
    CmdConfig Command
}

var cfg FileConfig

var mqttClientHandleTele mqtt.Client
var mqttClientHandleSerial mqtt.Client

var serialRXChannel chan string 
// var listCfgChatCmds string
var cmdListMapVN map[string]*LedControlCode
var cmdListMapEN map[string]*LedControlCode

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
            msg = "Timeout"
            return msg
    }
}

func handleTeleScript(script *LedControlCode, chatCmd string) {

    sendToSerial(script.DeviceCmd)

    resRxChan := readSerialRXChannel(cfg.CmdConfig.TickTimeout)

    resDataTele, checkKeyExists := script.ChatResponseMap[resRxChan];

    switch checkKeyExists {
        case true:
            sendToTelegram(resDataTele)

        default:
            sendToTelegram(script.ChatResponseMap["UnknowCmd"])
    }
} 

func cmdListMapInit(controlLedArr []LedControlCode,
                    cmdListMap map[string]*LedControlCode,
                    resMsgTimeout string,
                    resMsgUnknowCmd string) {

        var listChatCmds string 

        for i := 0 ; i < len(controlLedArr); i++ {
            for j := 0; j < len(controlLedArr[i].ChatCmd); j++ {
                cmdListMap[controlLedArr[i].ChatCmd[j]] = &controlLedArr[i]
                listChatCmds += "\n" + controlLedArr[i].ChatCmd[j] 
        }
        // listCfgChatCmds += listChatCmds


        //Add Timeout, UnknowCmd
        for _, controlLed :=range controlLedArr {
            controlLed.ChatResponseMap["Timeout"] = resMsgTimeout
            controlLed.ChatResponseMap["UnknowCmd"] = resMsgUnknowCmd + listChatCmds           
        }         

    }      
}

func handleTeleCmd(chatCmd string) {

    scriptVN, checkKeyExistsVN := cmdListMapVN[chatCmd];

    scriptEN, checkKeyExistsEN := cmdListMapEN[chatCmd];

    switch {        
    case checkKeyExistsVN == true:
        handleTeleScript(scriptVN, chatCmd)

    case checkKeyExistsEN == true:
        handleTeleScript(scriptEN, chatCmd)   
    default: 
        // sendToTelegram(cfg.CmdConfig.DefaultRespMsg["ErrorCmd"] + "\n" + listCfgChatCmds)
        sendToTelegram(cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
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

    cmdListMapVN = make(map[string]*LedControlCode)
    cmdListMapEN = make(map[string]*LedControlCode)
    
    cmdListMapInit(cfg.CmdConfig.ControlLedVN, 
                   cmdListMapVN, 
                   cfg.CmdConfig.DefaultRespMsg["TimeoutVN"],
                   cfg.CmdConfig.DefaultRespMsg["UnknowCmdVN"])

    cmdListMapInit(cfg.CmdConfig.ControlLedEN, 
                    cmdListMapEN, 
                    cfg.CmdConfig.DefaultRespMsg["TimeoutEN"],
                    cfg.CmdConfig.DefaultRespMsg["UnknowCmdEN"])


    mqttClientHandleTele = mqttBegin(cfg.MqttConfig.Broker, &messageTelePubHandler)

    mqttClientHandleTele.Subscribe(cfg.MqttConfig.TeleSrcTopic, 1, nil)

    mqttClientHandleSerial = mqttBegin(cfg.MqttConfig.Broker, &messageSerialPubHandler)

    mqttClientHandleSerial.Subscribe(cfg.MqttConfig.SerialSrcTopic, 1, nil)
    
    fmt.Println("Connected")

    for {

        time.Sleep(2 * time.Second)
    }
}