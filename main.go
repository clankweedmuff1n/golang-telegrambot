package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var config *Config

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	config, _ = loadConfig("config.json")

	opts := []bot.Option{
		bot.WithDefaultHandler(handler),
	}

	b, err := bot.New("5633528301:AAG2K8E-00NRa5duW18a4wxsZTYgHpZKiO0", opts...)
	if err != nil {
		panic(err)
	}

	b.Start(ctx)
}

func handler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.ChatJoinRequest != nil {
		handleJoinRequest(ctx, b, update)
	} else if update.CallbackQuery != nil {
		handleCallbackQuery(ctx, b, update)
	}
}

func handleJoinRequest(ctx context.Context, b *bot.Bot, update *models.Update) {
	request := update.ChatJoinRequest
	requestChatID := request.Chat.ID
	userID := request.From.ID

	allSubscribed := true
	var notSubscribedChats []string

	for _, targetChatID := range config.TargetChatIDs {
		member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{
			ChatID: targetChatID,
			UserID: userID,
		})

		if err != nil {
			log.Printf(fmt.Sprintf("INFO: User {%d} didn`t subscribe to chat {%s}", userID, targetChatID))
			allSubscribed = false
			notSubscribedChats = append(notSubscribedChats, targetChatID)
			continue
		}

		if member.Member != nil {
			log.Printf(fmt.Sprintf("INFO: User {%d} is subscribed to chat {%s}", userID, targetChatID))
		} else {
			allSubscribed = false
			notSubscribedChats = append(notSubscribedChats, targetChatID)
		}
	}

	if allSubscribed {
		_, err := b.ApproveChatJoinRequest(ctx, &bot.ApproveChatJoinRequestParams{
			ChatID: requestChatID,
			UserID: userID,
		})
		if err != nil {
			log.Printf(fmt.Sprintf("ERROR: Failed to approve request from user {%d}", userID))
		} else {
			log.Printf(fmt.Sprintf("INFO: User {%d} has been accepted to chat {%d}", userID, requestChatID))
		}
		return
	}

	var buttons [][]models.InlineKeyboardButton
	for _, targetChatID := range notSubscribedChats {
		targetChatInfo, err := b.GetChat(ctx, &bot.GetChatParams{
			ChatID: targetChatID,
		})

		if err != nil {
			log.Printf(fmt.Sprintf("ERROR: Can`t get info on target chat {%s}", targetChatID))
			continue
		}

		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text: targetChatInfo.Title,
				URL:  targetChatInfo.InviteLink,
			},
		})
	}

	buttons = append(buttons, []models.InlineKeyboardButton{
		{
			Text:         "Проверить подписку",
			CallbackData: fmt.Sprintf("check_sub%%%d", requestChatID),
		},
	})

	keyboard := &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      userID,
		Text:        "Чтобы ваша заявка была одобрена, подпишитесь на необходимые каналы!",
		ReplyMarkup: keyboard,
	})

	if err != nil {
		log.Printf(fmt.Sprintf("ERROR: Can`t send message to user {%d} for joining channels", userID))
	}
}

func handleCallbackQuery(ctx context.Context, b *bot.Bot, update *models.Update) {
	_, err := b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
	})

	if err != nil {
		log.Printf("WARN: Cannot answer on CallbackQuery")
	}

	splittedQuery := strings.Split(update.CallbackQuery.Data, "%")
	callbackQueryMarker := splittedQuery[0]
	requestChatID := splittedQuery[1]
	userID := update.CallbackQuery.From.ID

	if callbackQueryMarker == "check_sub" {
		_, err := b.DeleteMessage(ctx, &bot.DeleteMessageParams{
			ChatID:    userID,
			MessageID: update.CallbackQuery.Message.Message.ID,
		})

		if err != nil {
			log.Printf("WARN: Can`t delete previous message")
		}

		var buttons [][]models.InlineKeyboardButton
		allSubscribed := true

		for _, targetChatID := range config.TargetChatIDs {
			member, err := b.GetChatMember(ctx, &bot.GetChatMemberParams{
				ChatID: targetChatID,
				UserID: userID,
			})

			if err != nil {
				log.Printf("WARN: Can`t get chat {%s} member {%d} info", targetChatID, userID)
			}

			if member.Member == nil {
				log.Printf(fmt.Sprintf("INFO: User {%d} has not entered the chat {%s}", userID, targetChatID))
				allSubscribed = false

				targetChatInfo, err := b.GetChat(ctx, &bot.GetChatParams{
					ChatID: targetChatID,
				})

				if err != nil {
					log.Printf("WARN: Can`t get taget chat {%s} info", targetChatID)
				}

				buttons = append(buttons, []models.InlineKeyboardButton{
					{
						Text: targetChatInfo.Title,
						URL:  targetChatInfo.InviteLink,
					},
				})
			}
		}

		if !allSubscribed {
			buttons = append(buttons, []models.InlineKeyboardButton{
				{
					Text:         "Проверить подписку",
					CallbackData: fmt.Sprintf("check_sub%%%s", requestChatID),
				},
			})

			keyboard := &models.InlineKeyboardMarkup{
				InlineKeyboard: buttons,
			}

			_, err := b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID:      userID,
				Text:        "Чтобы ваша заявка была одобрена, подпишитесь на необходимые каналы!",
				ReplyMarkup: keyboard,
			})

			if err != nil {
				log.Printf(fmt.Sprintf("ERROR: Can`t send message to user {%d} for joining channels", userID))
			}
		} else {
			_, err := b.ApproveChatJoinRequest(ctx, &bot.ApproveChatJoinRequestParams{
				ChatID: requestChatID,
				UserID: userID,
			})

			if err != nil {
				log.Printf(fmt.Sprintf("ERROR: Failed to approve request from user {%d}", userID))
			} else {
				log.Printf(fmt.Sprintf("INFO: User {%d} has been accepted to chat {%s}", userID, requestChatID))
			}

			_, err = b.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: userID,
				Text:   "Ваша заявка была принята",
			})

			if err != nil {
				log.Printf("WARN: Can`t send greeting message")
			}
		}
	}
}
