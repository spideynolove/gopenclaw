package telegram

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/spideynolove/gopenclaw/core"
)

type Adapter struct {
	bot *tgbotapi.BotAPI
}

func New(token string) (*Adapter, error) {
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}
	return &Adapter{bot: bot}, nil
}

func (a *Adapter) Start(ctx context.Context, out chan<- core.Event) error {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := a.bot.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-ctx.Done():
				a.bot.StopReceivingUpdates()
				return
			case update := <-updates:
				if update.Message == nil {
					continue
				}

				evt := core.Event{
					SessionID: core.SessionID("telegram", update.Message.Chat.ID, update.Message.From.ID),
					ChatID:    update.Message.Chat.ID,
					UserID:    update.Message.From.ID,
					Text:      update.Message.Text,
				}

				select {
				case out <- evt:
				case <-ctx.Done():
					a.bot.StopReceivingUpdates()
					return
				}
			}
		}
	}()

	return nil
}

func (a *Adapter) Send(ctx context.Context, evt core.Event, text string) error {
	msg := tgbotapi.NewMessage(evt.ChatID, text)
	_, err := a.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	return nil
}
