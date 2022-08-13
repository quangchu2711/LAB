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

    message := tgbotapi.NewMessage(cfg.TelegramConfig.IdBotChat, string(msg.Payload()))
    bot.Send(message)
}

func yaml_file_handle() {
    yfile, err := ioutil.ReadFile("config.yaml")

    if err != nil {

      log.Fatal(err)
    }

    err2 := yaml.Unmarshal(yfile, &cfg)

    if err2 != nil {

      log.Fatal(err2)
    }

}

func mqtt_begin(broker string) mqtt.Client {
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


func telegram_bot_begin(bot_token string) (tgbotapi.UpdatesChannel, *tgbotapi.BotAPI) {    
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
    yaml_file_handle()

    mqtt_client := mqtt_begin(cfg.MqttConfig.Broker)

    mqtt_client.Subscribe(cfg.MqttConfig.TeleDstTopic, 1, nil)

    updates, bot = telegram_bot_begin(cfg.TelegramConfig.BotToken)

    for update := range updates {
        mqtt_client.Publish(cfg.MqttConfig.TeleSrcTopic, 0, false, update.Message.Text)
        fmt.Println(cfg.MqttConfig.TeleSrcTopic + ": " + update.Message.Text)
    }
}