package handler

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"html"
	"io"
	"mime"
	"mime/multipart"
	"mime/quotedprintable"
	"net"
	"net/mail"
	"net/textproto"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type crmIMAPMailboxConfig struct {
	ID            string
	UUID          pgtype.UUID
	Label         string
	Email         string
	Host          string
	Port          int32
	TLSMode       string
	Username      string
	SecretRef     string
	SMTPHost      string
	SMTPPort      int32
	SMTPTLSMode   string
	SMTPUsername  string
	SMTPSecretRef string
	OwnerType     string
	OwnerID       string
}

type crmIMAPFetchedMessage struct {
	UID         string
	MessageID   string
	InReplyTo   string
	References  []string
	Subject     string
	FromEmail   string
	FromName    string
	ToEmails    []string
	CcEmails    []string
	Date        time.Time
	BodyText    string
	BodyHTML    string
	Snippet     string
	RawSize     int
	RawHeaders  map[string][]string
	Attachments []crmEmailAttachment
}

type crmIMAPClient struct {
	raw  net.Conn
	conn *textproto.Conn
	tag  int
}

func resolveCRMIMAPSecret(ref string) (string, error) {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return "", fmt.Errorf("IMAP password is missing")
	}
	if strings.HasPrefix(ref, "inline:") {
		decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(ref, "inline:"))
		if err != nil {
			return "", fmt.Errorf("invalid inline IMAP secret")
		}
		return string(decoded), nil
	}
	if strings.HasPrefix(ref, "env:") {
		name := strings.TrimSpace(strings.TrimPrefix(ref, "env:"))
		if name == "" {
			return "", fmt.Errorf("invalid IMAP secret env ref")
		}
		value := os.Getenv(name)
		if value == "" {
			return "", fmt.Errorf("IMAP secret env var is empty")
		}
		return value, nil
	}
	// Backward compatibility for existing rows where secret_ref stored the password directly.
	return ref, nil
}

func encodeCRMIMAPInlineSecret(secret string) string {
	return "inline:" + base64.StdEncoding.EncodeToString([]byte(secret))
}

