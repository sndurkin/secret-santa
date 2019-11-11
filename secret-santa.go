package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"math/rand"
	"net/mail"
	"net/url"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/dajohi/goemail"
)

type Participant struct {
	ID           int64
	Name         string
	Gender       string
	SpouseID     int64
	MatchID      *int64
	EmailAddress string
	Wishlist     string
}

type EmailTemplateData struct {
	Name     string
	Pronoun  string
	Wishlist []string
}

type smtp struct {
	client      *goemail.SMTP // SMTP client
	mailName    string        // Email address name
	mailAddress string        // Email address
}

var (
	participants           map[int64]*Participant
	orderedParticipantsIDs []int64

	smtpClient *smtp

	cfg map[string]string

	emailTemplate *template.Template
)

func main() {
	rand.Seed(time.Now().Unix())

	readConfigFromFile("config.txt")
	readParticipantsFromFile("participants.csv")

	for {
		fmt.Println("Attempting matchmaking...")
		if attemptMatchMaking(participants, orderedParticipantsIDs) {
			fmt.Println("Matchmaking successful!")
			break
		}
	}

	if _, ok := cfg["mailhost"]; ok {
		fmt.Println("Sending emails...")
		sendEmails()
	} else {
		fmt.Printf("No email config found, printing results:\n\n")
		for _, p := range participants {
			fmt.Printf("%s -> %s\n", p.Name, participants[*p.MatchID].Name)
		}
	}
}

func readParticipantsFromFile(filePath string) {
	f, err := os.Open(filePath)
	defer f.Close()
	check(err)

	reader := csv.NewReader(f)
	reader.Comment = '#'
	lines, err := reader.ReadAll()
	check(err)

	participants = make(map[int64]*Participant, len(lines))
	orderedParticipantsIDs = make([]int64, 0, len(lines))
	for _, line := range lines {
		id, err := strconv.ParseInt(line[0], 10, 64)
		check(err)

		spouseID, err := strconv.ParseInt(line[3], 10, 64)
		check(err)

		participants[id] = &Participant{
			ID:           id,
			Name:         line[1],
			Gender:       line[2],
			SpouseID:     spouseID,
			EmailAddress: line[4],
			Wishlist:     line[5],
		}
		orderedParticipantsIDs = append(orderedParticipantsIDs, id)
	}
}

func attemptMatchMaking(participants map[int64]*Participant,
	orderedParticipantsIDs []int64) bool {

	unmatchedParticipantsIDs := map[int]int64{}
	for idx, id := range orderedParticipantsIDs {
		unmatchedParticipantsIDs[idx] = id
	}

	for _, id := range orderedParticipantsIDs {
		//fmt.Printf("Matching %s... ", participants[id].Name)
		for {
			// Grab a random index into the array of unmatched participants.
			unmatchedParticipantsIndices := reflect.ValueOf(
				unmatchedParticipantsIDs).MapKeys()
			keyIdx := rand.Intn(len(unmatchedParticipantsIndices))
			matchIdx := unmatchedParticipantsIndices[keyIdx].Interface().(int)
			matchID := unmatchedParticipantsIDs[matchIdx]
			if matchID != id && matchID != participants[id].SpouseID {
				participants[id].MatchID = &matchID

				delete(unmatchedParticipantsIDs, matchIdx)
				//fmt.Printf("matched with %s\n", participants[matchID].Name)
				break
			}

			if len(unmatchedParticipantsIDs) <= 2 {
				fmt.Printf("\nRan out of participants to match with, starting over\n")
				// We ended up with
				return false
			}
		}
	}

	return true
}

func sendEmails() {
	initEmailClient()
	initEmailTemplate()

	for _, participant := range participants {
		match := participants[*participant.MatchID]

		templateData := EmailTemplateData{
			Name: match.Name,
		}

		if len(match.Wishlist) == 0 {
			templateData.Wishlist = []string{
				"(not available)",
			}
		} else {
			templateData.Wishlist = strings.Split(match.Wishlist, ",")
		}

		if strings.Compare(match.Gender, "Male") == 0 {
			templateData.Pronoun = "his"
		} else {
			templateData.Pronoun = "her"
		}

		var buf bytes.Buffer
		err := emailTemplate.Execute(&buf, templateData)
		check(err)

		sendEmail(participant.EmailAddress, "Secret Santa 2019", buf.String())
	}
}

func initEmailClient() {
	// Parse mail host
	h := fmt.Sprintf("smtps://%v:%v@%v", cfg["mailuser"], cfg["mailpass"],
		cfg["mailhost"])
	u, err := url.Parse(h)
	check(err)

	// Parse email address
	a, err := mail.ParseAddress(cfg["mailaddr"])
	check(err)

	// Config tlsConfig based on config settings
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
	}

	// Initialize SMTP client
	client, err := goemail.NewSMTP(u.String(), tlsConfig)
	check(err)

	smtpClient = &smtp{
		client:      client,
		mailName:    a.Name,
		mailAddress: a.Address,
	}
}

func initEmailTemplate() {
	buf, err := ioutil.ReadFile("email.htmlt")
	check(err)

	emailTemplate, err = template.New("email").Parse(string(buf))
	check(err)
}

func sendEmail(toAddress, subject, body string) {
	msg := goemail.NewHTMLMessage(cfg["mailaddr"], subject, body)
	msg.AddTo(toAddress)

	msg.SetName(smtpClient.mailName)
	err := smtpClient.client.Send(msg)
	check(err)
}

func readConfigFromFile(filePath string) map[string]string {
	f, err := os.Open(filePath)
	defer f.Close()
	check(err)

	r := bufio.NewReader(f)
	cfg = make(map[string]string)

	for {
		t, _, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		check(err)

		if len(t) == 0 {
			break
		}

		s := strings.Split(string(t), "=")
		cfg[s[0]] = s[1]
	}

	return cfg
}

func check(e error) {
	if e != nil {
		panic(e)
	}
}
