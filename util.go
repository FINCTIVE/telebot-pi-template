package main

import (
	tb "gopkg.in/tucnak/telebot.v2"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
	"time"
	"unicode/utf8"
)

var globalBot *tb.Bot
var GlobalConfig Config

// Launch loads the yaml configuration file, and start the bot.
func Launch(load func(bot *tb.Bot)) {
	configBytes, err := ioutil.ReadFile("config.yaml")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	err = yaml.Unmarshal(configBytes, &GlobalConfig)
	if err != nil {
		log.Fatalf("error: %v", err)
	}

	globalBot, err = tb.NewBot(tb.Settings{
		Token:  GlobalConfig.BotToken,
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	load(globalBot)

	log.Println("verbose: bot started")
	globalBot.Start()
}

// CheckUser will check whether the username is in the config.yaml
// if the users property in config.yaml is set to be "*" or empty,
// all users will pass the check
func CheckUser(sender *tb.User) (pass bool) {
	if len(GlobalConfig.Users) == 0 || GlobalConfig.Users[0] == "*" {
		return true
	}

	pass = false
	for _, username := range GlobalConfig.Users {
		if username == sender.Username {
			pass = true
			break
		}
	}
	if pass == false {
		Send(sender, "Sorry, it's a bot for private usage.", "", "")
	}
	log.Println("verbose: check ", sender.Username, " - pass:", pass)
	return pass
}

// if the message is too long, cut it into pieces
// and send separately
const LongMessageLength = 4000
const MaxRetry = 1000

// Send sends message. If failed, retry until it's successful.
// (to deal with poor network problem ...)
// Also, Send() split long message into small pieces. (Telegram has message length limit.)
// and send them separately.
func Send(sender *tb.User, message string, options ...interface{}) {
	SendWithSurround(sender, message, "", "", options...)
}

// SendWithSurround is identical to Send
// Every message will be sent with prefix and postfix
func SendWithSurround(sender *tb.User, message, prefix, postfix string, options ...interface{}) {
	messages := splitByLines(message, LongMessageLength)
	retryCounter := 0
	for {
		_, err := globalBot.Send(sender, prefix+messages[0]+postfix, options...)
		if err != nil {
			log.Println("err: send:", messages[0])
			log.Println(err)
			retryCounter++
			if retryCounter >= MaxRetry {
				log.Println("err: send: tried ", MaxRetry, " times. Give it up.")
				// for errors not related with network
				_, _ = globalBot.Send(sender, "Messages not sent, please check your terminal log.(it may not be an issue with networking)", options...)
				break
			}
		} else {
			if len(messages) == 1 {
				break
			} else {
				messages = messages[1:]
			}
		}
	}
}

// splitByLines will split input string into pieces within limit length.
// it will split by lines. (\n)
// if one line is too long, it will be broken into two results item.
func splitByLines(input string, limit int) (results []string) {
	if utf8.RuneCountInString(input) <= limit {
		return []string{input}
	}

	messageRunes := []rune(input)
	var splitMessage [][]rune

	startIndex := 0
	for {
		cutIndex := startIndex + limit - 1
		if cutIndex > len(messageRunes)-1 {
			cutIndex = len(messageRunes) - 1
		}
		fullLine := false
		for i := cutIndex; i >= startIndex; i-- {
			if messageRunes[i] == '\n' {
				splitMessage = append(splitMessage, messageRunes[startIndex:i+1])
				startIndex = i + 1
				fullLine = true
				break
			}
		}
		if !fullLine {
			splitMessage = append(splitMessage, messageRunes[startIndex:cutIndex+1])
			startIndex = cutIndex + 1
		}
		if startIndex == len(messageRunes) {
			break
		}
	}

	for _, v := range splitMessage {
		msg := strings.Trim(string(v), " \n")
		if len(msg) != 0 {
			results = append(results, msg)
		}
	}

	return
}

// RunCommand will send terminal output messages to the user.
// listen to done to wait until the command is finished.
// notice: ParseMode is forced to be ModeHTML
func RunCommand(sender *tb.User, cmd *exec.Cmd, options ...interface{}) (done chan error) {
	done = make(chan error, 0)

	// option settings
	setHTMLParse := false
	for i := range options {
		value, ok := options[i].(*tb.SendOptions)
		if ok {
			(*value).ParseMode = tb.ModeHTML
			setHTMLParse = true
		}
	}
	if !setHTMLParse {
		options = append(options, &tb.SendOptions{ParseMode: tb.ModeHTML})
	}
	options = append(options, tb.Silent, tb.NoPreview)

	output, doneCmd := runCmdAndCapture(cmd)

	var terminalMessage *tb.Message
	go func() {
		for {
			select {
			case <-time.After(time.Second):
				messageStr := cleanRemovedOutput(string(*output))
				//messageStr := string(*output)
				if len(messageStr) == 0 {
					messageStr = "..." // prevent empty message
				}
				messageRunes := []rune(messageStr)
				surroundLen := len("<pre>" + "</pre>")
				if len(messageRunes) > LongMessageLength-surroundLen {
					messageRunes = messageRunes[len(messageRunes)-LongMessageLength-surroundLen-1:]
					messageStr = string(messageRunes)
				}
				messageStr = "<pre>" + messageStr + "</pre>"
				var err error
				if terminalMessage == nil {
					terminalMessage, err = globalBot.Send(sender, messageStr, options...)
					if err != nil {
						log.Println("err: ", "sending first terminal message err:\n", err)
						//log.Println("messageStr:\n", messageStr)
						terminalMessage = nil
					}
				} else {
					if terminalMessage.Text != messageStr {
						_, err = globalBot.Edit(terminalMessage, messageStr, options...)
						// ignore edit err. the message will be delete anyway.
						// a common err is that old message is still the same as the new one.
						//if err != nil {
						//log.Println("err: edit: \n", err)
						//log.Println("messageStr:\n", messageStr)
						//}
					}
				}
			case err := <-doneCmd:
				if terminalMessage != nil {
					err := globalBot.Delete(terminalMessage)
					if err != nil {
						log.Println(err)
					}
				}
				SendWithSurround(sender, cleanRemovedOutput(string(*output)), "<pre>", "</pre>", options...)
				//SendWithSurround(sender, string(*output), "<pre>", "</pre>", options...)
				if err != nil {
					terminatedMessage := "Terminated:\n\n<pre>" + err.Error() + "</pre>"
					Send(sender, terminatedMessage, options...)
				}
				done <- err
				return
			}
		}
	}()
	return
}

// runCmdAndCapture runs a command in the background and capture its output bytes in real time,
// combining stdout and stderr output.
// listen to done channel to wait the command finish.
// note: the output slice has no lock.
const OutputBufferSize = 1024 * 10

func runCmdAndCapture(cmd *exec.Cmd) (output *[]byte, done chan error) {
	outputBytes := make([]byte, 0)
	done = make(chan error, 0)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()
	outputReader := io.MultiReader(stdout, stderr)
	err := cmd.Start()
	if err != nil {
		// ignore
		log.Println(err)
	}

	go func() {
		done <- cmd.Wait()
	}()

	go func() {
		// read output
		var buffer = make([]byte, OutputBufferSize)
		for {
			n, readErr := outputReader.Read(buffer)
			if n > 0 {
				outputBytes = append(outputBytes, buffer[:n]...)
			}
			if readErr != nil {
				if readErr != io.EOF {
					// ignore err
					log.Println(readErr)
				}
				break
			}
		}
	}()

	output = &outputBytes
	return
}

// cleanRemovedOutput will remove \b and \r characters,
// and return a string that just like what you see in a terminal
func cleanRemovedOutput(input string) string {
	var inputRunes, outputRunes []rune
	inputRunes = []rune(input)
	maxLength := 0
	for i := range inputRunes {
		if inputRunes[i] == '\b' {
			outputRunes = outputRunes[:len(outputRunes)-1]
		} else if inputRunes[i] == '\r' {
			for index := range outputRunes {
				if outputRunes[len(outputRunes)-1-index] == '\n' {
					if maxLength < len(outputRunes) {
						maxLength = len(outputRunes)
					}
					outputRunes = outputRunes[:len(outputRunes)-index]
					break
				}
			}
			// When there is no \n
			if maxLength < len(outputRunes) {
				maxLength = len(outputRunes)
			}
			outputRunes = outputRunes[:0]
		} else {
			outputRunes = append(outputRunes, inputRunes[i])
		}
	}

	//return string(outputRunes)
	if maxLength > len(outputRunes) {
		return string(outputRunes[:maxLength])
	} else {
		return string(outputRunes)
	}
}
