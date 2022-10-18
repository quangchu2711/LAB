package main

import (
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
    "sort"
)

type Mqtt struct {
    Broker string
    User string
    Password string
    DeviceSrcTopic string
    DeviceDstTopic string
    TeleSrcTopic string  
    TeleDstTopic string   
}


type LedControlCode struct {
    ChatCmd string
    DeviceCmd string
    ChatResponseMap map[string]string
}

type Command struct {
    ControlLedVN []LedControlCode
    ControlLedEN []LedControlCode
    DefaultRespMsg map[string]string 
    TickTimeout time.Duration
    StringRateThreshold float32
    GroupIDLedDevice string
}

type FileConfig struct {
    MqttConfig Mqtt
    CmdConfig Command
}

type StringSearchResult int
const (
    AlmostSame StringSearchResult = iota
    Same    
    Different
)

type StringCompare struct {
    Data string
    RatePercent float32
}

var cfg FileConfig
var mqttClientHandleTele mqtt.Client
var mqttClientHandleDevice mqtt.Client

var deviceChannel chan string 
var cmdListMapVN map[string]*LedControlCode
var cmdListMapEN map[string]*LedControlCode
var chatCmdlist[] string 

var messageTelePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    teleMsg := string(msg.Payload())
    fmt.Printf("Received message: [%s] from topic: %s\n", teleMsg, msg.Topic())
    groupID, _ := getGroupIdTelegram(msg.Topic())
    handleTeleCmd(groupID, teleMsg)
}

var messageDevicePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    deviceMsg := string(msg.Payload())
    fmt.Printf("Received message: [%s] from topic: %s\n", deviceMsg, msg.Topic())

    deviceTopic := strings.Split(msg.Topic(), "/")
    if deviceTopic[1] == "telegram" {
        writeDataToDeviceChannel(deviceMsg)
    }else {
        groupID := cfg.CmdConfig.GroupIDLedDevice
        deviceResponse := "NULL"
        sendToTelegram(groupID, "HA user just controlled the device")
        for _, controlLed := range cmdListMapEN {
            // fmt.Println(controlLed.DeviceCmd)
            if controlLed.DeviceCmd == deviceMsg {
                fmt.Println(controlLed.DeviceCmd)
                deviceResponse = controlLed.ChatResponseMap[deviceMsg]
            }
        }
        if deviceResponse != "NULL" {
            // fmt.Println(deviceResponse)
            sendToTelegram(groupID, deviceResponse)
        }else {
            sendToTelegram(groupID, cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
        }
    }
}

func cmdListMapInit(controlLedArr []LedControlCode,
                    msgTimeout string) map[string]*LedControlCode {
    cmdListMap := make(map[string]*LedControlCode)

    for i := 0 ; i < len(controlLedArr); i++ {
        cmdListMap[controlLedArr[i].ChatCmd] = &controlLedArr[i]
        chatCmdlist = append(chatCmdlist, controlLedArr[i].ChatCmd)
    }
    for _, controlLed :=range controlLedArr {
        controlLed.ChatResponseMap["Timeout"] =  cfg.CmdConfig.DefaultRespMsg[msgTimeout] 
    } 

    return cmdListMap      
}

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

func getNormStr(inputStr string) string {
        lowerStr := strings.ToLower(inputStr)

        t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
        normStr, _, _ := transform.String(t, lowerStr)

        return normStr      
}

func getCommandSearchStatus(rateOfChange float32) StringSearchResult {
    cmdSearchRes := Different
    strRateThres := cfg.CmdConfig.StringRateThreshold

    if rateOfChange == 100.0 {
        cmdSearchRes = Same
    }else if rateOfChange >= strRateThres {
        cmdSearchRes = AlmostSame
    }

    return cmdSearchRes
}

