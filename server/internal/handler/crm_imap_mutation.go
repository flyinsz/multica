package handler

import (
	"fmt"
	"strings"
	"time"
)

func (c *crmIMAPClient) appendMessage(folder string, raw []byte, flags []string, when time.Time) error {
	folder = strings.TrimSpace(folder)
	if folder == "" {
		folder = "Sent"
	}
	if len(raw) == 0 {
		return fmt.Errorf("empty IMAP APPEND payload")
	}
	flagList := "()"
	if len(flags) > 0 {
		flagList = "(" + strings.Join(flags, " ") + ")"
	}
	dateValue := when.Format("02-Jan-2006 15:04:05 -0700")
	return c.commandLiteral("APPEND", []string{imapQuote(folder), flagList, imapQuote(dateValue)}, raw)
}

func (c *crmIMAPClient) storeUIDFlags(folder string, uid string, addFlags []string, removeFlags []string) error {
	folder = strings.TrimSpace(folder)
	if folder == "" || strings.EqualFold(folder, "inbox") {
		folder = "INBOX"
	}
	uid = strings.TrimSpace(uid)
	if uid == "" {
		return nil
	}
	passwordSafeUID := strings.Trim(uid, " ")
	if _, err := c.command("SELECT", imapQuote(folder)); err != nil {
		return err
	}
	if len(addFlags) > 0 {
		if _, err := c.command("UID STORE", passwordSafeUID, "+FLAGS.SILENT", "("+strings.Join(addFlags, " ")+")"); err != nil {
			return err
		}
	}
	if len(removeFlags) > 0 {
		if _, err := c.command("UID STORE", passwordSafeUID, "-FLAGS.SILENT", "("+strings.Join(removeFlags, " ")+")"); err != nil {
			return err
		}
	}
	return nil
}

func (c *crmIMAPClient) commandLiteral(command string, args []string, literal []byte) error {
	if c.raw != nil {
		_ = c.raw.SetDeadline(time.Now().Add(45 * time.Second))
		defer c.raw.SetDeadline(time.Time{})
	}
	tag := c.nextTag()
	parts := append([]string{tag, command}, args...)
	line := strings.Join(parts, " ") + fmt.Sprintf(" {%d}", len(literal))
	if err := c.conn.PrintfLine("%s", line); err != nil {
		return err
	}
	for {
		line, err := c.conn.ReadLine()
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, "+") {
			break
		}
		if strings.HasPrefix(line, tag+" ") {
			return fmt.Errorf("%s", line)
		}
	}
	payload := append([]byte{}, literal...)
	if !bytesHasCRLFLineEndings(payload) {
		payload = []byte(strings.ReplaceAll(string(payload), "\n", "\r\n"))
	}
	payload = append(payload, []byte("\r\n")...)
	if _, err := c.raw.Write(payload); err != nil {
		return err
	}
	for {
		line, err := c.conn.ReadLine()
		if err != nil {
			return err
		}
		if strings.HasPrefix(line, tag+" ") {
			upper := strings.ToUpper(line)
			if strings.Contains(upper, " OK") || strings.HasPrefix(upper, tag+" OK") {
				return nil
			}
			return fmt.Errorf("%s", line)
		}
	}
}

func bytesHasCRLFLineEndings(value []byte) bool {
	for i, b := range value {
		if b == '\n' && (i == 0 || value[i-1] != '\r') {
			return false
		}
	}
	return true
}

func appendCRMIMAPSentMessage(cfg crmIMAPMailboxConfig, raw []byte, sentAt time.Time) error {
	password, err := resolveCRMIMAPSecret(cfg.SecretRef)
	if err != nil {
		return err
	}
	client, err := dialCRMIMAP(cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.simple("LOGIN", imapQuote(cfg.Username), imapQuote(password)); err != nil {
		return fmt.Errorf("IMAP login failed: %w", err)
	}
	for _, folder := range []string{"Sent", "Sent Mail", "INBOX.Sent", "[Gmail]/Sent Mail"} {
		if err := client.appendMessage(folder, raw, []string{"\\Seen"}, sentAt); err == nil {
			return nil
		}
	}
	return fmt.Errorf("failed to append sent message to common Sent folders")
}

func syncCRMIMAPThreadFlags(cfg crmIMAPMailboxConfig, folder string, uid string, isRead *bool, isStarred *bool) error {
	password, err := resolveCRMIMAPSecret(cfg.SecretRef)
	if err != nil {
		return err
	}
	client, err := dialCRMIMAP(cfg)
	if err != nil {
		return err
	}
	defer client.Close()
	if err := client.simple("LOGIN", imapQuote(cfg.Username), imapQuote(password)); err != nil {
		return fmt.Errorf("IMAP login failed: %w", err)
	}
	addFlags := []string{}
	removeFlags := []string{}
	if isRead != nil {
		if *isRead {
			addFlags = append(addFlags, "\\Seen")
		} else {
			removeFlags = append(removeFlags, "\\Seen")
		}
	}
	if isStarred != nil {
		if *isStarred {
			addFlags = append(addFlags, "\\Flagged")
		} else {
			removeFlags = append(removeFlags, "\\Flagged")
		}
	}
	return client.storeUIDFlags(folder, uid, addFlags, removeFlags)
}
