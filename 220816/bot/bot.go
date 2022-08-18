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
    ControlLedVN []LedControlCode
    ControlLedEN []LedControlCode

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
            //sendToTelegram("TIMEOUT")

    }
} 

func cmdListMapInit(controlLedArr []LedControlCode,
                    cmdListMap map[string]*LedControlCode) {
    // for _, script := range *controlLedArr {
    //     for _, msgTele := range script.TokenCode {
    //         cmdListMap[msgTele] = &script
    //     } 
    // }

        for i := 0 ; i < len(controlLedArr); i++ {
            for j := 0; j < len(controlLedArr[i].TokenCode); j++ {
                cmdListMap[controlLedArr[i].TokenCode[j]] = &controlLedArr[i]  
        }  
    }      
}

func handleTeleCmd(tokenCode string) {

    cmdListMapVN := make(map[string]*LedControlCode)
    cmdListMapInit(cfg.CmdConfig.ControlLedVN, cmdListMapVN)

    cmdListMapEN := make(map[string]*LedControlCode)
    cmdListMapInit(cfg.CmdConfig.ControlLedEN, cmdListMapEN)

    // var checkKeyExistsVN bool
    // var scriptVN *LedControlCode
    // scriptVN = &LedControlCode{}

    scriptVN, checkKeyExistsVN := cmdListMapVN[tokenCode];

    scriptEN, checkKeyExistsEN := cmdListMapEN[tokenCode];

    if checkKeyExistsVN == true {
        scriptVN.ResponseMap["TIMEOUT"] = cfg.CmdConfig.TimeoutRespMsgVN
        handleTeleScript(scriptVN)
    }else if checkKeyExistsEN == true {
        scriptEN.ResponseMap["TIMEOUT"] = cfg.CmdConfig.TimeoutRespMsgEN
        handleTeleScript(scriptEN)           
    }else {
        sendToTelegram(cfg.CmdConfig.DefaultRespMsg)
    }


    // //switch checkKeyExistsVN || checkKeyExistsEN {
    // switch checkKeyExistsVN  {        
    // case checkKeyExistsVN == true:
    //     scriptVN.ResponseMap["TIMEOUT"] = cfg.CmdConfig.TimeoutRespMsgVN
    //     handleTeleScript(scriptVN)

    // // case checkKeyExistsEN == true:
    // //     scriptEN.ResponseMap["TIMEOUT"] = cfg.CmdConfig.TimeoutRespMsgEN
    // //     handleTeleScript(scriptEN)   
    // default:
    //     // scriptVN.ResponseMap["Default"] = "TRY AGAIN"
    //     // scriptEN.ResponseMap["Default"] = "TRY AGAIN"
        
    //     // fmt.Println(scriptVN, scriptEN)
    //     sendToTelegram(cfg.CmdConfig.DefaultRespMsg)
    // }

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