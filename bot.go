package main

import (
	"fmt"
	"log"
	"io"
	"net/http"
	"strings"
	"time"
	"encoding/base64"

	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v7/linebot"
)

var blob *messaging_api.MessagingApiBlobAPI

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			// Handle only on text message
			case *linebot.TextMessage:
				if !isGroupEvent(event) {
					return
				}

				if strings.HasPrefix(message.Text, "帥狗畫") {
					handleDraw(event, message.Text)
				} else if strings.EqualFold(message.Text, "帥狗總結一下") {
					handleSumAll(event)
				} else if strings.HasPrefix(message.Text, "帥狗 ") {
					handleReply(event, message.Text)
				} else {
					handleStoreMsg(event, message.Text)
				}

			// Handle only on Sticker message
			case *linebot.StickerMessage:
				var kw string
				for _, k := range message.Keywords {
					kw = kw + "," + k
				}

				log.Println("Sticker: PID=", message.PackageID, " SID=", message.StickerID, "keyword:", kw, "text:", message.Text)

				if isGroupEvent(event) {
					outStickerResult := fmt.Sprintf("貼圖訊息: %s ", kw)
					handleStoreMsg(event, outStickerResult)
				}
			case *linebot.ImageMessage:
				fmt.Printf("%+v\n", message)
				content, err := blob.GetMessageContent(message.ID)
				if err != nil {
					log.Println("Got GetMessageContent err:", err)
				}
				defer content.Body.Close()
				data, err := io.ReadAll(content.Body)
				if err != nil {
					log.Fatal(err)
				}
				handleReplyImage(event, base64.StdEncoding.EncodeToString(data))
			}
		}
	}
}

func handleSumAll(event *linebot.Event) {
	// Scroll through all the messages in the chat group (in chronological order).
	oriContext := ""
	q := summaryQueue.ReadGroupInfo(getGroupID(event))
	for _, m := range q {
		// [xxx]: ...
		userName, _ := bot.GetProfile(m.UserName).Do()
		oriContext = oriContext + fmt.Sprintf("[%s]: %s . %s\n", userName, m.MsgText, m.Time.Local().UTC().Format("2006-01-02 15:04:05"))
	}

	userName := event.Source.UserID
	userProfile, err := bot.GetProfile(event.Source.UserID).Do()
	if err == nil {
		userName = userProfile.DisplayName
	}

	if oriContext == "" {
		if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("沒事情總結了Ｒ "+userName)).Do(); err != nil {
			log.Print(err)
		}
		return
	}

	oriContext = fmt.Sprintf("幫我總結成三點 `%s`", oriContext)
	reply := gptGPT3CompleteContext(oriContext)
	log.Println(oriContext)

	if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
		log.Print(err)
	}
}

func handleReply(event *linebot.Event, askStr string) {
	// Scroll through all the messages in the chat group (in chronological order).
	oriContext := ""
	q := summaryQueue.ReadGroupInfo(getGroupID(event))
	for _, m := range q {
		// [xxx]: ...
		oriContext = oriContext + fmt.Sprintf("[%s]: %s . %s\n", m.UserName, m.MsgText, m.Time.Local().UTC().Format("2006-01-02 15:04:05"))
	}

	userName := event.Source.UserID
	userProfile, err := bot.GetProfile(event.Source.UserID).Do()
	if err == nil {
		userName = userProfile.DisplayName
	}

	askStr = strings.TrimPrefix(askStr, "帥狗 ")
	log.Print(askStr)

	// prompt
	oriContext = fmt.Sprintf("你是隻有著帥氣假髮的哈吧狗，大家都叫你帥狗，我是`%s`, 我想跟你說：`%s`,你在一個聊天的群組裡，請在群組中回覆我一些話，簡短並帶點幽默，語氣可以隨便一點，控制在兩句內，這是這個群組目前的對話內容：`%s`，你是群組中的帥狗，直接回覆你要說的就好不用加上自己的名稱", userName, askStr, oriContext)
	reply := gptGPT3CompleteContext(oriContext)
	log.Print(oriContext)

	m := MsgDetail{
		MsgText:  reply,
		UserName: "帥狗",
		Time:     time.Now(),
	}

	summaryQueue.AppendGroupInfo(getGroupID(event), m)

	if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
		log.Print(err)
	}
}

func handleReplyImage(event *linebot.Event, image string) {
	// prompt
	oriContext := fmt.Sprintf("給你一張圖片，針對這張圖片的細節稱讚他長得很好看：")
	reply := gptGPT3CompleteContextImage(oriContext, image)
	log.Print(oriContext, image)

	if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(reply)).Do(); err != nil {
		log.Print(err)
	}
}

func handleDraw(event *linebot.Event, message string) {
	message = strings.TrimPrefix(message, "帥狗畫")
	prompt := fmt.Sprintf("用簡單的線條，童趣且富有色彩的畫出`%s`", message)

	if reply, err := gptImageCreate(prompt); err != nil {
		if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("帥狗畫不出來")).Do(); err != nil {
			log.Print(err)
		}
	} else {
		if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("帥狗畫好啦："), linebot.NewImageMessage(reply, reply)).Do(); err != nil {
			log.Print(err)
		}
	}
}

func handleStoreMsg(event *linebot.Event, message string) {
	// Get user display name. (It is nick name of the user define.)
	userName := event.Source.UserID
	userProfile, err := bot.GetProfile(event.Source.UserID).Do()
	if err == nil {
		userName = userProfile.DisplayName
	}

	m := MsgDetail{
		MsgText:  message,
		UserName: userName,
		Time:     time.Now(),
	}
	summaryQueue.AppendGroupInfo(getGroupID(event), m)
}

func isGroupEvent(event *linebot.Event) bool {
	return event.Source.GroupID != "" || event.Source.RoomID != ""
}

func getGroupID(event *linebot.Event) string {
	if event.Source.GroupID != "" {
		return event.Source.GroupID
	} else if event.Source.RoomID != "" {
		return event.Source.RoomID
	}

	return ""
}
