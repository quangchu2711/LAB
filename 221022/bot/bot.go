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
    LedDeviceSrcTopic string
    LedDeviceDstTopic string
    TeleSrcTopic string  
    TeleDstTopic string   
    SensorDeviceSrcTopic string
    SensorDeviceDstTopic string
}


type DeviceControlCode struct {
    ChatCmd string
    DeviceCmd string
    ChatResponseMap map[string]string
}

type Command struct {
    ControlDeviceVN []DeviceControlCode
    ControlDeviceEN []DeviceControlCode
    DefaultRespMsg map[string]string 
    TickTimeout time.Duration
    StringRateThreshold float32
    GroupIDLedDevice string
    BotPassword string
}

type DeviceName int
const (
    LedDevice DeviceName = iota
    SensorDeivce
)

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
var chatCmdlist[] string 
var cmdListMapEN map[string]*DeviceControlCode
var cmdListMapVN map[string]*DeviceControlCode
var deviceChannel chan string 
var groupIDBotStatusMap map[string]string
var telegramDeviceClient mqtt.Client
var ledDeviceClient mqtt.Client
var sensorDeviceClient mqtt.Client

var messageTelePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    userMsg := string(msg.Payload())
    fmt.Printf("Received message: [%s] from topic: %s\n", userMsg, msg.Topic())
    groupID, _ := getGroupIdTelegram(msg.Topic())
    botHandle(groupID, userMsg)
}

var messageLedDevicePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    deviceMsg := string(msg.Payload())
    fmt.Printf("Received message: [%s] from topic: %s\n", deviceMsg, msg.Topic())

    deviceTopic := strings.Split(msg.Topic(), "/")
    if deviceTopic[2] == "telegram" {
        writeDataToDeviceChannel(deviceMsg)
    }else {
        groupID := cfg.CmdConfig.GroupIDLedDevice
        deviceResponse := "NULL"
        sendToTelegram(groupID, "HA user just controlled the device")
        for _, controlLed := range cmdListMapEN {
            if controlLed.DeviceCmd == deviceMsg {
                fmt.Println(controlLed.DeviceCmd)
                deviceResponse = controlLed.ChatResponseMap[deviceMsg]
            }
        }
        if deviceResponse != "NULL" {
            sendToTelegram(groupID, deviceResponse)
        }else {
            sendToTelegram(groupID, cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
        }
    }
}

var messageSensorDevicePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    deviceMsg := string(msg.Payload())
    fmt.Printf("Received message: [%s] from topic: %s\n", deviceMsg, msg.Topic())
    fmt.Println("Sensor")
    writeDataToDeviceChannel(deviceMsg)
}

func botHandle(groupID string, userMsg string) {
    botStatus, checkGroupID := groupIDBotStatusMap[groupID]
    switch checkGroupID {
    case true:
        handleUserEnteredPasswordForBot(groupID, userMsg, botStatus)
    default:
        handleUserNeverEnteredPasswordBot(groupID, userMsg)
    }
}

func checkDeviceName(script *DeviceControlCode) DeviceName {
    var deviceName DeviceName
    _, checkKeyExists := script.ChatResponseMap["Sensor"]
    if checkKeyExists == true {
        deviceName = SensorDeivce
    }else {
        deviceName = LedDevice
    }
    return deviceName
}

