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
    "encoding/json"
    "strconv"
)

type Mqtt struct {
    Broker string
    User string
    Password string
    LedDeviceCmdTopic string
    LedDeviceStatusTopic string
    TeleSrcTopic string
    TeleDstTopic string
    SensorTopic string
}


type DeviceControlCode struct {
    ChatCmd string
    DeviceID string
    DeviceCmd string
    ChatResponseMap map[string]string
}

type Command struct {
    ControlDeviceVN []DeviceControlCode
    ControlDeviceEN []DeviceControlCode

    DefaultRespMsg map[string]string
    TickTimeout time.Duration
    StringRateThreshold float32
    DeviceControlGroupID string
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

type Sensor struct {
    Temperature string
    Humidity string
}

var cfg FileConfig
var chatCmdlist[] string
var cmdListMapVN map[string]*DeviceControlCode
var cmdListMapEN map[string]*DeviceControlCode
var ledDeviceClient mqtt.Client
var ledDeviceChannel chan string
var telegramClient mqtt.Client
var sensorDeviceClient mqtt.Client
var deviceUserMap map[string]string

var messageTelegramPubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    teleMsg := string(msg.Payload())
    fmt.Printf("Received message: %s from topic: %s\n", teleMsg, msg.Topic())
    groupID, _ := getGroupIdTelegram(msg.Topic())
    handleTelegramCmd(groupID, teleMsg)
}

var messageLedDevicePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    deviceMsg := string(msg.Payload())
    fmt.Printf("Received message: %s from topic: %s\n", deviceMsg, msg.Topic())
    deviceID,_ := getDeviceIDFormLedStatusTopic(msg.Topic())
    _, checkKeyExists := deviceUserMap[deviceID]
    if checkKeyExists == true{
        writeDataToLedDeviceChannel(deviceMsg)
    }else {
        responseMsg := "NULL"
        textVN := "Người dùng HA: "
        for _, controlDevice :=range cfg.CmdConfig.ControlDeviceVN{
            if controlDevice.DeviceCmd == deviceMsg {
                textVN += controlDevice.ChatCmd + "\n"
                responseMsg = controlDevice.ChatResponseMap[deviceMsg]
            }
        }
        if responseMsg != "NULL" {
            sendToTelegram(cfg.CmdConfig.DeviceControlGroupID, textVN + responseMsg)
        }
    }
}

var messageSensorDevicePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    deviceMsg := string(msg.Payload())
    fmt.Printf("Received message: %s from topic: %s\n", deviceMsg, msg.Topic())
    deviceID, err := getDeviceIDFormSensorTopic(msg.Topic())
    if err == nil {
        testJsonString := `{"temperature":"23.45","humidity":"67"}`
        var sensorValue Sensor
        // err := json.Unmarshal([]byte(deviceMsg), &sensorValue)
        err := json.Unmarshal([]byte(testJsonString), &sensorValue)
        if err != nil {
            fmt.Println("[Json err]")
        }else {
            fmt.Printf("Data: %+v\n", sensorValue)
            updateDeviceSensorData(deviceID, sensorValue)
        }
    }
}

