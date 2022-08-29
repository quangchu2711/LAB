package main

import (
    "log"
    "fmt"
    "time"
    mqtt "github.com/eclipse/paho.mqtt.golang"
    "github.com/ghodss/yaml"
    "io/ioutil"
    "unicode/utf8"
    "strings"       
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
    fmt.Println(">>", listChatCmds, "<<")

    cfg.CmdConfig.DefaultRespMsg[resMsgUnknowCmd] += cmdStr

    //Add Timeout, UnknowCmd
    for _, controlLed :=range controlLedArr {
        controlLed.ChatResponseMap["Timeout"] =  cfg.CmdConfig.DefaultRespMsg[resMsgTimeout] 
        controlLed.ChatResponseMap["UnknowCmd"] = cfg.CmdConfig.DefaultRespMsg[resMsgUnknowCmd]   
    } 

    return cmdListMap      
}

// func handleTeleCmd(chatCmd string) {  
//     scriptVN, checkKeyExistsVN := cmdListMapVN[chatCmd];

//     scriptEN, checkKeyExistsEN := cmdListMapEN[chatCmd];

//     switch {        
//     case checkKeyExistsVN == true:
//         handleTeleScript(scriptVN, chatCmd)

//     case checkKeyExistsEN == true:
//         handleTeleScript(scriptEN, chatCmd)   
//     default: 
//         helpResVN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdVN"]
//         helpResEN := cfg.CmdConfig.DefaultRespMsg["UnknowCmdEN"]

//         sendToTelegram(helpResVN)
//         sendToTelegram(helpResEN)
//     }
// }

func handleTeleCmd(chatCmd string) {

    switch botChatSta {
    case FirstResponse:

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
                sendToTelegram("Co phải bạn muốn : " + msgRes + "?")
                scriptLanguage = scriptVN
                chatCmdLanguage = msgRes
                botChatSta = SecondResponse

            case checkKeyExistsEN == true:
                sendToTelegram("Do you want: " + msgRes + "?")  
                scriptLanguage = scriptEN
                chatCmdLanguage = msgRes
                botChatSta = SecondResponse 
            }
        }       

    case SecondResponse:
        switch chatCmd {
        case "Đúng", "Yes", "dung", "yes":
            handleTeleScript(scriptLanguage, chatCmdLanguage)
            botChatSta = FirstResponse
        default:
            sendToTelegram(cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
            botChatSta = FirstResponse
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
        numStep := levenDis(str, strArr[i])
        if numStep < minNumStep {
            minNumStep = numStep
            resStr = strArr[i]
        }
    }
    return resStr
}

func levenDis(str1, str2 string) int {  
    switch {
    case len(str1) == 0:
        return utf8.RuneCountInString(str1)
    case len(str2) == 0:
        return utf8.RuneCountInString(str2)
    case str1 == str2:
        return 0
    default:
        /*Add 1 space character at the beginning of the string 
            and remove duplicate whitespace*/
        str1 = " " + trimAllSpace(str1)
        str2 = " " + trimAllSpace(str2)

        /*Use Rune type to store string*/
        s1 := []rune(str1)
        s2 := []rune(str2)
        
        /*Get the length of the string*/
        lenS1 := len(s1) 
        lenS2 := len(s2)
        
        /*Algorithm Levenshtein distance*/
        var disTable [20][20] int

        for i := 0; i < lenS1; i++ {
            disTable[i][0] = i
        }
        for j := 0; j < lenS2; j++ {
            disTable[0][j] = j
        }

        cost := 0
        for i := 1; i < lenS1; i++ {
            for j := 1; j < lenS2; j++ {
                if s1[i] == s2[j] {
                    cost = 0
                }else {
                    cost = 1
                }
                disTable[i][j] = minimum(disTable[i-1][j] + 1, 
                                        disTable[i][j-1] + 1, 
                                        disTable[i-1][j-1] + cost) 
            }
        }
        return disTable[lenS1-1][lenS2-1]
    }
}

func minimum(a, b, c int) int {
    minVal := a
    if minVal >  b {
        minVal = b
    }
    if minVal >  c {
        minVal = c
    }
    return minVal
}

func trimAllSpace(s string) string {
    return strings.Join(strings.Fields(s), " ")
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