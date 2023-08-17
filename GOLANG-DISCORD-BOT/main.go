package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/togatoga/goforces"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/bwmarrin/discordgo"
)

var (
	Token     string
	BotPrefix string

	config *configStruct
)

type configStruct struct {
	Token     string `json:"Token"`
	BotPrefix string `json:"BotPrefix"`
}

func ReadConfig() error {
	fmt.Println("Reading config file...")
	file, err := ioutil.ReadFile("./config.json")

	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	fmt.Println(string(file))

	err = json.Unmarshal(file, &config)

	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	Token = config.Token
	BotPrefix = config.BotPrefix

	return nil

}

var BotId string
var goBot *discordgo.Session
var balance = make(map[string]int64)

func Start() {
	goBot, err := discordgo.New("Bot " + config.Token)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	u, err := goBot.User("@me")

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	BotId = u.ID

	goBot.AddHandler(messageHandler)

	err = goBot.Open()

	if err != nil {
		fmt.Println(err.Error())
		return
	}
	fmt.Println("Bot is running !")
}

func cota(r1 int, r2 int) (int, int) {
	var c1 float64 = (math.Pow(10, -(float64)(r1-r2)/400) + 1) * 100
	var c2 float64 = (math.Pow(10, -(float64)(r2-r1)/400) + 1) * 100

	return int(c1), int(c2)
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	if m.Content == BotPrefix+"ping" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "pong")
	} else if strings.HasPrefix(m.Content, BotPrefix+"add ") {
		m.Content += " "
		var user = ""
		var sum int64 = 0
		var i = 5
		for m.Content[i] != ' ' {
			user += string(m.Content[i])
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			sum = sum*10 + int(m.Content[i]-'0')
			i++
		}

		balance[user] += sum
		_, _ = s.ChannelMessageSend(m.ChannelID, strconv.Itoa(sum)+" were added to "+user+"'s balance")
	} else if m.Content == BotPrefix+"show" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Your balance is "+strconv.Itoa(balance[m.Author.ID]))
	} else if strings.HasPrefix(m.Content, BotPrefix+"show ") {
		m.Content += " "
		var user = ""
		var i = 6
		for m.Content[i] != ' ' {
			user += string(m.Content[i])
			i++
		}

		_, _ = s.ChannelMessageSend(m.ChannelID, user+"'s balance is "+strconv.Itoa(balance[user]))
	} else if m.Content == BotPrefix+"help" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "```show user: shows user's balance"+"\n"+"show: shows your balance"+
			"\n"+"add user sum: adds sum to user's balance```")
	} else if strings.HasPrefix(m.Content, BotPrefix+"cota ") {
		m.Content += " "
		var user1 = ""
		var user2 = ""
		var i = 6
		for m.Content[i] != ' ' {
			user1 += string(m.Content[i])
			i++
		}
		i++
		for m.Content[i] != ' ' {
			user2 += string(m.Content[i])
			i++
		}

		ctx := context.Background()
		logger := log.New(os.Stderr, "*** ", log.LstdFlags)
		api, _ := goforces.NewClient(logger)

		var ratingChange1, _ = api.GetUserRating(ctx, user1)
		var ratingChange2, _ = api.GetUserRating(ctx, user2)
		var rating1 = ratingChange1[len(ratingChange1)-1].NewRating
		var rating2 = ratingChange2[len(ratingChange2)-1].NewRating
		var cota1, cota2 = cota(rating1, rating2)
		cota1 += 1
		cota2 += 1
		if cota1 > 50100 {
			cota1 = 50100
		}
		if cota2 > 50100 {
			cota2 = 50100
		}

		_, _ = s.ChannelMessageSend(m.ChannelID, strconv.Itoa(cota1/100)+"."+strconv.Itoa(cota1/10%10)+strconv.Itoa(cota1%10)+"-"+strconv.Itoa(cota2/100)+"."+strconv.Itoa(cota2/10%10)+strconv.Itoa(cota2%10))
	} else if strings.HasPrefix(m.Content, BotPrefix+"bet ") {
		m.Content += " "
		var user1 = ""
		var user2 = ""
		var sum int64 = 0
		var id int64 = 0
		var i = 5
		for m.Content[i] != ' ' {
			user1 += string(m.Content[i])
			i++
		}
		i++
		for m.Content[i] != ' ' {
			user2 += string(m.Content[i])
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			sum = sum*10 + int(m.Content[i]-'0')
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			id = id*10 + int(m.Content[i]-'0')
			i++
		}

		ctx := context.Background()
		logger := log.New(os.Stderr, "*** ", log.LstdFlags)
		api, _ := goforces.NewClient(logger)
		contest := api.GetContestList()

		var ratingChange1, _ = api.GetUserRating(ctx, user1)
		var ratingChange2, _ = api.GetUserRating(ctx, user2)
		var rating1 = ratingChange1[len(ratingChange1)-1].NewRating
		var rating2 = ratingChange2[len(ratingChange2)-1].NewRating
		var cota1, cota2 = cota(rating1, rating2)
		cota1 += 1
		if cota1 > 50100 {
			cota1 = 50100
		}

		_, _ = s.ChannelMessageSend(m.ChannelID, strconv.Itoa(cota1/100)+"."+strconv.Itoa(cota1/10%10)+strconv.Itoa(cota1%10)+"-"+strconv.Itoa(cota2/100)+"."+strconv.Itoa(cota2/10%10)+strconv.Itoa(cota2%10))
	}
}

func main() {

	err := ReadConfig()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	Start()

	<-make(chan struct{})
	return
}
