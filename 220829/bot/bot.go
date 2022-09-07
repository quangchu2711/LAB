package main

import (
    "log"
    "fmt"
    "time"
    mqtt "github.com/eclipse/paho.mqtt.golang"
    "github.com/ghodss/yaml"
    "io/ioutil"
    "github.com/agnivade/levenshtein" 
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
    BotQuestion map[string]string
    BotReply map[string]string
    UserReplyYes []string
    UserReplyNo []string   

    TickTimeout time.Duration
}

type FileConfig struct {
    MqttConfig Mqtt
    CmdConfig Command
}

type UserStatus int
const (
    UserSendCmd UserStatus = 0
    UserReply UserStatus = 1
)
var userSta UserStatus = UserSendCmd

var cfg FileConfig
var mqttClientHandleTele mqtt.Client
var mqttClientHandleSerial mqtt.Client

var serialRXChannel chan string 
var listCfgChatCmds string
var cmdListMapVN map[string]*LedControlCode
var cmdListMapEN map[string]*LedControlCode

var scriptLanguage *LedControlCode
var chatCmdLanguage string

var listChatCmds[] string 

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

    var cmdStr string

    for i := 0 ; i < len(controlLedArr); i++ {
        for j := 0; j < len(controlLedArr[i].ChatCmd); j++ {
            cmdListMap[controlLedArr[i].ChatCmd[j]] = &controlLedArr[i]
            listChatCmds = append(listChatCmds, controlLedArr[i].ChatCmd[j])
        }
        // listChatCmds = append(listChatCmds, controlLedArr[i].ChatCmd[0])
        cmdStr += "\n" + controlLedArr[i].ChatCmd[0]
    }
    // fmt.Println("=",cmdStr,"=")
    // fmt.Println(">>", listChatCmds, "<<")

    cfg.CmdConfig.DefaultRespMsg[resMsgUnknowCmd] += cmdStr

    //Add Timeout, UnknowCmd
    for _, controlLed :=range controlLedArr {
        controlLed.ChatResponseMap["Timeout"] =  cfg.CmdConfig.DefaultRespMsg[resMsgTimeout] 
    } 

    return cmdListMap      
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}

func handleTeleCmd(chatCmd string) {

    switch userSta {
    case UserSendCmd:

        msgRes := getTheClosestString(chatCmd, listChatCmds)

        switch msgRes {
        case "NULL":
            helpResVN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdVN"]
            helpResEN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdEN"]
            sendToTelegram(helpResVN)
            sendToTelegram(helpResEN)
        default:
            scriptVN, checkKeyExistsVN := cmdListMapVN[msgRes];
            scriptEN, checkKeyExistsEN := cmdListMapEN[msgRes];

            switch {        
            case checkKeyExistsVN == true:
                questionVN :=  cfg.CmdConfig.BotQuestion["VN"] + msgRes + "?"
                sendToTelegram(questionVN)
                
                scriptLanguage = scriptVN
                chatCmdLanguage = msgRes
                userSta = UserReply

            case checkKeyExistsEN == true:
                questionVN :=  cfg.CmdConfig.BotQuestion["EN"] + msgRes + "?"
                sendToTelegram(questionVN)

                scriptLanguage = scriptEN
                chatCmdLanguage = msgRes
                userSta = UserReply 
            }
        }       

    case UserReply:
        checkReplyYes := stringInSlice(chatCmd, cfg.CmdConfig.UserReplyYes)
        checkReplyNo := stringInSlice(chatCmd, cfg.CmdConfig.UserReplyNo)
        
        switch {
        case checkReplyYes == true:
            handleTeleScript(scriptLanguage, chatCmdLanguage)
            userSta = UserSendCmd
        case checkReplyNo == true:
            sendToTelegram(cfg.CmdConfig.BotReply["No"])
            userSta = UserSendCmd
        default:
            sendToTelegram(cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
            userSta = UserSendCmd
        }
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

func getTheClosestString(str string, strArr[] string) string {
    minNumStep := 7
    resStr := "NULL"

    for i := 0; i < len(strArr); i++ {
        // fmt.Println("KQ: ", levenDis(first, strArr[i]))
        numStep := levenshtein.ComputeDistance(str, strArr[i])
        if numStep < minNumStep {
            minNumStep = numStep
            resStr = strArr[i]
        }
    }
    return resStr
}

func main() {
    // userSta := UserSendCmd

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