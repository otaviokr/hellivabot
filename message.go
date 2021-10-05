package hbot

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/sorcix/irc.v2"
)

// For more information, check PRIVMSG section at https://dev.twitch.tv/docs/irc/tags

const (
	WelcomeSignature string = ":tmi\\.twitch\\.tv 001 .+ :Welcome, GLHF!"
	JoinSignature string = ":(.+)!.+@.+\\.tmi\\.twitch\\.tv JOIN (#.+)"
	LoggedInSignature string = ":tmi\\.twitch\\.tv 376 .+ :>"
	CapabilityAcknowledgedSignature string = ":tmi\\.twitch\\.tv CAP \\* ACK :twitch\\.tv/.+"

	UNKNOWNSignature string = ":.+\\.tmi\\.twitch\\.tv (353|366) .+"
	IgnoreSignature string = ":tmi\\.twitch\\.tv (002|003|004|375|372|376) .+"
	EndOfNameListSignature string = ":.+\\.tmi\\.twitch\\.tv 366 .+ #.+ :End of /NAMES list"

	UserStateSignature string = "@badge-info=.+ :tmi\\.twitch\\.tv USERSTATE .+"
	RoomStateSignature string = "@emote-only=.+ :tmi\\.twitch\\.tv ROOMSTATE .+"

	PrivateMessageUserSignature string = ":(.+)!.+@.+\\.tmi\\.twitch\\.tv"

	PingSignature string = "PING :tmi\\.twitch\\.tv"
	NoticeSignature string = "@msg-id=(.+) :tmi\\.twitch\\.tv (NOTICE) (#.+) :(.+)"
)

type Emote struct {
	Code int
	Occurrences []int
	Size int
	Text string
}

type Message struct {
	BadgeInfo map[string]int	// Used only for subscribers. Value "subscriber/8" means user has been a subscriber for 8 months.
	Badges map[string]int		// Badges displayed next to display name. Value "admin/1" means user has the version 1 of admin badge.
	ClientNonce string			// Random identifier to link a response to a request.
	Color string				// User's defined color for their display name, if set.
	DisplayName string			// User's defined name to be displayed in the chat.
	Emotes map[string]Emote		// If the message contains emotes, they are detailed here. Check Emote type for more details.
	Flags string				//
	Id string
	Mod int
	RoomId int64
	Subscriber int
	TmiSentTs int64
	// Turbo int 				// This is deprecated. Use badges instead.
	UserId int64
	// UserType string  		// This is deprecated. Use badges instead.

	Content string				// This is the original message sent by the user or system.
	ContentNoEmotes string		// This is the message sent by the user, but all emotes have been removed.
	Unparsed map[string]string // Anything that was not parsed.

	//Time at which this message was recieved
	TimeStamp time.Time

	// Entity that this message was addressed to (channel or user)
	To string

	// Nick of the messages sender (equivalent to Prefix.Name)
	// Outdated, please use .Name
	From string

	Command string

	Params []string
}

func ParseMessage(raw string) (*Message, error) {
	if regexp.MustCompile(IgnoreSignature).MatchString(raw) ||
			regexp.MustCompile(UNKNOWNSignature).MatchString(raw) ||
			regexp.MustCompile(UserStateSignature).MatchString(raw) ||
			regexp.MustCompile(RoomStateSignature).MatchString(raw) {
		return &Message{}, nil
	}

	if regexp.MustCompile(NoticeSignature).MatchString(raw) {
		details := regexp.MustCompile(NoticeSignature).FindAllStringSubmatch(raw, -1)
		log.WithFields(
			log.Fields{
				"msg-id": details[0][1],
				"command": details[0][2],
				"channel": details[0][3],
				"content": details[0][4],
			}).Error("notice message received")
		return &Message{
			Content: details[0][4],
			Command: details[0][2],
			From: details[0][3],
		}, nil
	}

	if regexp.MustCompile(WelcomeSignature).MatchString(raw) {
		return &Message{
			Command: irc.RPL_WELCOME,
		}, nil
	}

	if regexp.MustCompile(PingSignature).MatchString(raw) {
		return &Message{
			Command: "PING",
			Content: "tmi.twitch.tv",
		}, nil
	}

	if regexp.MustCompile(JoinSignature).MatchString(raw) {
		 details := regexp.MustCompile(JoinSignature).FindAllStringSubmatch(raw, -1)
		 username := details[0][1]
		 channel := details[0][2]

		return &Message{
			Command: "JOIN",
			Content: username,
			From: channel,
		}, nil
	}

	if regexp.MustCompile(CapabilityAcknowledgedSignature).MatchString(raw) {
		return &Message{
			Command: "CAP * ACK",
		}, nil
	}



	// If it is not one of the system messages, we assume it is a regular message from one of the users.
	return ParseTwitchMessage(raw)
}
func parseTagAndVersion(raw string) (string, int) {
	details := strings.Split(raw, "/")

	if len(details) < 2 {
		return "", -1
	}

	value, err := strconv.Atoi(details[1])
	if err != nil {
		log.WithFields(
			log.Fields{
				"err": err.Error(),
				"raw": details[1],
			}).Error("failed to convert tag value")
	}
	return details[0], value
}

