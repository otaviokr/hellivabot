package hbot

import (
	"fmt"
	"testing"
)

func TestParseNewMessage(t *testing.T) {

	testcases := []string {
		"@badge-info=;badges=;client-nonce=6b03fa90c9f106ca101710bd5410ad15;color=;display-name=ProtocoloPardal;emotes=;flags=;id=33dafeb0-5f75-4983-a536-a5cfb2497d78;mod=0;room-id=588486943;subscriber=0;tmi-sent-ts=1620998389126;turbo=0;user-id=642302137;user-type= :protocolopardal!protocolopardal@protocolopardal.tmi.twitch.tv PRIVMSG #otavio_kr :oi",
		"@badge-info=;badges=broadcaster/1;client-nonce=fdd31b413b50dd3f59ee03e1b9f5e378;color=#1E90FF;display-name=Otavio_KR;emotes=;flags=;id=8312039c-98ee-468a-94b0-889db0727bbc;mod=0;room-id=588486943;subscriber=0;tmi-sent-ts=1620998461434;turbo=0;user-id=588486943;user-type= :otavio_kr!otavio_kr@otavio_kr.tmi.twitch.tv  PRIVMSG #otavio_kr :!bot",
		"@badge-info=;badges=moderator/1,glitchcon2020/1;color=#F51CFD;display-name=streamholics;emotes=303625154:121-130;flags=;id=ade4916b-b407-45ac-a208-00c315490b68;mod=1;room-id=588486943;subscriber=0;tmi-sent-ts=1620998745584;turbo=0;user-id=229964854;user-type=mod :streamholics!streamholics@streamholics.tmi.twitch.tv PRIVMSG #otavio_kr :Conheça (...)",
	}

	for _, tc := range testcases {
		result, err := ParseMessage(tc)
		if err != nil {
			t.Fail()
		}
		fmt.Printf("%+v\n", result)
	}
}

// func TestGenerateNewMessageObject(t *testing.T) {

// 	badgeInfo := "badgeInfo"
// 	badges := "badges"
// 	clientNonce := "clinetNonce"
// 	color := "color"
// 	displayName := "displayNames"
// 	emotes := "emotes"
// 	flags := "flags"
// 	id := "id"
// 	mod := "mod"
// 	roomId := "roomId"
// 	subscriber := "subscriber"
// 	timestamp := "timestamp"
// 	userId := "userId"

// 	result := GenerateMessageObject(badgeInfo, badges, clientNonce, color, displayName, emotes,
// 		flags, id, mod, roomId, subscriber, timestamp, userId, map[string]string{})

// 	fmt.Printf("%+v\n", result)

// }