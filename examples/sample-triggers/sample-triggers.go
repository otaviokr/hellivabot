package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	hbot "github.com/otaviokr/hellivabot"
)

// This trigger will op people in the given list who ask by saying "-opme"
var oplist = []string{"otaviokr", "tlane", "ltorvalds"}
var opPeople = hbot.Trigger{
	Condition: func(bot *hbot.Bot, m *hbot.Message) bool {
		if m.Content == "-opme" {
			for _, s := range oplist {
				if m.From == s {
					return true
				}
			}
		}
		return false
	},
	Action: func(irc *hbot.Bot, m *hbot.Message) bool {
		irc.ChMode(m.To, m.From, "+o")
		return false
	},
}

// This trigger will say the contents of the file "info" when prompted
var sayInfoMessage = hbot.Trigger{
	Condition: func(bot *hbot.Bot, m *hbot.Message) bool {
		return m.Command == "PRIVMSG" && m.Content == "-info"
	},
	Action: func(irc *hbot.Bot, m *hbot.Message) bool {
		fi, err := os.Open("info")
		if err != nil {
			return false
		}
		info, _ := ioutil.ReadAll(fi)

		irc.Send("PRIVMSG " + m.From + " : " + string(info))
		return false
	},
}

// This trigger will listen for -toggle, -next and -prev and then
// perform the mpc action of the same name to control an mpd server running
// on localhost
var mpc = hbot.Trigger{
	Condition: func(bot *hbot.Bot, m *hbot.Message) bool {
		return m.Command == "PRIVMSG" && (m.Content == "-toggle" || m.Content == "-next" || m.Content == "-prev")
	},
	Action: func(irc *hbot.Bot, m *hbot.Message) bool {
		var mpcCMD string
		switch m.Content {
		case "-toggle":
			mpcCMD = "toggle"
		case "-next":
			mpcCMD = "next"
		case "-prev":
			mpcCMD = "prev"
		default:
			fmt.Println("Invalid command.")
			return false
		}
		cmd := exec.Command("/usr/bin/mpc", mpcCMD)
		err := cmd.Run()
		if err != nil {
			fmt.Printf("error: %s\n", err)
		}
		return true
	},
}
