package main

import (
    "log"
    "fmt"
    mqtt "github.com/eclipse/paho.mqtt.golang"

    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
    "github.com/ghodss/yaml"
    "io/ioutil"
)

type Mqtt struct {
    Broker string
    User string
    Password string
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
var msgChatId int64

var messagePubHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
    fmt.Printf("Received message: [%s] from topic: %s\n", msg.Payload(), msg.Topic())

    // message := tgbotapi.NewMessage(cfg.TelegramConfig.IdBotChat, string(msg.Payload()))
    message := tgbotapi.NewMessage(msgChatId, string(msg.Payload()))
    bot.Send(message)
}

func yamlFileHandle() {
    yfile, err := ioutil.ReadFile("config.yaml")

    if err != nil {

      log.Fatal(err)
    }

    err2 := yaml.Unmarshal(yfile, &cfg)

    if err2 != nil {

      log.Fatal(err2)
    }

}

func mqttBegin(broker string, user string, pw string) mqtt.Client {
    var opts *mqtt.ClientOptions = new(mqtt.ClientOptions)

    opts = mqtt.NewClientOptions()

    opts.AddBroker(fmt.Sprintf(broker))

    opts.SetUsername(user)
    opts.SetPassword(pw)   
    opts.SetCleanSession(true)


    opts.SetDefaultPublishHandler(messagePubHandler)

    client := mqtt.NewClient(opts)
    if token := client.Connect(); token.Wait() && token.Error() != nil {
        panic(token.Error())
    }
    return client
}


func telegramBotBegin(bot_token string) (tgbotapi.UpdatesChannel, *tgbotapi.BotAPI) {    
    var error_bot error 
    var error_update error     
    var telegram_bot *tgbotapi.BotAPI
    var telgram_updates tgbotapi.UpdatesChannel

    telegram_bot, error_bot = tgbotapi.NewBotAPI(bot_token)
    if error_bot != nil {
        log.Panic(error_bot)
    }

    log.Printf("Authorized on account %s", telegram_bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    telgram_updates, error_update = telegram_bot.GetUpdatesChan(u)

    if error_update != nil {
        log.Panic(error_update)
    }         

    return telgram_updates, telegram_bot     
}

func main() {
    yamlFileHandle()

    mqttClient := mqttBegin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password)

    mqttClient.Subscribe(cfg.MqttConfig.TeleDstTopic, 1, nil)

    updates, bot = telegramBotBegin(cfg.TelegramConfig.BotToken)

    for update := range updates {
        // fmt.Printf("Type: %T\n", update.Message.Chat.ID)
        msgChatId = update.Message.Chat.ID
        mqttClient.Publish(cfg.MqttConfig.TeleSrcTopic, 0, false, update.Message.Text)
        fmt.Println(cfg.MqttConfig.TeleSrcTopic + ": " + update.Message.Text)
    }
}