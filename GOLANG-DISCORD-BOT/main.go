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
	"time"
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
	win, sum              int64
}

type EventBet struct {
	author, player string
	win, low, high int64
}
type Result struct {
	points, standing int64
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
var event bool = false
var event_betting bool = false
var event_rewarded bool = false
var event_points int64 = 0
var event_standings int64 = 0
var event_bets_points = make(map[string]*list.List)
var event_bets_standings = make(map[string]*list.List)
var event_results = make(map[string]Result)
var have_event_results = false

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

func cota_points(low int64, high int64) int64 {
	var ret = event_points + 1
	if low != high {
		ret = event_points * 100 / (high - low)
	}

	return ret
}

func cota_standings(low int64, high int64) int64 {
	var ret = event_standings + 1
	if low != high {
		ret = event_standings * 100 / (high - low)
	}

	return ret
}

func messageHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == BotId {
		return
	}

	if m.Content == BotPrefix+"ping" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "pong")
	} else if strings.HasPrefix(m.Content, BotPrefix+"add ") {
		m.Content += "  "
		var user = ""
		var sum int64 = 0
		var sign int64 = 1
		var have_sum = false
		var i = 5
		for m.Content[i] != ' ' {
			user += string(m.Content[i])
			i++
		}

		if user == "" {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Invalid command")
		} else {
			i++
			if m.Content[i] == '-' {
				sign = -1
				i++
			}
			for unicode.IsDigit(rune(m.Content[i])) {
				sum = sum*10 + int64(m.Content[i]-'0')
				i++
				have_sum = true
			}
			if !have_sum {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Invalid command")
			} else {
				sum *= sign
				balance[user] += sum
				_, _ = s.ChannelMessageSend(m.ChannelID, strconv.Itoa(int(sum))+" were added to "+user+"'s balance")
			}
		}
	} else if m.Content == BotPrefix+"show" {
		_, _ = s.ChannelMessageSend(m.ChannelID, "Your balance is "+strconv.Itoa(int(balance[m.Author.Username])))
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
			cota1 = max(101, cota1-5)
			cota2 = max(101, cota2-5)
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

				bet := Bet{m.Author.Username, user1, user2, win, sum}
				bets[id].PushBack(bet)

				_, _ = s.ChannelMessageSend(m.ChannelID, "You bet "+strconv.Itoa(int(sum))+" on "+user1+" vs "+user2+" in the Codforces contest: "+strconv.Itoa(int(id))+" with a potentially win of "+strconv.Itoa(int(win)))
			} else {
				_, _ = s.ChannelMessageSend(m.ChannelID, "Invalid user(s)")
			}
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Insufficient funds, you have only "+strconv.Itoa(int(balance[m.Author.ID])))
		}
	} else if m.Content == BotPrefix+"event start" {
		event = true
		_, _ = s.ChannelMessageSend(m.ChannelID, "Event started")
	} else if m.Content == BotPrefix+"event stop" {
		event = false
		_, _ = s.ChannelMessageSend(m.ChannelID, "Event ended")
	} else if strings.HasPrefix(m.Content, BotPrefix+"event betting start") {
		event_betting = true
		event_rewarded = false
		_, _ = s.ChannelMessageSend(m.ChannelID, "Event betting started")
	} else if strings.HasPrefix(m.Content, BotPrefix+"event betting stop") {
		event_betting = false
		_, _ = s.ChannelMessageSend(m.ChannelID, "Event betting ended")
	} else if strings.HasPrefix(m.Content, BotPrefix+"event cota points ") {
		m.Content += "  "
		var low int64 = 0
		var high int64 = 0
		var i = 19
		for unicode.IsDigit(rune(m.Content[i])) {
			low = low*10 + int64(m.Content[i]-'0')
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			high = high*10 + int64(m.Content[i]-'0')
			i++
		}

		if low < high {
			var aux = low
			low = high
			high = aux
		}

		var cota = cota_points(low, high)
		_, _ = s.ChannelMessageSend(m.ChannelID, strconv.Itoa(int(cota/100))+"."+strconv.Itoa(int(cota/10%10))+strconv.Itoa(int(cota%10)))
	} else if strings.HasPrefix(m.Content, BotPrefix+"event cota standings ") {
		m.Content += "  "
		var low int64 = 0
		var high int64 = 0
		var i = 22
		for unicode.IsDigit(rune(m.Content[i])) {
			low = low*10 + int64(m.Content[i]-'0')
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			high = high*10 + int64(m.Content[i]-'0')
			i++
		}

		if low < high {
			var aux = low
			low = high
			high = aux
		}

		var cota = cota_standings(low, high)
		_, _ = s.ChannelMessageSend(m.ChannelID, strconv.Itoa(int(cota/100))+"."+strconv.Itoa(int(cota/10%10))+strconv.Itoa(int(cota%10)))
	} else if strings.HasPrefix(m.Content, BotPrefix+"event bet points ") {
		m.Content += " "
		var low int64 = 0
		var high int64 = 0
		var sum int64 = 0
		var i = 18
		var player = ""
		for unicode.IsDigit(rune(m.Content[i])) {
			sum = sum*10 + int64(m.Content[i]-'0')
			i++
		}
		i++
		for m.Content[i] != ' ' {
			player += string(m.Content[i])
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			low = low*10 + int64(m.Content[i]-'0')
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			high = high*10 + int64(m.Content[i]-'0')
			i++
		}

		if low < high {
			var aux = low
			low = high
			high = aux
		}

		if balance[m.Author.Username] >= sum {
			balance[m.Author.Username] -= sum

			var cota = cota_points(low, high)
			var win = (sum - sum/10) * cota / 100
			event_bets_points[player].PushBack(EventBet{m.Author.Username, player, win, low, high})

			_, _ = s.ChannelMessageSend(m.ChannelID, "You bet "+strconv.Itoa(int(sum))+" on "+player+" scoring between "+strconv.Itoa(int(low))+" and"+strconv.Itoa(int(high)))
		} else {
			_, _ = s.ChannelMessageSend(m.ChannelID, "Insufficient funds")
		}
	} else if strings.HasPrefix(m.Content, BotPrefix+"event bet standings ") {
		m.Content += " "
		var low int64 = 0
		var high int64 = 0
		var sum int64 = 0
		var i = 22
		var player = ""
		for unicode.IsDigit(rune(m.Content[i])) {
			sum = sum*10 + int64(m.Content[i]-'0')
			i++
		}
		i++
		for m.Content[i] != ' ' {
			player += string(m.Content[i])
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			low = low*10 + int64(m.Content[i]-'0')
			i++
		}
		i++
		for unicode.IsDigit(rune(m.Content[i])) {
			high = high*10 + int64(m.Content[i]-'0')
			i++
		}

		if low < high {
			var aux = low
			low = high
			high = aux
		}

		if balance[m.Author.Username] >= sum {
			balance[m.Author.Username] -= sum

			var cota = cota_points(low, high)
			var win = (sum - sum/10) * cota / 100
			event_bets_standings[player].PushBack(EventBet{m.Author.Username, player, win, low, high})

			_, _ = s.ChannelMessageSend(m.ChannelID, "You bet "+strconv.Itoa(int(sum))+" on "+player+" standing between "+strconv.Itoa(int(low))+" and"+strconv.Itoa(int(high)))
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

	const ONE_DAY = 24 * 60 * 60 * 1000 * time.Second

	ctx := context.Background()
	logger := log.New(os.Stderr, "*** ", log.LstdFlags)
	api, _ := goforces.NewClient(logger)

	for true {
		fmt.Println("523\n523\n523\n523\n523\n523\n523\n523\n523\n523\n523\n523\n523\n523\n523\n")

		time.Sleep(ONE_DAY)

		if have_event_results && !event_rewarded {
			for player, result := range event_results {
				for e := event_bets_points[player].Front(); e != nil; e = e.Next() {
					var bett = e.Value.(EventBet)
					if bett.low <= result.points && result.points <= bett.high {
						balance[bett.author] += bett.win
					}
				}

				for e := event_bets_standings[player].Front(); e != nil; e = e.Next() {
					var bett = e.Value.(EventBet)
					if bett.low <= result.standing && result.standing <= bett.high {
						balance[bett.author] += bett.win
					}
				}
			}
		}

		contestList, _ := api.GetContestList(ctx, nil)
		for key, value := range bets {
			i := 0
			for contestList[i].ID != key {
				i++
			}
			if contestList[i].Finished() {
				var standings, err = api.GetContestStandings(ctx, int(contestList[i].ID), nil)
				if err != nil {

					for e := value.Front(); e != nil; e = e.Next() {
						var bett = e.Value.(Bet)

						var i = 0
						var rankwinner int64 = -1
						var rankloser int64 = -1
						for rankwinner == -1 && rankloser == -1 && i < len(standings.Rows) {
							var j = 0
							for j < len(standings.Rows[i].Party.Members) {
								if standings.Rows[i].Party.Members[j].Handle == bett.winner {
									rankwinner = standings.Rows[i].Rank
								}
								if standings.Rows[i].Party.Members[j].Handle == bett.loser {
									rankloser = standings.Rows[i].Rank
								}
								j++
							}
							i++
						}

						if rankloser == -1 || rankwinner == -1 {
							balance[bett.author] += bett.sum
						} else if rankwinner < rankloser {
							balance[bett.author] += bett.win
						}
					}
				}
			}
		}

	}

	<-make(chan struct{})
	return
}
