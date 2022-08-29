package main

import (
    "log"
    "fmt"
    "time"
    mqtt "github.com/eclipse/paho.mqtt.golang"
    "github.com/ghodss/yaml"
    "io/ioutil"  
    // "github.com/hexops/valast"  
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
    ChatCmd []string
    DeviceCmd string
    ChatResponseMap map[string]string
    ChatResponseUNKNOWNCMD []string
    ChatResponseTIMEOUT string
}

type Command struct {
    ControlLedVN []LedControlCode
    ControlLedEN []LedControlCode

    DefaultRespMsg string
    ErrorCmdVN string
    ErrorCmdEN string
    TimeoutRespMsgVN string
    TimeoutRespMsgEN string

    TickTimeout time.Duration
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

// func createResponseUnknowCmd () string {
//     var listChatCmds string 
//     for _, str := range script.ChatCmd {
//         listChatCmds += "\n" + str
//     }
// }

func handleTeleScript(script *LedControlCode, chatCmd string) {

    sendToSerial(script.DeviceCmd)

    resRxChan := readSerialRXChannel(cfg.CmdConfig.TickTimeout)

    resDataTele, checkKeyExists := script.ChatResponseMap[resRxChan];

    switch checkKeyExists {
        case true:
            sendToTelegram(resDataTele)

        default:
            //sendToTelegram(script.ChatResponseMap["ERROR CMD"])

            //listCfgCmds  := fmt.Sprintf("%+v", *script)
            var listChatCmds string 
            for _, str := range script.ChatCmd {
                if str != chatCmd {
                    listChatCmds += "\n" + str
                }else {
                    continue
                }
            }

            //listCfgCmds  := valast.String(*script)
            fmt.Println(listChatCmds) 
            sendToTelegram(script.ChatResponseMap["ERROR CMD"] + listChatCmds)
    }
} 

func cmdListMapInit(controlLedArr []LedControlCode,
                    cmdListMap map[string]*LedControlCode) {

        for i := 0 ; i < len(controlLedArr); i++ {
            for j := 0; j < len(controlLedArr[i].ChatCmd); j++ {
                cmdListMap[controlLedArr[i].ChatCmd[j]] = &controlLedArr[i]  
        }  
    }      
}

func handleTeleCmd(chatCmd string) {

    cmdListMapVN := make(map[string]*LedControlCode)
    cmdListMapEN := make(map[string]*LedControlCode)
    
    cmdListMapInit(cfg.CmdConfig.ControlLedVN, cmdListMapVN)
    cmdListMapInit(cfg.CmdConfig.ControlLedEN, cmdListMapEN)

    scriptVN, checkKeyExistsVN := cmdListMapVN[chatCmd];

    scriptEN, checkKeyExistsEN := cmdListMapEN[chatCmd];

    switch {        
    case checkKeyExistsVN == true:
        scriptVN.ChatResponseMap["TIMEOUT"] = cfg.CmdConfig.TimeoutRespMsgVN
        scriptVN.ChatResponseMap["ERROR CMD"] = cfg.CmdConfig.ErrorCmdVN 

        handleTeleScript(scriptVN, chatCmd)

    case checkKeyExistsEN == true:
        scriptEN.ChatResponseMap["TIMEOUT"] = cfg.CmdConfig.TimeoutRespMsgEN
        scriptEN.ChatResponseMap["ERROR CMD"] = cfg.CmdConfig.ErrorCmdEN 

        handleTeleScript(scriptEN, chatCmd)   
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

    mqttClientHandleTele = mqttBegin(cfg.MqttConfig.Broker, &messageTelePubHandler)

    mqttClientHandleTele.Subscribe(cfg.MqttConfig.TeleSrcTopic, 1, nil)

    mqttClientHandleSerial = mqttBegin(cfg.MqttConfig.Broker, &messageSerialPubHandler)

    mqttClientHandleSerial.Subscribe(cfg.MqttConfig.SerialSrcTopic, 1, nil)
    
    fmt.Println("Connected")

    for {

        time.Sleep(2 * time.Second)
    }
}