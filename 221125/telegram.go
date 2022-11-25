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
    TopicTest string
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

    textMsg := string(msg.Payload())
    var cmdMsg []string
    cmdMsg = append(cmdMsg, textMsg)
    cmdMsg = append(cmdMsg, textMsg)
    // cmdMsg := [...]string{"BUTTON 1", "BUTTON 2"}
    sendInlineButtonMsgToTelegramGroup(cfg.TelegramConfig.IdBotChat, textMsg, cmdMsg)
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

func mqtt_begin(broker string, user string, pw string) mqtt.Client {
    var opts *mqtt.ClientOptions = new(mqtt.ClientOptions)

    opts = mqtt.NewClientOptions()
    opts.AddBroker(fmt.Sprintf(broker))
    opts.SetUsername(user)
    opts.SetPassword(pw)
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

func sendInlineButtonMsgToTelegramGroup(groupID int64, textDisplay string, cmdArr []string) {
    messageSendTele := tgbotapi.NewMessage(groupID, textDisplay)
    var inlineCmd  tgbotapi.InlineKeyboardButton
    var rowInlineCmd []tgbotapi.InlineKeyboardButton
    var keyboard [][]tgbotapi.InlineKeyboardButton

    for _, text := range cmdArr {
        inlineCmd.Text = text
        cmd := text
        inlineCmd.CallbackData = &cmd
        rowInlineCmd = append(rowInlineCmd, inlineCmd)
    }
    jump := 2
    if len(cmdArr) % 2  != 0 {
        jump = 1
    }
    for i := 0; i < len(rowInlineCmd); i+=jump {
        sliceRowInlineCmd := rowInlineCmd[i:(i+jump)]
        keyboard = append(keyboard, sliceRowInlineCmd)
    }
    messageSendTele.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
        InlineKeyboard: keyboard,
    }

    bot.Send(messageSendTele)
}

func sendToTelegram(groupID int64, msg string) {
    message := tgbotapi.NewMessage(groupID, msg)
    bot.Send(message)
}

func main() {
    yaml_file_handle()

    mqtt_client := mqtt_begin(cfg.MqttConfig.Broker, cfg.MqttConfig.User, cfg.MqttConfig.Password)

    mqtt_client.Subscribe(cfg.MqttConfig.TopicTest, 1, nil)

    updates, bot = telegram_bot_begin(cfg.TelegramConfig.BotToken)

    for update := range updates {
        if update.Message != nil {
            message := tgbotapi.NewMessage(cfg.TelegramConfig.IdBotChat, update.Message.Text)
            bot.Send(message)
            fmt.Printf("======> ID: %d\n", update.Message.Chat.ID)
            fmt.Println("Telegram: " + update.Message.Text)
        }else if update.CallbackQuery != nil {
            groupID := update.CallbackQuery.Message.Chat.ID
            sendToTelegram(groupID, update.CallbackQuery.Data)
        }
    }
}