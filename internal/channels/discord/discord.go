package discord

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/spideynolove/gopenclaw/core"
)

type Discord struct {
	session            *discordgo.Session
	token              string
	dmPairingEnabled   bool
	dmPairingCodes     map[string]string
	dmPairingMu        sync.RWMutex
	outChan            chan<- core.Event
	tenantID           string
}

func New(token string, tenantID string) (*Discord, error) {
	sess, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, fmt.Errorf("create discord session: %w", err)
	}

	return &Discord{
		session:          sess,
		token:            token,
		tenantID:         tenantID,
		dmPairingEnabled: true,
		dmPairingCodes:   make(map[string]string),
	}, nil
}

func (d *Discord) Start(ctx context.Context, out chan<- core.Event) error {
	d.outChan = out

	d.session.AddHandler(d.messageCreate)

	if err := d.session.Open(); err != nil {
		return fmt.Errorf("discord session open: %w", err)
	}

	go func() {
		<-ctx.Done()
		d.session.Close()
	}()

	return nil
}

func (d *Discord) Send(ctx context.Context, evt core.Event, text string) error {
	if evt.ChatID == 0 {
		return fmt.Errorf("invalid discord channel id")
	}

	_, err := d.session.ChannelMessageSend(fmt.Sprintf("%d", evt.ChatID), text)
	if err != nil {
		return fmt.Errorf("send discord message: %w", err)
	}

	return nil
}

func (d *Discord) messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}

	var chatID int64
	var channelType string

	if m.GuildID != "" {
		channelType = "discord"
		if i64, err := parseID(m.ChannelID); err == nil {
			chatID = i64
		}
	} else {
		if !d.dmPairingEnabled {
			slog.Info("ignoring dm from unknown user", "user_id", m.Author.ID)
			return
		}

		if !d.isDMPaired(m.Author.ID) {
			d.handleDMPairingRequest(s, m)
			return
		}

		channelType = "discord_dm"
		if i64, err := parseID(m.ChannelID); err == nil {
			chatID = i64
		}
	}

	userID, err := parseID(m.Author.ID)
	if err != nil {
		slog.Error("parse user id", "err", err, "user_id", m.Author.ID)
		return
	}

	sessionID := core.SessionID(d.tenantID, channelType, chatID, userID)

	evt := core.Event{
		SessionID: sessionID,
		ChatID:    chatID,
		UserID:    userID,
		Text:      m.Content,
		TenantID:  d.tenantID,
	}

	select {
	case d.outChan <- evt:
	default:
		slog.Error("event channel full", "session_id", sessionID)
	}
}

func (d *Discord) handleDMPairingRequest(s *discordgo.Session, m *discordgo.MessageCreate) {
	code := generatePairingCode()
	d.dmPairingMu.Lock()
	d.dmPairingCodes[m.Author.ID] = code
	d.dmPairingMu.Unlock()

	msg := fmt.Sprintf("To enable direct messages with this bot, please type this code in a server channel:\n`%s`", code)
	_, err := s.ChannelMessageSend(m.ChannelID, msg)
	if err != nil {
		slog.Error("send pairing code", "err", err)
	}

	go func() {
		time.Sleep(5 * time.Minute)
		d.dmPairingMu.Lock()
		delete(d.dmPairingCodes, m.Author.ID)
		d.dmPairingMu.Unlock()
	}()
}

func (d *Discord) isDMPaired(userID string) bool {
	d.dmPairingMu.RLock()
	_, exists := d.dmPairingCodes[userID]
	d.dmPairingMu.RUnlock()
	return exists
}

func (d *Discord) CompletePMPairing(userID string) bool {
	d.dmPairingMu.Lock()
	defer d.dmPairingMu.Unlock()
	_, exists := d.dmPairingCodes[userID]
	if exists {
		delete(d.dmPairingCodes, userID)
	}
	return exists
}

func generatePairingCode() string {
	const charset = "0123456789"
	code := make([]byte, 6)
	for i := range code {
		code[i] = charset[rand.Intn(len(charset))]
	}
	return string(code)
}

func parseID(id string) (int64, error) {
	var i64 int64
	_, err := fmt.Sscanf(id, "%d", &i64)
	return i64, err
}
