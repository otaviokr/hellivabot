// This is an example program showing the usage of hellivabot
package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	hbot "github.com/otaviokr/hellivabot"
	log "github.com/sirupsen/logrus"
)

var (
	serv = "irc.chat.twitch.tv:6697"	// This is the actual Twitch SSL host and port. No need to change this.
	nick = "the_name_of_the_bot" 		// it needs to be an actual registered username.
	password = "oauth:abcdefghijk" 		// To get the token, login to the bot account and go to https://twitchapps.com/tmi/
)

func main() {
	log.SetLevel(log.DebugLevel)

	// We cannot use the hijacking feature, because we are connecting via SSL.
	hijackSession := func(bot *hbot.Bot) {
		bot.HijackSession = false
	}

	// Feel free to change the channel, but be careful when testing the bot in a channel that does not belong to you!
	channels := func(bot *hbot.Bot) {
		bot.Channels = []string{"#" + nick}
	}

	// No need to change this. It will pass the OAUth password token to authenticate.
	passwordOpt := func(bot *hbot.Bot) {
		bot.SSL = true
		if len(password) > 0 {
			bot.Password = password
		}
	}

	// Use this to not send messages to Twitch too quickly.
	// https://dev.twitch.tv/docs/irc/guide#command--message-limits
	throttleDelay := func(bot *hbot.Bot) {
		bot.ThrottleDelay = 500 * time.Millisecond
	}

	// Finally, we create the bot.
	irc, err := hbot.NewBot(serv, nick, hijackSession, channels, passwordOpt, throttleDelay)
	if err != nil {
		panic(err)
	}

	// Add the triggers that will monitor the messages.
	irc.AddTrigger(sayHello)
	irc.AddTrigger(replyHello)

	// Start up bot (this blocks until we disconnect)
	irc.Run()
	log.Warn("bot shutting down")
}

// This trigger says hello when someone enters the chat.
var sayHello = hbot.Trigger{
	Condition: func(bot *hbot.Bot, m *hbot.Message) bool {
		return m.Command == "JOIN" && strings.EqualFold(m.Content, nick)
	},
	Action: func(bot *hbot.Bot, m *hbot.Message) bool {
		bot.Reply(m, fmt.Sprintf("Hello, %s! Welcome!", m.Content))
		return false
	},
}

// This trigger replies hello when you send "!hello" or "!hi"
var replyHello = hbot.Trigger {
	Condition: func(bot *hbot.Bot, m *hbot.Message) bool {
		return m.Command == "PRIVMSG" && regexp.MustCompile("!h(i|ello).*").MatchString(m.Content)
	},
	Action: func(bot *hbot.Bot, m *hbot.Message) bool {
		bot.Reply(m, fmt.Sprintf("Hello back to you, %s", m.From))
		return false
	},
}
