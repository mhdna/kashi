package api

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	db "github.com/mhdna/kashi/db/sqlc"
)

// A client whose referral code a friend used gets a WhatsApp once the
// friend's sale reaches kashi (after checkout, never when the cashier only
// types the code): the reward they earned, and until when.

const referralRewardTemplate = "referral_reward"

// referralReward is what the owner of a code is told about.
type referralReward struct {
	BranchID    int64
	Owner       db.Client
	Percent     int16
	ExpiresAt   sql.NullTime
	InvoiceCode string
}

// referralRewardText is the message: the owner's first name, the reward, the
// branch, and the last day it can be used when it ends.
func referralRewardText(ownerName, branchName string, percent int16, expires sql.NullTime) string {
	name := strings.TrimSpace(ownerName)
	if fields := strings.Fields(name); len(fields) > 0 {
		name = fields[0]
	}
	hello := "Hello"
	if name != "" {
		hello += " " + name
	}
	text := fmt.Sprintf("%s! A friend just shopped at %s with your referral code. You've earned %d%% off your next purchase.", hello, branchName, percent)
	if expires.Valid {
		// expires_at is the first moment it is over: the day before is the last.
		text += " Use it by " + expires.Time.Add(-time.Second).Format("2 Jan 2006") + "."
	}
	return text + " Thank you for sharing!"
}

// notifyReferralReward sends the owner their reward message in the
// background, so a sync never waits on WhatsApp. Tests replace it.
func (server *Server) notifyReferralReward(r referralReward) {
	if server.referralNotifier != nil {
		server.referralNotifier(r)
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := server.sendReferralRewardWhatsApp(ctx, r); err != nil {
			log.Printf("whatsapp referral reward for %s: %v", r.InvoiceCode, err)
		}
	}()
}

// sendReferralRewardWhatsApp sends the message through the branch's WhatsApp
// number, when WhatsApp is on (overall and for the branch) and the owner has
// a phone, and logs how it went on the WhatsApp page.
func (server *Server) sendReferralRewardWhatsApp(ctx context.Context, r referralReward) error {
	to := whatsAppDigits(r.Owner.Phone)
	if to == "" {
		return nil
	}
	settings, err := server.store.GetWhatsAppSettings(ctx)
	if err != nil {
		return err
	}
	if !settings.WhatsappEnabled {
		return nil
	}
	branch, err := server.store.GetBranch(ctx, r.BranchID)
	if err != nil {
		return err
	}
	if !branch.WhatsappEnabled {
		return nil
	}
	entry := db.CreateWhatsAppMessageParams{
		BranchID:     sql.NullInt64{Int64: r.BranchID, Valid: true},
		InvoiceCode:  r.InvoiceCode,
		ClientName:   strings.TrimSpace(r.Owner.Name),
		ClientPhone:  to,
		TemplateName: referralRewardTemplate,
		Sender:       whatsAppSenderBridge,
		Body:         referralRewardText(r.Owner.Name, branch.Name, r.Percent, r.ExpiresAt),
	}
	logIt := func(status, reason string) error {
		entry.Status, entry.Reason = status, reason
		_, err := server.store.CreateWhatsAppMessage(ctx, entry)
		return err
	}
	sender, err := server.store.GetWhatsAppSenderSettings(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(sender.WhatsappBridgeKey) == "" {
		return logIt(whatsAppStatusFailed, "WhatsApp bridge not set up (Settings > WhatsApp)")
	}
	// The message isn't about a sale's size: any number of the branch's.
	number, err := server.store.WhatsAppNumberForBranch(ctx, db.WhatsAppNumberForBranchParams{BranchID: r.BranchID, TotalCents: 1 << 40})
	if errors.Is(err, sql.ErrNoRows) {
		return logIt(whatsAppStatusFailed, "no WhatsApp number for this branch (Settings > WhatsApp)")
	}
	if err != nil {
		return err
	}
	entry.WhatsappNumberID = sql.NullInt64{Int64: number.ID, Valid: true}
	entry.BridgeRef = newReviewCode()
	body := map[string]any{"number": to, "text": entry.Body, "source": "kashi", "force": true, "ref": entry.BridgeRef}
	var out struct {
		Status string `json:"status"`
	}
	if err := whatsAppBridgeCall(ctx, sender, http.MethodPost, "/message/sendText/"+url.PathEscape(number.Instance), body, &out); err != nil {
		return logIt(whatsAppStatusFailed, whatsAppNumberDownError(number.Label, err).Error())
	}
	return logIt(whatsAppStatusSent, "")
}