func cmdListMapInit(controlLedArr []DeviceControlCode,
                    msgTimeout string) map[string]*DeviceControlCode {
    cmdListMap := make(map[string]*DeviceControlCode)

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

func handleBlockStatusOfBot(groupID string, userMsg string) {
    splitUserMsg := strings.Split(userMsg, " ")    
    fmt.Printf("[Bot block] GroupID: %s\n", groupID)
    if len(splitUserMsg) == 1 {
        if splitUserMsg[0] == "start" || splitUserMsg[0] == "/start"{
            groupIDBotStatusMap[groupID] = "Running"
            sendToTelegram(groupID, "Bot is back to work")                    
        }
    }else if len(splitUserMsg) == 2 {
        if splitUserMsg[0] == "start" || splitUserMsg[0] == "/start" {
            if splitUserMsg[1] == cfg.CmdConfig.BotPassword {
                groupIDBotStatusMap[groupID] = "Running"
                sendToTelegram(groupID, "Enter password successfully. Bot is back to work")
            }else {
                sendToTelegram(groupID, "Password input failed, please try again")                
            }
        }
    }
}


func handleUserEnteredPasswordForBot(groupID string, userMsg string, botStatus string) {
    switch botStatus {
    case "Running":
        handleRunningStatusOfBot(groupID, userMsg)
    case "Block":
        handleBlockStatusOfBot(groupID, userMsg)
    }
}

func handleUserNeverEnteredPasswordBot(groupID string, userMsg string) {
    splitUserMsg := strings.Split(userMsg, " ")
    if len(splitUserMsg) == 1 {
        if splitUserMsg[0] == "start" || splitUserMsg[0] == "/start"{                
            sendToTelegram(groupID, "Your group has not entered a password for the bot. Enter the password in the format: \nstart + password or /start + password")
        }
    }else if len(splitUserMsg) == 2 {
        if splitUserMsg[0] == "/start" || splitUserMsg[0] == "start" {
            if splitUserMsg[1] == cfg.CmdConfig.BotPassword {
                groupIDBotStatusMap[groupID] = "Running"
                fmt.Printf("Status: [%s]", groupIDBotStatusMap[groupID])
                sendToTelegram(groupID, "Enter password successfully. Bot is working")
            }else {
                sendToTelegram(groupID, "Password input failed. Please try again")                                    
            }
        }
    }
}

func handleRunningStatusOfBot(groupID string, userMsg string) {
    splitUserMsg := strings.Split(userMsg, " ")    
    if len(splitUserMsg) == 1 {
        if splitUserMsg[0] == "/close" || splitUserMsg[0] == "close" {
            groupIDBotStatusMap[groupID] = "Block"
            sendToTelegram(groupID, "Bot temporarily stopped working")
        }else {
            handleTeleCmd(groupID, userMsg)            
        }
    }else {
        handleTeleCmd(groupID, userMsg)            
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
            handleTelegramScript(scriptVN, groupID)
        }else {
            fmt.Println("English")
            handleTelegramScript(scriptEN, groupID)          
        }            
    }
}

func handleTelegramScript(script *DeviceControlCode, groupID string) {
    readDataChannel, dataName := readDataFromDeviceChannel(script, cfg.CmdConfig.TickTimeout)
    if dataName == SensorDeivce {
        if readDataChannel != "Timeout" {
            responseSensor := script.ChatResponseMap["Sensor"]
            index := strings.Index(responseSensor, ":")
            textSensor := responseSensor[0:index+1]
            script.ChatResponseMap["Sensor"] = textSensor + " " + readDataChannel
            readDataChannel = "Sensor"
        }
    }
    resDataTele, checkKeyExists := script.ChatResponseMap[readDataChannel];

    switch checkKeyExists {
    case true:
        sendToTelegram(groupID, resDataTele)
    default:
        sendToTelegram(groupID, cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
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

func readDataFromDeviceChannel(deviceScript *DeviceControlCode, timeOut time.Duration) (string, DeviceName) {
    var dataChannel string
    deviceName := checkDeviceName(deviceScript)
    if deviceName == SensorDeivce {
        sendToSensorDeivce(deviceScript.DeviceCmd)
    }else {
        senToLedDeivce(deviceScript.DeviceCmd)
    }
    select {
    case dataChannel =  <-deviceChannel:
        return dataChannel, deviceName
    case <-time.After(timeOut * time.Second):
        dataChannel = "Timeout"
        return dataChannel, deviceName
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

func senToLedDeivce(msg string) {
    ledDeviceClient.Publish(cfg.MqttConfig.LedDeviceDstTopic, 0, false, msg)
}

func sendToSensorDeivce(msg string) {
    sensorDeviceClient.Publish(cfg.MqttConfig.SensorDeviceDstTopic, 0, false, msg)
    fmt.Printf("Publish: %s: %s\n", cfg.MqttConfig.SensorDeviceDstTopic, msg)
}

func sendToTelegram(groupID string, msg string) {
    teleDstTopic := strings.Replace(cfg.MqttConfig.TeleDstTopic, "GroupID", groupID, 1)
    telegramDeviceClient.Publish(teleDstTopic, 0, false, msg)
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
    groupIDBotStatusMap = make(map[string]string)
    yamlFileHandle()
    deviceChannel = make(chan string, 1)
    cmdListMapVN = cmdListMapInit(cfg.CmdConfig.ControlDeviceVN, "TimeoutVN")
    cmdListMapEN = cmdListMapInit(cfg.CmdConfig.ControlDeviceEN, "TimeoutEN")
    chatCmdlist = sortChatCmdlist()
    telegramDeviceClient = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageTelePubHandler)
    telegramDeviceClient.Subscribe(cfg.MqttConfig.TeleSrcTopic, 1, nil)
    ledDeviceClient = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageLedDevicePubHandler)
    ledDeviceClient.Subscribe(cfg.MqttConfig.LedDeviceSrcTopic, 1, nil)
    sensorDeviceClient = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageSensorDevicePubHandler)
    sensorDeviceClient.Subscribe(cfg.MqttConfig.SensorDeviceSrcTopic, 1, nil)

    fmt.Println("Connected")

    for {

        time.Sleep(2 * time.Second)
    }
}