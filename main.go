package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
)

func main() {
	Launch(func(bot *tb.Bot) {
		bot.Handle("/hello", func(m *tb.Message) {
			pass := CheckUser(m.Sender)
			if !pass {
				return
			}
			Send(m.Sender, "hello!"+m.Sender.LastName)
			Sendln(m.Sender, "bot configuration:")
			Sendf(m.Sender, "%#v", GlobalConfig)
		})
	})
}