func addSensorData(deviceID string,
                   controlDeviceArr []DeviceControlCode,
                   valueSenosr Sensor,
                   updateTime string) {
    for _, control := range controlDeviceArr {
        if control.DeviceID == deviceID {
            if control.DeviceCmd == "TEM" {
                control.ChatResponseMap["Value"] = valueSenosr.Temperature + "℃"
            }else if control.DeviceCmd == "HUM" {
                control.ChatResponseMap["Value"] = valueSenosr.Humidity + "%"
            }
            control.ChatResponseMap["UpdateTime"] = updateTime
        }
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

func getDeviceIDFormLedStatusTopic(topic string) (string, error) {
    topicItem := strings.Split(topic, "/")
    err := "Incorrect topic format"

    if topicItem[0] != "xuong" {
        return "0", errors.New(err)
    }else {
        if topicItem[1] != "device" {
            return "0", errors.New(err)
        }else {
            if topicItem[3] != "status" {
                return "0", errors.New(err)
            }else {
                deviceID := topicItem[2]
                return deviceID, nil
            }
        }
    }
}

func getDeviceIDFormSensorTopic(topic string) (string, error) {
    topicItem := strings.Split(topic, "/")
    err := "Incorrect topic format"

    if topicItem[0] != "xuong" {
        return "0", errors.New(err)
    }else {
        if topicItem[1] != "device" {
            return "0", errors.New(err)
        }else {
            if topicItem[3] != "sensor" {
                return "0", errors.New(err)
            }else {
                deviceID := topicItem[2]
                return deviceID, nil
            }
        }
    }
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

func createTextAboutTimeDifference(language string, hours int, minutes, seconds int) string {
    var text string
    if hours > 0 {
        if hours > 24 {
            text = ""
        }else {
            if language == "Vietnamese" {
                text = "Cập nhật " + strconv.Itoa(hours) + " giờ trước"
            }else {
                text = "Updated " + strconv.Itoa(hours) + " hours ago"
            }
        }

    }else if minutes > 0 {
        if language == "Vietnamese" {
            text = "Cập nhật " + strconv.Itoa(minutes) + " phút trước"
        }else {
            text = "Updated " + strconv.Itoa(minutes) + " minutes ago"
        }
    }else if seconds > 0 {
        if language == "Vietnamese" {
            text = "Cập nhật " + strconv.Itoa(seconds) + " giây trước"
        }else {
            text = "Updated " + strconv.Itoa(seconds) + " seconds ago"
        }
    }
    return text

}

func handleTelegramScript(script *DeviceControlCode, groupID string, language string) {
    deviceName := checkDeviceName(script)

    switch deviceName {
    case LedDevice:
        readDataChannel := readDataFromLedDeviceChannel(script)
        resDataTele, checkKeyExists := script.ChatResponseMap[readDataChannel];
        if checkKeyExists == true {
            sendToTelegram(groupID, resDataTele)
        }else {
            sendToTelegram(groupID, cfg.CmdConfig.DefaultRespMsg["ErrorCmd"])
        }
    case SensorDeivce:
        sensorText := script.ChatResponseMap["Sensor"]
        sensorValue := script.ChatResponseMap["Value"]
        updateTime := script.ChatResponseMap["UpdateTime"]

        previousTime, _ := time.Parse("01-02-2006 15:04:05", updateTime)
        timeNow := time.Now()
        timeNowFormat := timeNow.Format("01-02-2006 15:04:05")
        currentTime, _ := time.Parse("01-02-2006 15:04:05", timeNowFormat)
        diffTime := currentTime.Sub(previousTime)
        hours := int(diffTime.Hours())
        minutes := int(diffTime.Minutes())
        seconds := int(diffTime.Seconds())
        if language == "Vietnamese" {
            textDiffVN := createTextAboutTimeDifference("Vietnamese", hours, minutes, seconds)
            sendToTelegram(groupID, sensorText + sensorValue + "\n" + updateTime + "\n" + textDiffVN)
        }else {
            textDiffEN := createTextAboutTimeDifference("English", hours, minutes, seconds)
            sendToTelegram(groupID, sensorText + sensorValue + "\n" + updateTime + "\n" + textDiffEN)
        }
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
            handleTelegramScript(scriptVN, groupID, "Vietnamese")
        }else {
            fmt.Println("English")
            handleTelegramScript(scriptEN, groupID, "English")
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

func readDataFromLedDeviceChannel(deviceScript *DeviceControlCode) (string) {
    var channelData string
    fmt.Printf("<%s>\n", deviceScript.DeviceID)
    deviceUserMap[deviceScript.DeviceID] = "TelegramUser"
    sendToDeivce(deviceScript.DeviceID, deviceScript.DeviceCmd)
    select {
    case channelData =  <-ledDeviceChannel:
        delete(deviceUserMap, deviceScript.DeviceID)
        return channelData
    case <-time.After(cfg.CmdConfig.TickTimeout * time.Second):
        delete(deviceUserMap, deviceScript.DeviceID)
        channelData = "Timeout"
        return channelData
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

func writeDataToLedDeviceChannel(cmd string) {
    ledDeviceChannel <-cmd
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

func sendToDeivce(deviceID string, msg string) {
    ledDeviceCmdTopic := strings.Replace(cfg.MqttConfig.LedDeviceCmdTopic, "DeviceID", deviceID, 1)
    ledDeviceClient.Publish(ledDeviceCmdTopic, 0, false, msg)
    fmt.Println("Publish: " + ledDeviceCmdTopic + " msg: " + msg)
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

func updateDeviceSensorData(deviceID string, valueSensor Sensor) {
    currentTime := time.Now()
    updateSensorTime := currentTime.Format("01-02-2006 15:04:05")
    addSensorData(deviceID, cfg.CmdConfig.ControlDeviceVN, valueSensor, updateSensorTime)
    addSensorData(deviceID, cfg.CmdConfig.ControlDeviceEN, valueSensor, updateSensorTime)
    fmt.Println("Updated")
}

func yamlFileHandle() {
    yfile, _ := ioutil.ReadFile("config.yaml")
    yaml.Unmarshal(yfile, &cfg)
}


func main() {
    yamlFileHandle()
    deviceUserMap = make(map[string]string)
    ledDeviceChannel = make(chan string, 1)
    cmdListMapVN = cmdListMapInit(cfg.CmdConfig.ControlDeviceVN, "TimeoutVN")
    cmdListMapEN = cmdListMapInit(cfg.CmdConfig.ControlDeviceEN, "TimeoutEN")
    chatCmdlist = sortChatCmdlist()

    telegramClient = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageTelegramPubHandler)
    telegramClient.Subscribe(cfg.MqttConfig.TeleSrcTopic, 1, nil)

    ledDeviceClient = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageLedDevicePubHandler)
    ledDeviceClient.Subscribe(cfg.MqttConfig.LedDeviceStatusTopic, 1, nil)

    sensorDeviceClient = mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password, &messageSensorDevicePubHandler)
    sensorDeviceClient.Subscribe(cfg.MqttConfig.SensorTopic, 1, nil)

    fmt.Println("Connected")

    for {

        time.Sleep(2 * time.Second)
    }
}