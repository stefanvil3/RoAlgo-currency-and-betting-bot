package main

import (
	"container/list"
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

type Bet struct {
	author, winner, loser string
	win                   int64
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
var bets = make(map[int64]*list.List)

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
			sum = sum*10 + int64(m.Content[i]-'0')
			i++
		}

		balance[user] += sum
		_, _ = s.ChannelMessageSend(m.ChannelID, strconv.Itoa(int(sum))+" were added to "+user+"'s balance")
	} else if m.Content == BotPrefix+"show" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Your balance is "+strconv.Itoa(int(balance[m.Author.ID])))
	} else if strings.HasPrefix(m.Content, BotPrefix+"show ") {
		m.Content += " "
		var user = ""
		var i = 6
		for m.Content[i] != ' ' {
			user += string(m.Content[i])
			i++
		}

		_, _ = s.ChannelMessageSend(m.ChannelID, user+"'s balance is "+strconv.Itoa(int(balance[user])))
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

		var ratingChange1, err1 = api.GetUserRating(ctx, user1)
		var ratingChange2, err2 = api.GetUserRating(ctx, user2)
		if err1 == nil && err2 == nil {
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
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Invalid user(s)")
		}
	} else if strings.HasPrefix(m.Content, BotPrefix+"bet cf ") {
		m.Content += " "
		var user1 = ""
		var user2 = ""
		var sum int64 = 0
		var id int64 = 0
		var i = 8
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
			sum = sum*10 + int64(m.Content[i]-'0')
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			id = id*10 + int64(m.Content[i]-'0')
			i++
		}

		ctx := context.Background()
		logger := log.New(os.Stderr, "*** ", log.LstdFlags)
		api, _ := goforces.NewClient(logger)
		contestList, _ := api.GetContestList(ctx, nil)
		i = 0
		for contestList[i].ID != id && !contestList[i].Finished() {
			i++
		}

		if !(contestList[i].ID == id && !contestList[i].Finished()) {
			_, _ = s.ChannelMessageSend(m.ChannelID, "The contest is invalid!")
		} else if balance[m.Author.ID] >= sum {

			balance[m.Author.ID] -= sum

			var ratingChange1, err1 = api.GetUserRating(ctx, user1)
			var ratingChange2, err2 = api.GetUserRating(ctx, user2)
			if err1 == nil && err2 == nil {
				var rating1 = ratingChange1[len(ratingChange1)-1].NewRating
				var rating2 = ratingChange2[len(ratingChange2)-1].NewRating
				var cota1, _ = cota(rating1, rating2)
				cota1 += 1
				if cota1 > 50100 {
					cota1 = 50100
				}
				var win int64 = (sum - sum/20) * int64(cota1) / 100

				bet := Bet{m.Author.ID, user1, user2, win}
				bets[id].PushBack(bet)

				_, _ = s.ChannelMessageSend(m.ChannelID, "You bet "+strconv.Itoa(int(sum))+" on "+user1+" vs "+user2+" in the Codforces contest: "+strconv.Itoa(int(id))+" with a potentially win of "+strconv.Itoa(int(win)))
			} else {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Invalid user(s)")
			}
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Insufficient funds")
		}
	}
}

func main() {

	err := ReadConfig()

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	Start()
	fmt.Println("ok")
	/*
		const ONE_DAY = 24 * 60 * 60

		ctx := context.Background()
		logger := log.New(os.Stderr, "*** ", log.LstdFlags)
		api, _ := goforces.NewClient(logger)
			for true {
				time.Sleep(ONE_DAY)
				contestList, _ := api.GetContestList(ctx, nil)
				for key, value := range bets {
					i := 0
					for contestList[i].ID != key {
						i++
					}
					if contestList[i].Finished() {

					}
				}
			}*/

	<-make(chan struct{})
	return
}
