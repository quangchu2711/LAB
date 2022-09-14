package main

import (
    "log"
    "fmt"
    mqtt "github.com/eclipse/paho.mqtt.golang"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
    "github.com/ghodss/yaml"
    "io/ioutil"
    "strings"
    "strconv"
    "errors"
)

type Mqtt struct {
    Broker string
    TeleSrcTopic string
    TeleDstTopic string   
}

type Telegram struct {
    BotName string
    BotToken string
    IdBotChat int64
}

type FileConfig struct {
    TelegramConfig Telegram    
    MqttConfig Mqtt
}
var cfg FileConfig

var bot *tgbotapi.BotAPI
var updates tgbotapi.UpdatesChannel

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    fmt.Printf("Received message: [%s] from topic: %s\n", msg.Payload(), msg.Topic())

    groupID, err := getGroupIdTelegram(msg.Topic())

    if err != nil {

      log.Fatal(err)
    }

    message := tgbotapi.NewMessage(groupID, string(msg.Payload()))
    bot.Send(message)
}

func yamlFileHandle() {
    yfile, err1 := ioutil.ReadFile("config.yaml")

    if err1 != nil {

      log.Fatal(err1)
    }

    err2 := yaml.Unmarshal(yfile, &cfg)

    if err2 != nil {

      log.Fatal(err2)
    }

}

func mqttBegin(broker string) mqtt.Client {
    var opts *mqtt.ClientOptions = new(mqtt.ClientOptions)

    opts = mqtt.NewClientOptions()

    opts.AddBroker(fmt.Sprintf(broker))

    opts.SetDefaultPublishHandler(messagePubHandler)

    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }
    return client
}


func telegramBotBegin(bot_token string) (tgbotapi.UpdatesChannel, *tgbotapi.BotAPI) {      
    var telegramBot *tgbotapi.BotAPI
    var telgramUpdates tgbotapi.UpdatesChannel

    telegramBot, _ = tgbotapi.NewBotAPI(bot_token)

    log.Printf("Authorized on account %s", telegramBot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    telgramUpdates, _ = telegramBot.GetUpdatesChan(u)

    return telgramUpdates, telegramBot     
}
func getGroupIdTelegram (topic string) (int64, error) {
    topicItem := strings.Split(topic, "/")
    err := "Incorrect topic format"

    if topicItem[0] != "Telegram" {
        return 0, errors.New(err)
    }else {
        if topicItem[2] != "Tx" {
            return 0, errors.New(err)
        }else {
            groupID, _ := strconv.Atoi(topicItem[1])
            return int64(groupID), nil
        }
    }
}

func main() {
    yamlFileHandle()

    mqttClient := mqttBegin(cfg.MqttConfig.Broker)

    mqttClient.Subscribe(cfg.MqttConfig.TeleDstTopic, 1, nil)

    updates, bot = telegramBotBegin(cfg.TelegramConfig.BotToken)

    for update := range updates {

        groupID := strconv.FormatInt(int64(update.Message.Chat.ID), 10)
        // fmt.Println("[", groupID,"]")
        teleSrcTopic := strings.Replace(cfg.MqttConfig.TeleSrcTopic, "GroupID", groupID, 1)
        // fmt.Println("[", TeleSrcTopic,"]")
         
        mqttClient.Publish(teleSrcTopic, 0, false, update.Message.Text)
        fmt.Println(cfg.MqttConfig.TeleSrcTopic + ": " + update.Message.Text)
    }
}