func dialCRMIMAP(cfg crmIMAPMailboxConfig) (*crmIMAPClient, error) {
	addr := net.JoinHostPort(cfg.Host, strconv.Itoa(int(cfg.Port)))
	dialer := &net.Dialer{Timeout: 15 * time.Second}
	var c net.Conn
	var err error
	if cfg.TLSMode == "ssl" {
		c, err = tls.DialWithDialer(dialer, "tcp", addr, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
	} else {
		c, err = dialer.Dial("tcp", addr)
	}
	if err != nil {
		return nil, err
	}
	client := &crmIMAPClient{raw: c, conn: textproto.NewConn(c)}
	_ = c.SetDeadline(time.Now().Add(30 * time.Second))
	if _, err := client.conn.ReadLine(); err != nil {
		_ = client.Close()
		return nil, err
	}
	_ = c.SetDeadline(time.Time{})
	if cfg.TLSMode == "starttls" {
		if err := client.simple("STARTTLS"); err != nil {
			_ = client.Close()
			return nil, err
		}
		tlsConn := tls.Client(c, &tls.Config{ServerName: cfg.Host, MinVersion: tls.VersionTLS12})
		if err := tlsConn.Handshake(); err != nil {
			_ = c.Close()
			return nil, err
		}
		client.raw = tlsConn
		client.conn = textproto.NewConn(tlsConn)
	}
	return client, nil
}

func (c *crmIMAPClient) Close() error {
	if c == nil || c.conn == nil {
		return nil
	}
	_ = c.simple("LOGOUT")
	return c.conn.Close()
}

func (c *crmIMAPClient) nextTag() string {
	c.tag++
	return fmt.Sprintf("A%04d", c.tag)
}

func (c *crmIMAPClient) simple(command string, args ...string) error {
	_, err := c.command(command, args...)
	return err
}

func (c *crmIMAPClient) command(command string, args ...string) ([]string, error) {
	if c.raw != nil {
		_ = c.raw.SetDeadline(time.Now().Add(30 * time.Second))
		defer c.raw.SetDeadline(time.Time{})
	}
	tag := c.nextTag()
	parts := append([]string{tag, command}, args...)
	if err := c.conn.PrintfLine("%s", strings.Join(parts, " ")); err != nil {
		return nil, err
	}
	var lines []string
	for {
		line, err := c.conn.ReadLine()
		if err != nil {
			return lines, err
		}
		lines = append(lines, line)
		if strings.HasPrefix(line, tag+" ") {
			upper := strings.ToUpper(line)
			if strings.Contains(upper, " OK") || strings.HasPrefix(upper, tag+" OK") {
				return lines, nil
			}
			return lines, fmt.Errorf("%s", line)
		}
	}
}

func imapQuote(value string) string {
	value = strings.ReplaceAll(value, "\\", "\\\\")
	value = strings.ReplaceAll(value, "\"", "\\\"")
	return "\"" + value + "\""
}

func fetchCRMIMAPMessages(cfg crmIMAPMailboxConfig, folder string, limit int, rangeDays int, requestedUIDs []string) ([]crmIMAPFetchedMessage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	folder = strings.TrimSpace(folder)
	if folder == "" || strings.EqualFold(folder, "inbox") {
		folder = "INBOX"
	}
	password, err := resolveCRMIMAPSecret(cfg.SecretRef)
	if err != nil {
		return nil, err
	}
	client, err := dialCRMIMAP(cfg)
	if err != nil {
		return nil, err
	}
	defer client.Close()
	if err := client.simple("LOGIN", imapQuote(cfg.Username), imapQuote(password)); err != nil {
		return nil, fmt.Errorf("IMAP login failed: %w", err)
	}
	if _, err := client.command("SELECT", imapQuote(folder)); err != nil {
		return nil, fmt.Errorf("IMAP select failed: %w", err)
	}
	uids := requestedUIDs
	if len(uids) == 0 {
		args := []string{"ALL"}
		if rangeDays > 0 {
			since := time.Now().AddDate(0, 0, -rangeDays).Format("02-Jan-2006")
			args = []string{"SINCE", since}
		}
		lines, err := client.command("UID SEARCH", args...)
		if err != nil {
			return nil, fmt.Errorf("IMAP search failed: %w", err)
		}
		uids = parseIMAPSearchUIDs(lines)
		if len(uids) > limit {
			uids = uids[len(uids)-limit:]
		}
	}
	if len(uids) == 0 {
		return []crmIMAPFetchedMessage{}, nil
	}
	// newest first for UI preview
	sort.SliceStable(uids, func(i, j int) bool { return atoiSafe(uids[i]) > atoiSafe(uids[j]) })
	messages := make([]crmIMAPFetchedMessage, 0, len(uids))
	for _, uid := range uids {
		lines, err := client.command("UID FETCH", uid, "(UID BODY.PEEK[])")
		if err != nil {
			return messages, err
		}
		raw := extractIMAPLiteral(lines)
		if strings.TrimSpace(raw) == "" {
			continue
		}
		msg := parseCRMIMAPMessage(uid, raw)
		messages = append(messages, msg)
	}
	return messages, nil
}

func parseIMAPSearchUIDs(lines []string) []string {
	for _, line := range lines {
		if strings.HasPrefix(line, "* SEARCH") {
			fields := strings.Fields(strings.TrimPrefix(line, "* SEARCH"))
			return fields
		}
	}
	return nil
}

func extractIMAPLiteral(lines []string) string {
	if len(lines) <= 1 {
		return ""
	}
	var body []string
	for _, line := range lines {
		if strings.HasPrefix(line, "*") || regexp.MustCompile(`^A\d+ `).MatchString(line) {
			continue
		}
		body = append(body, line)
	}
	return strings.Join(body, "\r\n")
}

func parseCRMIMAPMessage(uid, raw string) crmIMAPFetchedMessage {
	msg := crmIMAPFetchedMessage{UID: uid, RawSize: len(raw)}
	parsed, err := mail.ReadMessage(strings.NewReader(raw))
	if err != nil {
		msg.BodyText = raw
		msg.Snippet = makeSnippet(raw)
		return msg
	}
	decode := new(mime.WordDecoder).DecodeHeader
	msg.RawHeaders = cloneHeaderValues(parsed.Header)
	msg.MessageID = normalizeCRMMessageID(parsed.Header.Get("Message-Id"))
	if msg.MessageID == "" {
		msg.MessageID = uid
	}
	msg.InReplyTo = normalizeCRMMessageID(parsed.Header.Get("In-Reply-To"))
	msg.References = parseCRMMessageIDList(parsed.Header.Get("References"))
	msg.Subject, _ = decode(parsed.Header.Get("Subject"))
	if froms, err := parsed.Header.AddressList("From"); err == nil && len(froms) > 0 {
		msg.FromEmail = froms[0].Address
		msg.FromName, _ = decode(froms[0].Name)
	}
	msg.ToEmails = headerEmails(parsed.Header, "To")
	msg.CcEmails = headerEmails(parsed.Header, "Cc")
	if date, err := parsed.Header.Date(); err == nil {
		msg.Date = date
	}
	bodyText, bodyHTML, rawBody, attachments := parseCRMIMAPBodyParts(parsed.Header, parsed.Body)
	msg.BodyText = bodyText
	msg.BodyHTML = bodyHTML
	msg.Attachments = attachments
	if strings.TrimSpace(msg.BodyText) == "" && strings.TrimSpace(msg.BodyHTML) != "" {
		msg.BodyText = htmlToPlainText(msg.BodyHTML)
	}
	if strings.TrimSpace(msg.BodyText) == "" {
		msg.BodyText = string(rawBody)
	}
	if text, html := extractEmbeddedCRMIMAPBodies(msg.BodyText); text != "" || html != "" {
		if text != "" {
			msg.BodyText = text
		}
		if html != "" {
			msg.BodyHTML = html
		}
	}
	msg.Snippet = makeSnippet(msg.BodyText)
	return msg
}

func normalizeCRMMessageID(value string) string {
	value = strings.TrimSpace(value)
	value = strings.Trim(value, "<>")
	return strings.TrimSpace(value)
}

func parseCRMMessageIDList(value string) []string {
	return normalizeCRMMessageIDSlice(strings.Fields(value))
}

func normalizeCRMMessageIDSlice(values []string) []string {
	ids := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		id := normalizeCRMMessageID(value)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	return ids
}

func cloneHeaderValues(header mail.Header) map[string][]string {
	if len(header) == 0 {
		return map[string][]string{}
	}
	out := make(map[string][]string, len(header))
	for key, values := range header {
		copied := append([]string(nil), values...)
		out[key] = copied
	}
	return out
}

func parseCRMIMAPBodyParts(header mail.Header, body io.Reader) (string, string, []byte, []crmEmailAttachment) {
	contentType := header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		decoded, _ := io.ReadAll(body)
		return decodeTransferBody(decoded, ""), "", decoded, nil
	}
	if !strings.HasPrefix(strings.ToLower(mediaType), "multipart/") {
		raw, _ := io.ReadAll(body)
		decoded := decodeTransferBody(raw, header.Get("Content-Transfer-Encoding"))
		if strings.EqualFold(mediaType, "text/html") {
			return htmlToPlainText(decoded), decoded, raw, nil
		}
		return decoded, "", raw, nil
	}
	boundary := params["boundary"]
	mr := multipart.NewReader(body, boundary)
	var textBody, htmlBody string
	var rawBody []byte
	var attachments []crmEmailAttachment
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			break
		}
		partBody, _ := io.ReadAll(part)
		partType := part.Header.Get("Content-Type")
		decoded := decodeTransferBody(partBody, part.Header.Get("Content-Transfer-Encoding"))
		disposition := strings.ToLower(strings.TrimSpace(part.Header.Get("Content-Disposition")))
		contentID := normalizeCRMMessageID(part.Header.Get("Content-ID"))
		partMediaType, _, _ := mime.ParseMediaType(partType)
		if strings.HasPrefix(strings.ToLower(partMediaType), "multipart/") {
			nestedText, nestedHTML, nestedRaw, nestedAttachments := parseCRMIMAPBodyParts(mail.Header(part.Header), bytes.NewReader(partBody))
			if textBody == "" {
				textBody = nestedText
			}
			if htmlBody == "" {
				htmlBody = nestedHTML
			}
			if len(rawBody) == 0 {
				rawBody = nestedRaw
			}
			attachments = append(attachments, nestedAttachments...)
			continue
		}
		if strings.HasPrefix(disposition, "attachment") || contentID != "" {
			fileName := filenameFromPartHeader(part.Header, contentID)
			attachments = append(attachments, crmEmailAttachment{
				FileName:    fileName,
				LegacyName:  fileName,
				ContentType: cleanCRMEmailAttachmentContentType(partType),
				Content:     base64.StdEncoding.EncodeToString([]byte(decoded)),
				ContentID:   contentID,
				Size:        len(partBody),
				LegacySize:  len(partBody),
			})
			continue
		}
		switch {
		case strings.EqualFold(partMediaType, "text/plain") && textBody == "":
			textBody = decoded
			if len(rawBody) == 0 {
				rawBody = partBody
			}
		case strings.EqualFold(partMediaType, "text/html") && htmlBody == "":
			htmlBody = decoded
			if len(rawBody) == 0 {
				rawBody = partBody
			}
		}
	}
	if textBody == "" && htmlBody != "" {
		textBody = htmlToPlainText(htmlBody)
	}
	return textBody, htmlBody, rawBody, attachments
}

