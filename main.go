package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	ubot "github.com/UBotPlatform/UBot.Common.Go"
	"github.com/go-co-op/gocron"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var api *ubot.AppApi

func goodMorning() {
	msg := buildGoodMorningMsg()
	bots, err := api.GetBotList()
	if err != nil {
		return
	}
	for _, bot := range bots {
		groups, err := api.GetGroupList(bot)
		if err != nil {
			continue
		}
		for _, group := range groups {
			_ = api.SendChatMessage(bot, ubot.GroupMsg, group, "", msg)
			time.Sleep(3 * time.Second)
		}
	}
}

func buildGoodMorningMsg() string {
	date := time.Now().Local()
	var builder ubot.MsgBuilder
	builder.WriteString("早上好，今天是")
	builder.WriteString(date.Format("01月02日"))
	builder.WriteString("，")
	weekDayStr := [...]string{"星期日", "星期一", "星期二", "星期三", "星期四", "星期五", "星期六"}[date.Weekday()]
	festivalName := GetFestivalNameThisYear(int(date.Month()), date.Day())
	if festivalName != "" {
		builder.WriteString(festivalName)
		builder.WriteString("（")
		builder.WriteString(weekDayStr)
		builder.WriteString("）")
	} else {
		builder.WriteString(weekDayStr)
	}
	builder.WriteString("，祝您生活愉快。\n每日一言：")
	builder.WriteString(GetHitokoto())
	return builder.String()
}

func GetHitokoto() string {
	resp, err := http.Get("https://v1.hitokoto.cn/?encode=text")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	return string(data)
}

func GetFestivalNameThisYear(month int, day int) string {
	resp, err := http.Get("https://tools.2345.com/jieri.htm")
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	reader := transform.NewReader(resp.Body, simplifiedchinese.GBK.NewDecoder())
	binData, err := ioutil.ReadAll(reader)
	if err != nil {
		return ""
	}
	data := string(binData)
	suffix := fmt.Sprintf("</a>[%02d/%02d]</li>", month, day)
	pEnd := strings.Index(data, suffix)
	if pEnd == -1 {
		return ""
	}
	pStart := strings.LastIndex(data[:pEnd], ">")
	if pStart == -1 {
		return ""
	}
	pStart++
	return data[pStart:pEnd]
}

func main() {
	err := ubot.HostApp("GoodMorning", func(e *ubot.AppApi) *ubot.App {
		api = e
		s := gocron.NewScheduler(time.Local)
		_, err := s.Every(1).Day().At("08:00").Do(goodMorning)
		ubot.AssertNoError(err)
		s.StartAsync()
		return &ubot.App{}
	})
	ubot.AssertNoError(err)
}
