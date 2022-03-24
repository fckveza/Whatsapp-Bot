package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	waProto "go.mau.fi/whatsmeow/binary/proto"
	waLog "go.mau.fi/whatsmeow/util/log"
)

type VEZZA struct {
	VClient *whatsmeow.Client
}

var Client VEZZA
var Log *logrus.Logger

func (vh *VEZZA) register() {
	vh.VClient.AddEventHandler(vh.MessageHandler)
}

func (vh *VEZZA) newClient(d *store.Device, l waLog.Logger) {
	vh.VClient = whatsmeow.NewClient(d, l)
}

func (vh *VEZZA) SendMessageV2(evt interface{}, msg *string) {
	v := evt.(*events.Message)
	resp := &waProto.Message{
		ExtendedTextMessage: &waProto.ExtendedTextMessage{
			Text: msg,
			ContextInfo: &waProto.ContextInfo{
				StanzaId:    &v.Info.ID,
				Participant: proto.String(v.Info.MessageSource.Sender.String()),
			},
		},
	}
	vh.VClient.SendMessage(v.Info.Sender, "", resp)
}

func (vh *VEZZA) SendTextMessage(jid types.JID, text string) {
	vh.VClient.SendMessage(jid, "", &waProto.Message{Conversation: proto.String(text)})
}

func (vh *VEZZA) MessageHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.Message:
		cok := evt.(*events.Message)
		fmt.Println(cok.Info.Chat)

		txt := strings.ToLower(v.Message.GetConversation())
		to := cok.Info.Chat
		if strings.HasPrefix(txt, "test") {
			fmt.Println("Meeek")
			vh.SendTextMessage(to, "Halloo")
		}
		return
	}
}

func main() {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	container, err := sqlstore.New("sqlite3", "file:commander.db?_foreign_keys=on", dbLog)
	if err != nil {
		panic(err)
	}

	deviceStore, err := container.GetFirstDevice()
	if err != nil {
		panic(err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	Client.newClient(deviceStore, clientLog)
	Client.register()

	if Client.VClient.Store.ID == nil {
		qrChan, _ := Client.VClient.GetQRChannel(context.Background())
		err = Client.VClient.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}

	} else {
		err = Client.VClient.Connect()
		fmt.Println("Login Success")
		if err != nil {
			panic(err)
		}
	}
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	Client.VClient.Disconnect()
}
