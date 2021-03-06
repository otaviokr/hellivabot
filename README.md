# Hellivabot

[![GoDoc](https://godoc.org/github.com/otaviokr/hellivabot?status.png)](https://godoc.org/github.com/otaviokr/hellivabot)

Hellivabot is forked from the super [WhyRUSleeping's HellaBot](https://github.com/whyrusleeping/hellabot), an IRC bot in Go that, unfortunately does not play nice with Twitch's chat particularities. Thanks for WhyRUSleeping and everyone who contributed to Hellabot!

# Overview

This is an IRC bot written in Go focused on Twitch's chats. That means that the message handling, the expected parameters, values and details are always to favour how Twitch works. As mentioned above, if you need something less specific, check out [WhyRUSleeping's HellaBot](https://github.com/whyrusleeping/hellabot) project.

Bots for Twitch's chat are useful to manage the chats, help users and keep thing running nice while you livestream. This bot is not programmed to do anything out-of-the-box, except the most basic connection and authentication process, but you can easily add triggers for all events that you want it to react to.

If you want some inspiration, I invite you to check out my implementation of the Hellivabot that I use for my livestreams: [BOTavio_KR-twitch-bot](https://github.com/otaviokr/botaviokr-twitch-bot). If you want to see it in action, come visit my live at [otavio_kr](https://www.twitch.tv/otavio_kr).

# The Name

The original project is called Hellabot, which sounds to me like "hell-of-a-bot". Initially, I thought calling mine as "helluvabot", but I thought I could squeeze a "live" in there, so it is now something like "hell-live-a-bot".

# Changes from the original project

## A different log library

I prefer [Sirupsen's Logrus](https://github.com/sirupsen/logrus) over the chosen log15 used. It is basically personal preference, and since I would already perform other changes, I decided to change this as well. Like I said, nothing wrong with the original log package, I'm just more comfortable with Logrus.

## Request specific capabilities

Twitch requires to request 3 capabilities to have access to data and commands in the chat. This can be easily implemented (just send the requests during connection/authentication), but since this is something so basic and fundamental, I think it makes sense to embed it to the bot "framework".

So, the following capabilities are automatically requested:
- membership
- commands
- tags

## Specific tags from incoming PRIVMSGs

Twitch sends in the PRIVMSG a lot of details about the user who sent the message. This broke the original message parser, so I rewrote it in a way that we can get all the details. This gives more options to monitor to trigger and act on them.

## Other changes

This an IRC chabot for Twitch. That means that, when in doubt, checkout the commands and standards defined by Twitch and those should be the final list. If any Twitch-specific command is not working, consider that to be a bug, and please open an issue reporting it.

# What remains the same

### Trigger functionality

This feature  is really, really well done, so it haven't changed anything. All triggers for the Hellabot should work for Hellivabot perfectly.

```go
# If message is coming/related to user otaviokr, write in chat "otaviokr said something"
var MyTrigger = hbot.Trigger{
	func (b *hbot.Bot, m *Message) bool {
		return m.From == "otaviokr"
	},
	func (b *hbot.Bot, m *hbot.Message) bool {
		b.Reply(m, "otaviokr said something")
		return false
	},
}
```

The trigger above makes the bot announce to everyone in chat that something was said by user otaviokr in the current chat. Use the code snippet below to instantiate the bot, add the trigger and start the bot (connect and join the channel).

```go
mybot, err := hbot.NewBot("irc.freenode.net:6667","hellivabot")
if err != nil {
    panic(err)
}
mybot.AddTrigger(MyTrigger)
mybot.Run() // Blocks until exit
```

# Why not merge it to Hellabot?

This is a very specific, biased implementation focused on Twitch and Twitch alone. I understand I would be shrinking Hellabot instead of contributing, enhancing or expanding it. If you think it should be better to have just one single solution, you can get the changes I did here and try to incorporate to Hellabot and submit the Pull Request - and it is up to WhyRUSleeping to accept them.
