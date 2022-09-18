package main

import (
    "log"
    "fmt"
    "time"
    mqtt "github.com/eclipse/paho.mqtt.golang"
    "github.com/ghodss/yaml"
    "io/ioutil"
    "github.com/agnivade/levenshtein" 
    "strings"
    "errors"
    "golang.org/x/text/runes"
    "golang.org/x/text/transform"
    "golang.org/x/text/unicode/norm"
    "unicode"
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
    // BotQuestion map[string]string
    // BotReply map[string]string
    // UserReplyYes []string
    // UserReplyNo []string   

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

// var scriptLanguage *LedControlCode
// var chatCmdLanguage string

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

func sendToTelegram(groupID string, msg string) {
    // fmt.Println(groupID)
    teleDstTopic := strings.Replace(cfg.MqttConfig.TeleDstTopic, "GroupID", groupID, 1)
    // fmt.Println(teleDstTopic)
    mqttClientHandleSerial.Publish(teleDstTopic, 0, false, msg)
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

func handleTeleScript(script *LedControlCode, groupID string, chatCmd string) {

    sendToSerial(script.DeviceCmd)

    resRxChan := readSerialRXChannel(cfg.CmdConfig.TickTimeout)

    resDataTele, checkKeyExists := script.ChatResponseMap[resRxChan];

    switch checkKeyExists {
        case true:
            sendToTelegram(groupID, resDataTele)

        default:
            sendToTelegram(groupID, cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
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
        cmdStr += "\n" + controlLedArr[i].ChatCmd[0]
    }

    cfg.CmdConfig.DefaultRespMsg[resMsgUnknowCmd] += cmdStr

    for _, controlLed :=range controlLedArr {
        controlLed.ChatResponseMap["Timeout"] =  cfg.CmdConfig.DefaultRespMsg[resMsgTimeout] 
    } 

    return cmdListMap      
}

func handleTeleCmd(groupID string, chatCmd string) {  
    //normStr := getNormStr(chatCmd)

    resStr := getClosestStr(chatCmd, listChatCmds)

    fmt.Printf("chat: %s  -> norm: %s\n", chatCmd, resStr)

    scriptVN, checkKeyExistsVN := cmdListMapVN[chatCmd];

    scriptEN, checkKeyExistsEN := cmdListMapEN[chatCmd];

    switch {        
    case checkKeyExistsVN == true:
        handleTeleScript(scriptVN, groupID, chatCmd)

    case checkKeyExistsEN == true:
        handleTeleScript(scriptEN, groupID, chatCmd)   
    default: 
        helpResVN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdVN"]
        helpResEN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdEN"]

        sendToTelegram(groupID, helpResVN)
        sendToTelegram(groupID, helpResEN)
    }
}

// func stringInSlice(a string, list []string) bool {
//     for _, b := range list {
//         if b == a {
//             return true
//         }
//     }
//     return false
// }

// func handleTeleCmd(chatCmd string, groupID string) {

//     switch userSta {
//     case UserSendCmd:

//         msgRes := getClosestStr(chatCmd, listChatCmds)

//         switch msgRes {
//         case "NULL":
//             helpResVN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdVN"]
//             helpResEN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdEN"]
//             sendToTelegram(groupID, helpResVN)
//             sendToTelegram(groupID, helpResEN)
//         default:
//             scriptVN, checkKeyExistsVN := cmdListMapVN[msgRes];
//             scriptEN, checkKeyExistsEN := cmdListMapEN[msgRes];

//             switch {        
//             case checkKeyExistsVN == true:
//                 questionVN :=  cfg.CmdConfig.BotQuestion["VN"] + msgRes + "?"
//                 sendToTelegram(groupID, questionVN)
                
//                 scriptLanguage = scriptVN
//                 chatCmdLanguage = msgRes
//                 userSta = UserReply

//             case checkKeyExistsEN == true:
//                 questionVN :=  cfg.CmdConfig.BotQuestion["EN"] + msgRes + "?"
//                 sendToTelegram(groupID, questionVN)

//                 scriptLanguage = scriptEN
//                 chatCmdLanguage = msgRes
//                 userSta = UserReply 
//             }
//         }       

//     case UserReply:
//         checkReplyYes := stringInSlice(chatCmd, cfg.CmdConfig.UserReplyYes)
//         checkReplyNo := stringInSlice(chatCmd, cfg.CmdConfig.UserReplyNo)
        
//         switch {
//         case checkReplyYes == true:
//             handleTeleScript(scriptLanguage, groupID, chatCmdLanguage)
//             userSta = UserSendCmd
//         case checkReplyNo == true:
//             sendToTelegram(groupID, cfg.CmdConfig.BotReply["No"])
//             userSta = UserSendCmd
//         default:
//             sendToTelegram(groupID, cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
//             userSta = UserSendCmd
//         }
//     }
// }

func getGroupIdTelegram (topic string) (string, error) {
    topicItem := strings.Split(topic, "/")
    err := "Incorrect topic format"

    if topicItem[0] != "Telegram" {
        return "0", errors.New(err)
    }else {
        if topicItem[2] != "Rx" {
            return "0", errors.New(err)
        }else {
            groupID := topicItem[1]
            return groupID, nil
        }
    }
}

func handleSerialCmd(cmd string) {
    serialRXChannel <-cmd
}

func getNormStr(inputStr string) string {

        lowerStr := strings.ToLower(inputStr)

        t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
        normStr, _, _ := transform.String(t, lowerStr)

        return normStr      
}

func getClosestStr(str string, strArr[] string) string {
    minNumStep := 7
    resStr := "NULL"

    for i := 0; i < len(strArr); i++ {
        numStep := levenshtein.ComputeDistance(getNormStr(str), getNormStr(strArr[i]))
        fmt.Printf("[%s - %d]\n", getNormStr(strArr[i]), numStep)
        if numStep < minNumStep {
            minNumStep = numStep
            resStr = strArr[i]
        }
    }
    return resStr
}

var messageTelePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {

    teleMsg := string(msg.Payload())

    fmt.Printf("Received message: [%s] from topic: %s\n", teleMsg, msg.Topic())

    groupID, err := getGroupIdTelegram(msg.Topic())

    if err != nil {

      log.Fatal(err)
    }

    handleTeleCmd(groupID, teleMsg)

    // topicItem := strings.Split(msg.Topic(), "/")


    // if topicItem[0] == "Telegram" && topicItem[2] == "Rx" {
    //     groupID := topicItem[1]
    //     handleTeleCmd(groupID, teleMsg)
    // }
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