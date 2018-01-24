package pop3client

import (
	"strings"
	"github.com/go-pop3"
	"github.com/helioina/api/log"
	"net/smtp"
	"net/mail"
	"fmt"
	"bytes"
)

func GetMessages(pop3Address, username , password string){
	c, err := pop3.Dial( pop3Address , pop3.UseTimeout(0) )
	if err != nil{
		log.LogError(err.Error())
	}

	if err := c.User(username); err != nil {
		log.LogError(err.Error())
		c.Quit()
		return
	}
	if err := c.Pass(password); err !=nil {
		log.LogError(err.Error())
		c.Quit()
		return
	}

	count, size, err := c.Stat() 
	if err !=nil {
		log.LogError(err.Error())
		c.Quit()
		return
	}
	log.LogInfo("count : %d -- size: %d", count, size)

	msgInfo, err := c.UIDlAll() 
	if err !=nil {
		log.LogError(err.Error())
		c.Quit()
		return
	}

	for _,v := range msgInfo{
		if v.Seq == count{ // For testing get last email only
			msg, err := c.Retr(v.Seq)
			if err !=nil {
				log.LogError(err.Error())
			}else{
				sendMessages(msg)
			}
		}
	}

	// Send the QUIT command and close the connection.
	if err := c.Quit(); err !=nil {
		log.LogError(err.Error())
	}
}

func sendMessages( data string){
	// Connect to the remote SMTP server.
	c, err := smtp.Dial("localhost:2525")
	if err != nil {
			log.LogError(err.Error())
			return
	}
	senderRcpt := getSenderRcpt(data)
	getSndRcpt := strings.Split(senderRcpt,"~")

	// Set the sender and recipient first
	if err := c.Mail(getSndRcpt[0]); err != nil {
			log.LogError(err.Error())
	}
	if err := c.Rcpt(getSndRcpt[1]); err != nil {
			log.LogError(err.Error())
			return
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
			log.LogError(err.Error())
	}
	_, err = fmt.Fprintf(wc, data)
	if err != nil {
			log.LogError(err.Error())
	}
	err = wc.Close()
	if err != nil {
			log.LogError(err.Error())
	}

	// Send the QUIT command and close the connection.
	err = c.Quit()
	if err != nil {
			log.LogError(err.Error())
	}
}

func getSenderRcpt(msg string)string{
	if rm, err := mail.ReadMessage(bytes.NewBufferString(msg)); err == nil {
		mailFrom := rm.Header.Get("From")
		mailTo := rm.Header.Get("To")
		mailCc := rm.Header.Get("Cc")
		return fmt.Sprintf("%s~%s~%s", mailFrom, mailTo , mailCc )
	}else{
		log.LogError(err.Error())
	}
	
	return ""
}