func filenameFromPartHeader(header textproto.MIMEHeader, fallback string) string {
	if value := strings.TrimSpace(header.Get("Content-Disposition")); value != "" {
		if _, params, err := mime.ParseMediaType(value); err == nil {
			if filename := strings.TrimSpace(params["filename"]); filename != "" {
				return filename
			}
		}
	}
	if value := strings.TrimSpace(header.Get("Content-Type")); value != "" {
		if _, params, err := mime.ParseMediaType(value); err == nil {
			if filename := strings.TrimSpace(params["name"]); filename != "" {
				return filename
			}
		}
	}
	if fallback != "" {
		return fallback
	}
	return "attachment"
}

func cleanCRMEmailAttachmentContentType(value string) string {
	mediaType, _, err := mime.ParseMediaType(strings.TrimSpace(value))
	if err == nil && strings.TrimSpace(mediaType) != "" {
		return strings.TrimSpace(mediaType)
	}
	if strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return "application/octet-stream"
}

func extractReadableEmailBodies(contentType string, body []byte) (string, string) {
	mediaType, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return decodeTransferBody(body, ""), ""
	}
	if strings.HasPrefix(mediaType, "multipart/") {
		mr := multipart.NewReader(bytes.NewReader(body), params["boundary"])
		var textBody, htmlBody string
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				break
			}
			partType := part.Header.Get("Content-Type")
			partBody, _ := io.ReadAll(part)
			pt, _, _ := mime.ParseMediaType(partType)
			decoded := decodeTransferBody(partBody, part.Header.Get("Content-Transfer-Encoding"))
			if strings.HasPrefix(pt, "multipart/") {
				nestedText, nestedHTML := extractReadableEmailBodies(partType, partBody)
				if textBody == "" {
					textBody = nestedText
				}
				if htmlBody == "" {
					htmlBody = nestedHTML
				}
			} else if strings.EqualFold(pt, "text/plain") && textBody == "" {
				textBody = decoded
			} else if strings.EqualFold(pt, "text/html") && htmlBody == "" {
				htmlBody = decoded
			}
		}
		return textBody, htmlBody
	}
	decoded := decodeTransferBody(body, "")
	if strings.EqualFold(mediaType, "text/html") {
		return htmlToPlainText(decoded), decoded
	}
	return decoded, ""
}

