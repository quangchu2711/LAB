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

type BotChatResState int
const (
    FirstResponse BotChatResState = 0
    SecondResponse BotChatResState = 1
)
var botChatSta BotChatResState = FirstResponse

var cfg FileConfig
var mqttClientHandleTele mqtt.Client
var mqttClientHandleSerial mqtt.Client

var serialRXChannel chan string 
var listCfgChatCmds string
var cmdListMapVN map[string]*LedControlCode
var cmdListMapEN map[string]*LedControlCode
var scriptLanguage *LedControlCode
var chatCmdLanguage string

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
            sendToTelegram(cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
    }
} 

func cmdListMapInit(controlLedArr []LedControlCode,
                    resMsgTimeout string,
                    resMsgUnknowCmd string) map[string]*LedControlCode {

    cmdListMap := make(map[string]*LedControlCode)

    var listChatCmds string 
    for i := 0 ; i < len(controlLedArr); i++ {
        for j := 0; j < len(controlLedArr[i].ChatCmd); j++ {
            cmdListMap[controlLedArr[i].ChatCmd[j]] = &controlLedArr[i]
        }
        listChatCmds += "\n" + controlLedArr[i].ChatCmd[0] 
    }
    // listCfgChatCmds += listChatCmds

    cfg.CmdConfig.DefaultRespMsg[resMsgUnknowCmd] += listChatCmds

    //Add Timeout, UnknowCmd
    for _, controlLed :=range controlLedArr {
        controlLed.ChatResponseMap["Timeout"] =  cfg.CmdConfig.DefaultRespMsg[resMsgTimeout] 
        controlLed.ChatResponseMap["UnknowCmd"] = cfg.CmdConfig.DefaultRespMsg[resMsgUnknowCmd]        
    } 

    return cmdListMap      
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
        //sendToTelegram(cfg.CmdConfig.DefaultRespMsg["UnknowCmdVN"] + listChatCmds)
        //sendToTelegram("Yêu cầu không rõ, bạn có thể thử: " + listCfgChatCmds)
        helpResVN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdVN"]
        helpResEN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdEN"]

        sendToTelegram(helpResVN)
        sendToTelegram(helpResEN)
    }
}

// func handleTeleCmd(chatCmd string) {

//     switch botChatSta {
//     case FirstResponse:
        
//         scriptVN, checkKeyExistsVN := cmdListMapVN[chatCmd];
//         scriptEN, checkKeyExistsEN := cmdListMapEN[chatCmd];

//         switch {        
//         case checkKeyExistsVN == true:
//             sendToTelegram("Co phải bạn muốn : " + chatCmd + "?")
//             scriptLanguage = scriptVN
//             chatCmdLanguage = chatCmd
//             botChatSta = SecondResponse

//         case checkKeyExistsEN == true:
//             sendToTelegram("Do you want: " + chatCmd + "?")  
//             scriptLanguage = scriptEN
//             chatCmdLanguage = chatCmd
//             botChatSta = SecondResponse

//         default: 
//             helpResVN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdVN"]
//             helpResEN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdEN"]
//             sendToTelegram(helpResVN)
//             sendToTelegram(helpResEN)
//         }

//     case SecondResponse:
//         switch chatCmd {
//         case "Đúng", "Yes", "dung", "yes":
//             handleTeleScript(scriptLanguage, chatCmdLanguage)
//             botChatSta = FirstResponse
//         default:
//             sendToTelegram(cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
//             botChatSta = FirstResponse
//         }
//     }
// }

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
    // botChatSta := FirstResponse

    serialRXChannel = make(chan string, 1)

    yamlFileHandle()
    
    cmdListMapVN = cmdListMapInit(cfg.CmdConfig.ControlLedVN, "TimeoutVN", "UnknowCmdVN")

    cmdListMapEN = cmdListMapInit(cfg.CmdConfig.ControlLedEN, "TimeoutEN", "UnknowCmdEN")

    mqttClientHandleTele = mqttBegin(cfg.MqttConfig.Broker, &messageTelePubHandler)

    mqttClientHandleTele.Subscribe(cfg.MqttConfig.TeleSrcTopic, 1, nil)

    mqttClientHandleSerial = mqttBegin(cfg.MqttConfig.Broker, &messageSerialPubHandler)

    mqttClientHandleSerial.Subscribe(cfg.MqttConfig.SerialSrcTopic, 1, nil)
    
    fmt.Println("Connected")

    for {

        time.Sleep(2 * time.Second)
    }
}