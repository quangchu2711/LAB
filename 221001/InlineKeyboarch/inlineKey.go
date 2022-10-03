package main

import (
    "log"
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

var numericKeyboard = tgbotapi.NewInlineKeyboardMarkup(
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonData("1", "Số một"),
        tgbotapi.NewInlineKeyboardButtonData("2", "Số hai"),
        tgbotapi.NewInlineKeyboardButtonData("3", "Số ba"),
    ),
    tgbotapi.NewInlineKeyboardRow(
        tgbotapi.NewInlineKeyboardButtonData("4", "Số 4"),
        tgbotapi.NewInlineKeyboardButtonData("5", "Số 5"),
        tgbotapi.NewInlineKeyboardButtonData("6", "Số 6"),
    ),
)

func main() {
    bot, err := tgbotapi.NewBotAPI("5379690042:AAHsmvcazskbA3zmC6IyINMp8k0rezViCGk")
    if err != nil {
        log.Panic(err)
    }

    bot.Debug = false

    log.Printf("Authorized on account %s", bot.Self.UserName)

    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60

    updates := bot.GetUpdatesChan(u)

    // Loop through each update.
    for update := range updates {
        // Check if we've gotten a message update.
        if update.Message != nil {
            // Construct a new message from the given chat ID and containing
            // the text that we received.
            msg := tgbotapi.NewMessage(update.Message.Chat.ID, update.Message.Text)

            // If the message was open, add a copy of our numeric keyboard.
            switch update.Message.Text {
            case "open":
                msg.ReplyMarkup = numericKeyboard

            }

            // Send the message.
            if _, err = bot.Send(msg); err != nil {
                panic(err)
            }
        } else if update.CallbackQuery != nil {
            // Respond to the callback query, telling Telegram to show the user
            // a message with the data received.
            callback := tgbotapi.NewCallback(update.CallbackQuery.ID, update.CallbackQuery.Data)
            if _, err := bot.Request(callback); err != nil {
                panic(err)
            }

            // And finally, send a message containing the data received.
            msg := tgbotapi.NewMessage(update.CallbackQuery.Message.Chat.ID, update.CallbackQuery.Data)
            if _, err := bot.Send(msg); err != nil {
                panic(err)
            }
        }
    }
}
