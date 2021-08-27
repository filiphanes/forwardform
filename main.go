package main

import (
	"fmt"
	"log"
	"net"
	"net/smtp"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

var (
	from         = ""
	smtpUser     = ""
	smtpPassword = ""
	smtpHost     = ""
	smtpHostName = ""
	smtpPort     = ""
)

func getenv(name string, whenempty string) string {
	val := os.Getenv(name)
	if val == "" {
		return whenempty
	}
	return val
}

func main() {
	// This function will be called by the server for each incoming request.
	//
	// RequestCtx provides a lot of functionality related to http request
	// processing. See RequestCtx docs for details.
	from = getenv("SMTP_FROM", "forwardemail@hanes.tech")
	smtpUser = getenv("SMTP_USER", "forwardemail@hanes.tech")
	smtpPassword = os.Getenv("SMTP_PASSWORD")
	smtpHost = getenv("SMTP_HOST", "127.0.0.1:1025")
	var err error
	smtpHostName, smtpPort, err = net.SplitHostPort(smtpHost)
	if err != nil {
		log.Println("Invalid SMTP_HOST", err)
	}

	requestHandler := func(ctx *fasthttp.RequestCtx) {
		switch string(ctx.Path()) {
		case "/form":
			formHandler(ctx)
		case "/":
			rootHandler(ctx)

		default:
			ctx.Error("Unknown path", 404)
		}
		log.Println(ctx.Response.StatusCode(), string(ctx.Method()), string(ctx.Path()))
	}

	s := &fasthttp.Server{
		Handler: requestHandler,
	}

	// ListenAndServe returns only on error, so usually it blocks forever.
	if err := s.ListenAndServe("127.0.0.1:8000"); err != nil {
		log.Fatalf("error in ListenAndServe: %s", err)
	}
}

func formHandler(ctx *fasthttp.RequestCtx) {
	var to string

	referer := string(ctx.Request.Header.Peek("Referer"))
	if referer == "" {
		log.Println("Empty referer")
		ctx.Redirect("/", 307)
		return
	}

	u, err := url.Parse(referer)
	if err != nil {
		log.Println("Referer parse err", err)
		ctx.Redirect(referer, 307)
		return
	}
	hostname := u.Hostname()
	// DNS lookup referer host
	txts, err := net.LookupTXT(hostname)
	if err != nil {
		if hostname == "127.0.0.1" {
			// Testing
			to = from
		} else {
			log.Println("LookupTXT err", err)
			ctx.Redirect(referer, 307)
			return
		}
	}
	for _, txt := range txts {
		if strings.HasPrefix(txt, "forwardform=") {
			to = strings.TrimPrefix(txt, "forwardform=")
			break
		}
	}
	if to == "" {
		log.Println("Empty to")
		ctx.Redirect(referer, 307)
		return
	}

	msg := fmt.Sprintf(`Date: %v
From: %s
To: %s
Subject: Form from %s
Content-Type: text/plain

Hi,
you recieved form from %s

`,
		time.Now().UTC().Format(time.RFC1123),
		from,
		to,
		hostname,
		referer)

	appendArg := func(key, value []byte) {
		msg += string(key) + ": " + string(value) + "\n"
	}
	ctx.QueryArgs().VisitAll(appendArg)
	ctx.PostArgs().VisitAll(appendArg)

	err = smtp.SendMail(
		smtpHost,
		smtp.PlainAuth("", smtpUser, smtpPassword, smtpHostName),
		from,
		[]string{to},
		[]byte(msg),
	)

	if err != nil {
		log.Printf("smtp error: %s", err)
		return
	}

	log.Println("Sent to", to)
	ctx.Redirect(referer, 307)
}

func rootHandler(ctx *fasthttp.RequestCtx) {
	fmt.Fprintf(ctx, `<form action="/form" method="POST">
<input name="title" value="Sir" />
<input name="name" value="Norbert" />
<input name="surname" value="Surikata" />
<input name="age" value="100" />
<input name="height" value="200" />
<input name="weight" value="300" />
<input name="speed" value="400" />
<input type="submit" value="Send" />
</form>`)
	ctx.SetContentType("text/html")
	ctx.SetStatusCode(200)
}
