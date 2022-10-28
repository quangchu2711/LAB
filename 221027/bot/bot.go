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
var telegramClient mqtt.Client
var deviceClient mqtt.Client
var sensorDeviceClient mqtt.Client
var deviceChannel chan string
var cmdListMapVN map[string]*DeviceControlCode
var cmdListMapEN map[string]*DeviceControlCode
var chatCmdlist[] string

var messageTelegramPubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    teleMsg := string(msg.Payload())
    fmt.Printf("Received message: [%s] from topic: %s\n", teleMsg, msg.Topic())
    groupID, _ := getGroupIdTelegram(msg.Topic())
    handleTelegramCmd(groupID, teleMsg)
}

var messageDevicePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    deviceMsg := string(msg.Payload())
    fmt.Printf("Received message: [%s] from topic: %s\n", deviceMsg, msg.Topic())
    writeDataToDeviceChannel(deviceMsg)
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

func cmdListMapInit(controlDeviceArr []DeviceControlCode,
                    msgTimeout string) map[string]*DeviceControlCode {
    cmdListMap := make(map[string]*DeviceControlCode)

    for i := 0 ; i < len(controlDeviceArr); i++ {
        cmdListMap[controlDeviceArr[i].ChatCmd] = &controlDeviceArr[i]
        chatCmdlist = append(chatCmdlist, controlDeviceArr[i].ChatCmd)
    }
    for _, controlLed :=range controlDeviceArr {
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

func handleTelegramCmd(groupID string, cmd string) {
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
    var channelData string
    deviceName := checkDeviceName(deviceScript)
    sendToDeivce(deviceScript.DeviceCmd)
    select {
    case channelData =  <-deviceChannel:
        return channelData, deviceName
    case <-time.After(timeOut * time.Second):
        channelData = "Timeout"
        return channelData, deviceName
    }
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

func writeDataToDeviceChannel(cmd string) {
    deviceChannel <-cmd
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

func sendToDeivce(msg string) {
    deviceClient.Publish(cfg.MqttConfig.DeviceDstTopic, 0, false, msg)
    fmt.Println("Publish: " + cfg.MqttConfig.DeviceDstTopic + " msg: " + msg)
}

func sendToTelegram(groupID string, msg string) {
    teleDstTopic := strings.Replace(cfg.MqttConfig.TeleDstTopic, "GroupID", groupID, 1)
    telegramClient.Publish(teleDstTopic, 0, false, msg)
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
        numTransStep := levenshtein.ComputeDistance(str1, str2)
        lenStr2 := len(str2)
        strCmp.Data = strArr[i]
        if numTransStep >= lenStr2 {
            strCmp.RatePercent = 0.0
        }else {
            strCmp.RatePercent = 100.0 - (float32(numTransStep) / float32(len(str2)) * 100.0)
        }
        chatCmdArr = append(chatCmdArr, strCmp)
        // fmt.Printf("[%s - %d - %.2f]\n", getNormStr(strArr[i]), numTransStep, strCmp.RatePercent)
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
    cmdListMapVN = cmdListMapInit(cfg.CmdConfig.ControlDeviceVN, "TimeoutVN")
    cmdListMapEN = cmdListMapInit(cfg.CmdConfig.ControlDeviceEN, "TimeoutEN")
    chatCmdlist = sortChatCmdlist()
    telegramClient = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageTelegramPubHandler)
    telegramClient.Subscribe(cfg.MqttConfig.TeleSrcTopic, 1, nil)
    deviceClient = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageDevicePubHandler)
    deviceClient.Subscribe(cfg.MqttConfig.DeviceSrcTopic, 1, nil)

    fmt.Println("Connected")

    for {

        time.Sleep(2 * time.Second)
    }
}