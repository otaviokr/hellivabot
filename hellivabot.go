package hbot

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/sorcix/irc.v2"
)

// Bot implements an irc bot to be connected to a given server
type Bot struct {

	// This is set if we have hijacked a connection
	reconnecting bool
	// Channel for user to read incoming messages
	Incoming chan *Message
	con      net.Conn
	outgoing chan string
	handlers []Handler
	// When did we start? Used for uptime
	started time.Time
	// Unix domain abstract socket address for reconnects (linux only)
	unixastr string
	// Unix domain socket address for other Unixes
	unixsock string
	unixlist net.Listener
	// Log15 loggger
	log.Logger
	didJoinChannels sync.Once

	// sasl handler
	sasl    *saslAuth
	addSASL sync.Once

	// Exported fields
	Host          string
	Password      string
	Channels      []string
	SSL           bool
	SASL          bool
	HijackSession bool
	// An optional function that connects to an IRC server over plaintext:
	Dial func(network, addr string) (net.Conn, error)
	// An optional function that connects to an IRC server over a secured connection:
	DialTLS func(network, addr string, tlsConf *tls.Config) (*tls.Conn, error)
	// This bots nick
	Nick string
	// Duration to wait between sending of messages to avoid being
	// kicked by the server for flooding (default 200ms)
	ThrottleDelay time.Duration
	// Maxmimum time between incoming data
	PingTimeout time.Duration

	TLSConfig tls.Config
}

// NewBot creates a new instance of Bot
func NewBot(host, nick string, options ...func(*Bot)) (*Bot, error) {
	// Defaults are set here
	bot := Bot{
		Incoming:      make(chan *Message, 16),
		outgoing:      make(chan string, 16),
		started:       time.Now(),
		unixastr:      fmt.Sprintf("@%s-%s/bot", host, nick),
		unixsock:      fmt.Sprintf("/tmp/%s-%s-bot.sock", host, nick),
		sasl:          &saslAuth{},
		Host:          host,
		Nick:          nick,
		ThrottleDelay: 200 * time.Millisecond,
		PingTimeout:   300 * time.Second,
		HijackSession: false,
		SSL:           false,
		SASL:          false,
		Channels:      []string{"#test"},
		Password:      "",
	}
	for _, option := range options {
		option(&bot)
	}
	// Discard logs by default
	// bot.Logger = log.New("id", logext.RandId(8), "host", bot.Host, "nick", log.Lazy{bot.getNick})

	// bot.Logger.SetHandler(log.DiscardHandler())
	bot.AddTrigger(pingPong)
	bot.AddTrigger(joinChannels)
	return &bot, nil
}

func (bot *Bot) String() string {
	return fmt.Sprintf("Server: %s, Channels: %v, Nick: %s", bot.Host, bot.Channels, bot.Nick)
}

// Uptime returns the uptime of the bot
func (bot *Bot) Uptime() string {
	return fmt.Sprintf("Started: %s, Uptime: %s", bot.started, time.Since(bot.started))
}

// WaitFor will block until a message matching the given filter is received
func (bot *Bot) WaitFor(filter func(*Message) bool) {
	for mes := range bot.Incoming {
		if filter(mes) {
			return
		}
	}
}

// StandardRegistration performsa a basic set of registration commands
func (bot *Bot) StandardRegistration() {
	//Server registration
	if bot.Password != "" {
		bot.Send("PASS " + bot.Password)
	}
	// bot.Debug("Sending standard registration")
	log.Debug("sending standard registration")
	bot.sendUserCommand(bot.Nick, bot.Nick)
	bot.SetNick(bot.Nick)
}

// SetNick sets the bots nick on the irc server
func (bot *Bot) SetNick(nick string) {
	bot.Nick = nick
	bot.Send(fmt.Sprintf("NICK %s", nick))
}