func extractEmbeddedCRMIMAPBodies(value string) (string, string) {
	if !strings.Contains(strings.ToLower(value), "content-type:") {
		return "", ""
	}
	normalized := strings.ReplaceAll(value, "\r\n", "\n")
	parts := strings.Split(normalized, "\nContent-Type:")
	if len(parts) < 2 {
		parts = strings.Split(normalized, "\ncontent-type:")
	}
	var textBody, htmlBody string
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if !strings.HasPrefix(strings.ToLower(part), "content-type:") {
			part = "Content-Type: " + part
		}
		sections := strings.SplitN(part, "\n\n", 2)
		if len(sections) != 2 {
			continue
		}
		headerText := sections[0]
		bodyText := strings.TrimSpace(sections[1])
		lines := strings.Split(headerText, "\n")
		contentType := ""
		encoding := ""
		for _, line := range lines {
			lower := strings.ToLower(strings.TrimSpace(line))
			switch {
			case strings.HasPrefix(lower, "content-type:"):
				contentType = strings.TrimSpace(line[len("Content-Type:"):])
			case strings.HasPrefix(lower, "content-transfer-encoding:"):
				encoding = strings.TrimSpace(line[len("Content-Transfer-Encoding:"):])
			}
		}
		mediaType, _, _ := mime.ParseMediaType(contentType)
		decoded := decodeTransferBody([]byte(bodyText), encoding)
		switch {
		case strings.EqualFold(mediaType, "text/plain") && textBody == "":
			textBody = decoded
		case strings.EqualFold(mediaType, "text/html") && htmlBody == "":
			htmlBody = decoded
		}
	}
	if textBody == "" && htmlBody != "" {
		textBody = htmlToPlainText(htmlBody)
	}
	return strings.TrimSpace(textBody), strings.TrimSpace(htmlBody)
}

