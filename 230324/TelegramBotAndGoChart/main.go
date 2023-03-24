package main

//go:generate go run main.go

import (
    "os"
    "log"
    "io/ioutil"
    "github.com/wcharczuk/go-chart/v2"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var bot *tgbotapi.BotAPI
var updates tgbotapi.UpdatesChannel

func TelegramBot_Begin(bot_token string) (tgbotapi.UpdatesChannel, *tgbotapi.BotAPI) {
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

func TelegramBot_SendMessage(groupID int64, msg string) {
    message := tgbotapi.NewMessage(groupID, msg)
    bot.Send(message)
}

func TelegramBot_SendPhoto(groupID int64, url string) {
    photoBytes, err := ioutil.ReadFile(url)
    if err != nil {
        panic(err)
    }
    photoFileBytes := tgbotapi.FileBytes{
        Name:  "picture",
        Bytes: photoBytes,
    }
    bot.Send(tgbotapi.NewPhotoUpload(groupID, photoFileBytes))
}


func main() {
    _, bot = TelegramBot_Begin("5379690042:AAHsmvcazskbA3zmC6IyINMp8k0rezViCGk")

    graph := chart.Chart{
        Series: []chart.Series{
            chart.ContinuousSeries{
                XValues: []float64{1.0, 2.0, 3.0, 4.0, 5.0},
                YValues: []float64{1.0, 2.0, 3.0, 4.0, 5.0},
            },
            chart.AnnotationSeries{
                Annotations: []chart.Value2{
                    {XValue: 1.0, YValue: 1.0, Label: "Một"},
                    {XValue: 2.0, YValue: 2.0, Label: "Hai"},
                    {XValue: 3.0, YValue: 3.0, Label: "Ba"},
                    {XValue: 4.0, YValue: 4.0, Label: "Bốn"},
                    {XValue: 5.0, YValue: 5.0, Label: "Năm"},
                },
            },
        },
    }

    f, _ := os.Create("output.png")
    defer f.Close()
    graph.Render(chart.PNG, f)

    TelegramBot_SendMessage(2105095097, "Gửi đồ thị:")
    TelegramBot_SendPhoto(2105095097, "output.png")
}