// Run starts the bot and connects to the server. Blocks until we disconnect from the server.
func (bot *Bot) Run() {
	// bot.Debug("Starting bot goroutines")
	log.Debug("starting bot goroutines")

	// Attempt reconnection
	var hijack bool
	if bot.HijackSession {
		if bot.SSL {
			// bot.Crit("Can't Hijack a SSL connection")
			log.Fatal("cannot hijack a SSL connection")
			return
		}
		hijack = bot.hijackSession()
		// bot.Debug("Hijack", "Did we?", hijack)
		log.WithFields(
			log.Fields{
				"hijack": hijack,
			}).Debug("did we hijack?")
	}

	if !hijack {
		err := bot.connect(bot.Host)
		if err != nil {
			// bot.Crit("bot.Connect error", "err", err.Error())
			log.WithFields(
				log.Fields{
					"err": err.Error(),
					"host": bot.Host,
				}).Fatal("failed to connect")
			return
		}
		// bot.Info("Connected successfully!")
		log.Info("connected successfully")
	}

	go bot.handleIncomingMessages()
	go bot.handleOutgoingMessages()

	go bot.StartUnixListener()

	// Only register on an initial connection
	if !bot.reconnecting {
		if bot.SASL {
			bot.SASLAuthenticate(bot.Nick, bot.Password)
		} else {
			bot.StandardRegistration()
		}
	}
	for m := range bot.Incoming {
		if m == nil {
			log.Info("Disconnected")
			return
		}
	}
}

// Reply sends a message to where the message came from (user or channel)
func (bot *Bot) Reply(m *Message, text string) {
	var target string
	if strings.Contains(m.To, "#") {
		target = m.To
	} else {
		target = m.From
	}
	bot.Msg(target, text)
}

// Msg sends a message to 'who' (user or channel)
func (bot *Bot) Msg(who, text string) {
	for _, line := range splitText(text) {
		bot.Send("PRIVMSG " + who + " :" + line)
	}
}

// Notice sends a NOTICE message to 'who' (user or channel)
func (bot *Bot) Notice(who, text string) {
	for _, line := range splitText(text) {
		bot.Send("NOTICE " + who + " :" + line)
	}
}

// Action sends an action to 'who' (user or channel)
func (bot *Bot) Action(who, text string) {
	msg := fmt.Sprintf("\u0001ACTION %s\u0001", text)
	bot.Msg(who, msg)
}

// Topic sets the channel 'c' topic (requires bot has proper permissions)
func (bot *Bot) Topic(c, topic string) {
	str := fmt.Sprintf("TOPIC %s :%s", c, topic)
	bot.Send(str)
}

// Send any command to the server
func (bot *Bot) Send(command string) {
	bot.outgoing <- command
}

// ChMode is used to change users modes in a channel
// operator = "+o" deop = "-o"
// ban = "+b"
func (bot *Bot) ChMode(user, channel, mode string) {
	bot.Send("MODE " + channel + " " + mode + " " + user)
}

// Join a channel
func (bot *Bot) Join(ch string) {
	bot.Send("JOIN " + ch)
}

// Part a channel. Notice that Twitch does not expect a leave message, like default IRC.
func (bot *Bot) Part(ch string) {
	bot.Send("PART " + ch)
}

// Close closes the bot
func (bot *Bot) Close() error {
	if bot.unixlist != nil {
		return bot.unixlist.Close()
	}
	return nil
}

// AddTrigger adds a trigger to the bot's handlers
func (bot *Bot) AddTrigger(h Handler) {
	bot.handlers = append(bot.handlers, h)
}

// Handler is used to subscribe and react to events on the bot Server
type Handler interface {
	Handle(*Bot, *Message) bool
}

// Trigger is a Handler which is guarded by a condition
type Trigger struct {
	// Returns true if this trigger applies to the passed in message
	Condition func(*Bot, *Message) bool

	// The action to perform if Condition is true
	// return true if the message was 'consumed'
	Action func(*Bot, *Message) bool
}

// Handle executes the trigger action if the condition is satisfied
func (t Trigger) Handle(b *Bot, m *Message) bool {
	return t.Condition(b, m) && t.Action(b, m)
}

// A trigger to respond to the servers ping pong messages
// If PingPong messages are not responded to, the server assumes the
// client has timed out and will close the connection.
// Note: this is automatically added in the IrcCon constructor
var pingPong = Trigger{
	Condition: func(bot *Bot, m *Message) bool {
		return m.Command == "PING"
	},
	Action: func(bot *Bot, m *Message) bool {
		bot.Send("PONG :" + m.Content)
		return true
	},
}