func ParseTwitchMessage(raw string) (*Message, error) {
	badgeInfo := map[string]int{}
	badges := map[string]int{}
	var clientNonce, color, displayName, flags, id string
	var mod, subscriber int
	var roomId, timestamp, userId int64
	emotes := map[string]Emote{}

	unparsed := map[string]string{}
	// log.Infof("Incoming: %s", raw)

	// Separate tags from the actual message
	data := strings.SplitN(raw, " ", 5)

	// Parsing tags from data[0]
	for _, entry := range strings.Split(data[0], ";") {
		tag := strings.SplitN(entry, "=", 2)
		switch tag[0] {
		case "@badge-info":
			info, months := parseTagAndVersion(tag[1])
			if len(info) > 0 {
				badgeInfo[info] = months
			}

		case "badges":
			for _, entry := range strings.Split(tag[1], ",") {
				badge, version := parseTagAndVersion(entry)
				if len(badge) > 0 {
					badges[badge] = version
				}
			}

		case "client-nonce":
			clientNonce = tag[1]

		case "color":
			color = tag[1]

		case "display-name":
			displayName = tag[1]

		case "emotes":
			log.Error("not implemented parsing for emotes")

		case "flags":
			flags = tag[1]

		case "id":
			id = tag[1]

		case "mod":
			value, err := strconv.Atoi(tag[1])
			if err != nil {
				log.WithFields(
					log.Fields{
						"err": err.Error(),
						"raw": tag[1],
					}).Error("failed to convert tag value for mod")
			}
			mod = value

		case "room-id":
			value, err := strconv.ParseInt(tag[1], 0, 64)
			if err != nil {
				log.WithFields(
					log.Fields{
						"err": err.Error(),
						"raw": tag[1],
					}).Error("failed to convert tag value for room ID")
			}
			roomId = value

		case "subscriber":
			value, err := strconv.Atoi(tag[1])
			if err != nil {
				log.WithFields(
					log.Fields{
						"err": err.Error(),
						"raw": tag[1],
					}).Error("failed to convert tag value for subscriber")
			}
			subscriber = value

		case "tmi-sent-ts":
			value, err := strconv.ParseInt(tag[1], 0, 64)
			if err != nil {
				log.WithFields(
					log.Fields{
						"err": err.Error(),
						"raw": tag[1],
					}).Error("failed to convert tag value for timestamp TMI-SENT-TS")
			}
			timestamp = value

		case "user-id":
			value, err := strconv.ParseInt(tag[1], 0, 64)
			if err != nil {
				log.WithFields(
					log.Fields{
						"err": err.Error(),
						"raw": tag[1],
					}).Error("failed to convert tag value for user ID")
			}
			userId = value

		default:
			if len(tag) > 1 {
				unparsed[tag[0]] = tag[1]
			}
		}
	}

	// Some messages have a different pattern than others
	var username, command, channel, content string
	if len(data) == 3 {
		// Parsing username from data[0]
		username = regexp.MustCompile(PrivateMessageUserSignature).FindStringSubmatch(data[0])[1]
		command = data[1]
		channel = data[2]
	} else {
		// Parsing username from data[1]
		username = regexp.MustCompile(PrivateMessageUserSignature).FindStringSubmatch(data[1])[1]
		command = data[2]
		channel = data[3]
		content = data[4][1:]
	}

	return GenerateMessageObject(command, channel, username, content, badgeInfo, badges, clientNonce, color, displayName, flags, id,
		mod, subscriber, roomId, timestamp, userId, emotes, unparsed), nil

	}

func GenerateMessageObject(command, channel, username, content string, badgeInfo, badges map[string]int, clientNonce, color, displayName, flags, id string,
	mod, subscriber int, roomId, timestamp, userId int64, emotes map[string]Emote, unparsed map[string]string) (*Message) {
	result := &Message{
		Content: content,
		From: username,
		To: channel,
		Command: command,

		BadgeInfo: badgeInfo,
		Badges: badges,
		ClientNonce: clientNonce,
		Color: color,
		DisplayName: displayName,
		Emotes: emotes,
		Flags: flags,
		Id: id,
		Mod: mod,
		RoomId: roomId,
		Subscriber: subscriber,
		TmiSentTs: timestamp,
		UserId: userId,
		Unparsed: unparsed,
	}

	log.WithFields(
		log.Fields{
			"result": result,
		}).Debug("new message parsed")

	return result
}

// Param returns the i'th parameter or the empty string if the requested element doesn't exist.
func (m *Message) Param(i int) string {
	if i < 0 || i >= len(m.Params) {
		return ""
	}
	return m.Params[i]
}
