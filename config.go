package main

type Config struct {
	BotToken string   `yaml:"botToken"`
	Users    []string `yaml:"users"`
}