var joinChannels = Trigger{
	Condition: func(bot *Bot, m *Message) bool {
		return m.Command == irc.RPL_WELCOME || m.Command == irc.RPL_ENDOFMOTD // 001 or 372
	},
	Action: func(bot *Bot, m *Message) bool {
		bot.didJoinChannels.Do(func() {
			bot.Send("CAP REQ :twitch.tv/membership")
			bot.Send("CAP REQ :twitch.tv/tags")
			bot.Send("CAP REQ :twitch.tv/commands")

			for _, channel := range bot.Channels {
				splitchan := strings.SplitN(channel, ":", 2)
				if len(splitchan) == 2 {
					channel = splitchan[0]
					password := splitchan[1]
					bot.Send(fmt.Sprintf("JOIN %s %s", channel, password))
				} else {
					bot.Send(fmt.Sprintf("JOIN %s", channel))
				}
			}
		})
		return true
	},
}

func SaslAuth(pass string) func(*Bot) {
	return func(b *Bot) {
		b.SASL = true
		b.Password = pass
	}
}

func ReconOpt() func(*Bot) {
	return func(b *Bot) {
		b.HijackSession = true
	}
}

// func (bot *Bot) getNick() string {
// 	return bot.Nick
// }

func (bot *Bot) connect(host string) (err error) {
	// bot.Debug("Connecting")
	log.WithFields(
		log.Fields{
			"host": host,
		}).Debug("connecting to server")
	dial := bot.Dial
	if dial == nil {
		dial = net.Dial
	}
	dialTLS := bot.DialTLS
	if dialTLS == nil {
		dialTLS = tls.Dial
	}

	if bot.SSL {
		bot.con, err = dialTLS("tcp", host, &bot.TLSConfig)
	} else {
		bot.con, err = dial("tcp", host)
	}
	return err
}

// Incoming message gathering routine
func (bot *Bot) handleIncomingMessages() {
	scan := bufio.NewScanner(bot.con)
	for scan.Scan() {
		// Disconnect if we have seen absolutely nothing for 300 seconds
		bot.con.SetDeadline(time.Now().Add(bot.PingTimeout))
		msg, err := ParseMessage(scan.Text())
		if err != nil {
			log.WithFields(
				log.Fields{
					"err": err.Error(),
				}).Error("failed to parse incoming message")
		}
		// bot.Debug("Incoming", "raw", scan.Text(), "msg.To", msg.To, "msg.From", msg.From, "msg.Params", msg.Params, "msg.Trailing", msg.Trailing())
		log.WithFields(
			log.Fields{
				"raw": scan.Text(),
				"msg.To": msg.To,
				"msg.From": msg.From,
				// "msg.Params": msg.Params,
				// "msg.Trailing", msg.Trailing(),
			}).Debug("incoming message")
		go func() {
			for _, h := range bot.handlers {
				if h.Handle(bot, msg) {
					break
				}
			}
		}()
		bot.Incoming <- msg
	}
	close(bot.Incoming)
}

// Handles message speed throtling
func (bot *Bot) handleOutgoingMessages() {
	for s := range bot.outgoing {
		// bot.Debug("Outgoing", "data", s)
		log.WithFields(
			log.Fields{
				"data": s,
			}).Debug("outgoing message")
		_, err := fmt.Fprint(bot.con, s+"\r\n")
		if err != nil {
			// bot.Error("handleOutgoingMessages fmt.Fprint error", "err", err)
			log.WithFields(
				log.Fields{
					"err": err.Error(),
				}).Error("failed to handle outgoing message")
			return
		}
		time.Sleep(bot.ThrottleDelay)
	}
}

// Set username, real name, and mode
func (bot *Bot) sendUserCommand(user, realname string) {
	bot.Send(fmt.Sprintf("USER %s 0 * :%s", user, realname))
}

// Splits a given string into a string slice, in chunks ending
// either with \n, or with \r\n, or of a size of 400 characters.
func splitText(text string) []string {
	var ret []string
	scanner := bufio.NewScanner(strings.NewReader(text))
	for scanner.Scan() {
		line := scanner.Text()
		for len(line) > 400 {
			ret = append(ret, line[:400])
			line = line[400:]
		}
		ret = append(ret, line)
	}
	return ret
}