func decodeTransferBody(body []byte, encoding string) string {
	switch strings.ToLower(strings.TrimSpace(encoding)) {
	case "base64":
		decoded, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(string(body)), ""))
		if err == nil {
			return string(decoded)
		}
	case "quoted-printable":
		decoded, err := io.ReadAll(quotedprintable.NewReader(bufio.NewReader(bytes.NewReader(body))))
		if err == nil {
			return string(decoded)
		}
	}
	return string(body)
}

func htmlToPlainText(value string) string {
	value = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`).ReplaceAllString(value, " ")
	value = regexp.MustCompile(`(?i)<br\s*/?>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?i)</p>`).ReplaceAllString(value, "\n")
	value = regexp.MustCompile(`(?s)<[^>]+>`).ReplaceAllString(value, " ")
	value = html.UnescapeString(value)
	return strings.Join(strings.Fields(value), " ")
}

func headerEmails(header mail.Header, key string) []string {
	addresses, err := header.AddressList(key)
	if err != nil {
		return []string{}
	}
	out := make([]string, 0, len(addresses))
	for _, address := range addresses {
		if address.Address != "" {
			out = append(out, address.Address)
		}
	}
	return out
}

func makeSnippet(body string) string {
	body = strings.Join(strings.Fields(body), " ")
	if len(body) > 240 {
		return body[:240]
	}
	return body
}

func atoiSafe(v string) int {
	n, _ := strconv.Atoi(v)
	return n
}
