package pop3client

import (
	"strings"
	"github.com/fahrudina/go-pop3"
	"github.com/helioina/api/log"
	"net/smtp"
	"net/mail"
	"fmt"
	"bytes"
	"time"
	"github.com/helioina/api/contexts"
)

func InitiateGetMessages(pop3Address, username , password, email string, ctx *contexts.Context){
	c, err := pop3.Dial( pop3Address , pop3.UseTimeout(0) )
	//log.LogTrace("%s----- %s ", username, password)
	if err != nil{
		log.LogError(err.Error())
		return
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
	}else{
		var download uint32
		dataPop := map[string]interface{}{
			"username" : username,
			"email": email,
			"password": encPasswd(password), //password,
			"count": count,
			"downloaded": download,
			"lastdownload": time.Now(),
		}
		if err := ctx.Ds.AddDataPop3Client(dataPop); err != nil{
			log.LogError(err.Error())
		}
	}
	log.LogInfo("count : %d -- size: %d", count, size)

	msgInfo, err := c.UIDlAll() 
	if err !=nil {
		log.LogError(err.Error())
		c.Quit()
		return 
	}

	for _,v := range msgInfo{
		if v.Seq != count{ // For testing get last email only
			msg, err := c.Retr(v.Seq)
			if err !=nil {
				log.LogError(err.Error())
			}else{
				if err := sendMessages(msg, email); err == nil{
					fieldsToUpdate:= map[string]interface{}{
						"msgdownloaded": +1,
					}
					if err := ctx.Ds.UpdatePop3Data(username, fieldsToUpdate); err != nil{
						log.LogError(err.Error())
					}
				}

			}
		}
	}

	// Send the QUIT command and close the connection.
	if err := c.Quit(); err !=nil {
		log.LogError(err.Error())
	}
}

func GetMessages(pop3Address, username string, ctx *contexts.Context){

	c, err := pop3.Dial( pop3Address , pop3.UseTimeout(0) )
	if err != nil{
		log.LogError(err.Error())
		return
	}

	popData , err := ctx.Ds.CheckPop3Download(username)
	if err != nil{
		log.LogError(err.Error())
		return
	}
	
	if err := c.User(popData.Username); err != nil {
		log.LogError(err.Error())
		c.Quit()
		return 
	}
	if err := c.Pass(decPassword( popData.Password)); err !=nil {
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
	amount := popData.MsgDownloaded
	for _,_ = range msgInfo{
		if amount != count{ 
			amount++
			msg, err := c.Retr(amount)
			if err !=nil {
				log.LogError(err.Error())
			}else{
				if err := sendMessages(msg, popData.Email); err != nil{
					log.LogError(err.Error())
				}else{
					fieldsToUpdate:= map[string]interface{}{ "msgdownloaded": +1 }

					if err := ctx.Ds.UpdatePop3Data(popData.Username, fieldsToUpdate); err != nil{
						log.LogError(err.Error())
					}
				}
			}
		}
	}

	// Send the QUIT command and close the connection.
	if err := c.Quit(); err !=nil {
		log.LogError(err.Error())
	}
}

func sendMessages( data , emailRcpt string) error {
	// Connect to the remote SMTP server.
	c, err := smtp.Dial("localhost:2525")
	if err != nil {
		log.LogError(err.Error())
		return err
	}
	senderRcpt := getSenderRcpt(data)
	getSndRcpt := strings.Split(senderRcpt,"~")
	log.LogTrace(senderRcpt)
	// Set the sender and recipient first
	if err := c.Mail(strings.Replace(getSndRcpt[0],"<", "", -1 ) ); err != nil {
			log.LogError(err.Error())
			return err
	}

	if err := c.Rcpt(emailRcpt/*getSndRcpt[1]*/); err != nil {
			log.LogError(err.Error())
			return err
	}

	// Send the email body.
	wc, err := c.Data()
	if err != nil {
		log.LogError(err.Error())
		return err
	}
	_, err = fmt.Fprintf(wc, data)
	if err != nil {
		log.LogError(err.Error())
		return err
	}
	err = wc.Close()
	if err != nil {
		log.LogError(err.Error())
		return err
	}

	// Send the QUIT command and close the connection.
	err = c.Quit()
	if err != nil {
		log.LogError(err.Error())
		return err
	}
	return nil
}

func getSenderRcpt(msg string)string{
	if rm, err := mail.ReadMessage(bytes.NewBufferString(msg)); err == nil {
		mailFrom := senderRcptParser(rm.Header.Get("From"))
		mailTo := senderRcptParser(rm.Header.Get("To"))
		mailCc := senderRcptParser(rm.Header.Get("Cc"))
		return fmt.Sprintf("%s~%s~%s", mailFrom, mailTo , mailCc )
	}else{
		log.LogError(err.Error())
	}
	
	return ""
}

func senderRcptParser( mailAccount string ) string {
	var accParser string
	if strings.Contains(mailAccount, "<"){
		mAccSpl := strings.Split(mailAccount,"<")
		accParser = strings.TrimLeft(strings.TrimRight(mAccSpl[1], ">"), "<")
	}else{
		accParser = strings.TrimLeft(strings.TrimRight(mailAccount, ">"), "<")
	}
	return accParser
}

func encPasswd(password string)[]byte {
    key := []byte("67890987654321234567890098765432") // 32 bytes
    plaintext := []byte(password)
    ciphertext, err := encrypt(key, plaintext)
    if err != nil {
        log.LogError(err.Error())
	}
	return ciphertext
}

func decPassword(ciphertext []byte)string{
	key := []byte("67890987654321234567890098765432") // 32 bytes
	result, err := decrypt(key, ciphertext)
    if err != nil {
		log.LogError(err.Error())
    }
    return bytesToString(result)
}

func bytesToString(data []byte) string {
	return string(data[:])
}