func handleTeleScript(script *LedControlCode, groupID string) {
    sendToDevice(script.DeviceCmd)
    resRxChan := readDataFromDeviceChannel(cfg.CmdConfig.TickTimeout)
    resDataTele, checkKeyExists := script.ChatResponseMap[resRxChan];

    switch checkKeyExists {
    case true:
        sendToTelegram(groupID, resDataTele)
    default:
        sendToTelegram(groupID, cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
    }
} 

func handleTeleCmd(groupID string, cmd string) {
    chatCmd := removeElementAfterBracket(cmd)  
    chatCmdArr := sortCommandCompareArrayDescending(chatCmd, chatCmdlist)
    maxRatePercent :=  chatCmdArr[0].RatePercent
    cmdSearchRes := getCommandSearchStatus(maxRatePercent)

    switch cmdSearchRes {
    case Different:
        sendHelpResponseToTelegramUser(groupID)
    case AlmostSame:
        sendSuggestResponseToTelegramUser(groupID, chatCmdArr)
    case Same:
        scriptVN, checkKeyExistsVN := cmdListMapVN[chatCmdArr[0].Data]
        scriptEN, _ := cmdListMapEN[chatCmd];
        if checkKeyExistsVN == true {
            fmt.Println("Vietnamese")
            handleTeleScript(scriptVN, groupID)
        }else {
            fmt.Println("English")
            handleTeleScript(scriptEN, groupID)          
        }            
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

func readDataFromDeviceChannel(timeOut time.Duration) string {
    var msg string

    select {
    case msg =  <-deviceChannel:
        return msg;
    case <-time.After(timeOut * time.Second):
        msg = "Timeout"
        return msg
    }
}

func writeDataToDeviceChannel(cmd string) {
    deviceChannel <-cmd
}

func removeElementAfterBracket(strInput string) string {
    var strOutput string
    index := strings.Index(strInput, "(")
    if index == -1 {
        strOutput = strInput
    }else {
        strOutput = strInput[0:(index-1)]        
    }
    return strOutput
}

func sendHelpResponseToTelegramUser(groupID string) {
    var cmdKeyboard string
    textVN := cfg.CmdConfig.DefaultRespMsg["ResponseHelpVN"]
    textEN := cfg.CmdConfig.DefaultRespMsg["ResponseHelpEN"]
    textKeyboard := textVN + "\n" + textEN
    
    for _, cmd := range chatCmdlist {
        cmdKeyboard += "/" + cmd
    }

    sendToTelegram(groupID, "[" + textKeyboard + cmdKeyboard + "]")   
}

func sendSuggestResponseToTelegramUser (groupID string, chatCmdArr []StringCompare) {
    var textKeyboard string
    var cmdKeyboard string 
    chatCmd := chatCmdArr[0].Data

    _, checkKeyExistsVN := cmdListMapVN[chatCmd]
    if checkKeyExistsVN == true {
        textKeyboard = cfg.CmdConfig.DefaultRespMsg["SuggestVN"]
    }else {
        textKeyboard = cfg.CmdConfig.DefaultRespMsg["SuggestEN"]        
    }
    for i := 0; i < 3; i++ {
        rateValue := fmt.Sprintf("%.2f", chatCmdArr[i].RatePercent)
        cmdKeyboard += "/" + chatCmdArr[i].Data + " (" + rateValue + " %" + ")" 
    }

    sendToTelegram(groupID, "[" + textKeyboard + cmdKeyboard + "]")    
}

func sendToDevice(msg string) {
    mqttClientHandleTele.Publish(cfg.MqttConfig.DeviceDstTopic, 0, false, msg)
}

func sendToTelegram(groupID string, msg string) {
    teleDstTopic := strings.Replace(cfg.MqttConfig.TeleDstTopic, "GroupID", groupID, 1)
    mqttClientHandleDevice.Publish(teleDstTopic, 0, false, msg)
}

func sortChatCmdlist () []string{
    var cmdList []string
    halfLength := len(chatCmdlist) / 2

    for i := 0; i < halfLength; i++ {
        cmdList = append(cmdList, chatCmdlist[i])        
        cmdList = append(cmdList, chatCmdlist[i + halfLength])        
    }

    return cmdList
}

func sortCommandCompareArrayDescending(str string, strArr[] string) ([]StringCompare) {
    var strCmp StringCompare
    var chatCmdArr []StringCompare

    for i := 0; i < len(strArr); i++ {
        str1 := getNormStr(str)
        str2 := getNormStr(strArr[i])
        numTranStep := levenshtein.ComputeDistance(str1, str2)
        lenStr2 := len(str2)
        strCmp.Data = strArr[i]
        if numTranStep > lenStr2 {
            strCmp.RatePercent = 0.0
        }else {
            strCmp.RatePercent = 100.0 - (float32(numTranStep) / float32(len(str2)) * 100.0)
        }
        chatCmdArr = append(chatCmdArr, strCmp)
        // fmt.Printf("[%s - %d - %.2f]\n", getNormStr(strArr[i]), numTranStep, strCmp.RatePercent)
    }
    sort.SliceStable(chatCmdArr, func(i, j int) bool {
        return chatCmdArr[i].RatePercent > chatCmdArr[j].RatePercent
    })

    return chatCmdArr
}

func yamlFileHandle() {
    yfile, _ := ioutil.ReadFile("config.yaml")
    yaml.Unmarshal(yfile, &cfg)
}

func main() {
    yamlFileHandle()
    deviceChannel = make(chan string, 1)
    cmdListMapVN = cmdListMapInit(cfg.CmdConfig.ControlLedVN, "TimeoutVN")
    cmdListMapEN = cmdListMapInit(cfg.CmdConfig.ControlLedEN, "TimeoutEN")
    chatCmdlist = sortChatCmdlist()
    mqttClientHandleTele = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageTelePubHandler)
    mqttClientHandleTele.Subscribe(cfg.MqttConfig.TeleSrcTopic, 1, nil)
    mqttClientHandleDevice = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageDevicePubHandler)
    mqttClientHandleDevice.Subscribe(cfg.MqttConfig.DeviceSrcTopic, 1, nil)
    fmt.Println("Connected")

    for {

        time.Sleep(2 * time.Second)
    }
}