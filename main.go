package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	ubot "github.com/UBotPlatform/UBot.Common.Go"
	"github.com/go-co-op/gocron"
	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

var api *ubot.AppApi
var configFile string

type ConfigModel struct {
	Switches map[string]bool `json:"switches,omitempty"`
	At       string          `json:"at,omitempty"`
}

func fetchConfig() ConfigModel {
	var config ConfigModel
	configBinary, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println("failed to fetch config:", err)
		return config
	}
	err = json.Unmarshal(configBinary, &config)
	if err != nil {
		fmt.Println("failed to parse config:", err)
	}
	return config
}

func goodMorning() {
	fmt.Println("=== TASK EXECUTED AT", time.Now().UTC().Format(time.RFC3339), "===")

	config := fetchConfig()

	var switchDefault bool
	var ok bool
	var sent = make(map[string]bool)

	if switchDefault, ok = config.Switches[""]; !ok {
		switchDefault = true
	}

	msg := buildGoodMorningMsg()
	bots, err := api.GetBotList()
	if err != nil {
		fmt.Println("failed to fetch bot list:", err)
		return
	}

	for _, bot := range bots {
		platformID, _ := api.GetPlatformID(bot)
		groups, err := api.GetGroupList(bot)
		if err != nil {
			continue
		}

		var switchThisPlatform bool
		if switchThisPlatform, ok = config.Switches[platformID]; !ok {
			switchThisPlatform = switchDefault
		}

		for _, group := range groups {
			var ugid = platformID + group
			if sent[ugid] {
				continue
			} else {
				sent[ugid] = true
			}

			var switchThisGroup bool
			if switchThisGroup, ok = config.Switches[ugid]; !ok {
				switchThisGroup = switchThisPlatform
			}
			if !switchThisGroup {
				fmt.Println("skip", ugid, "due to switch off")
				continue
			}
			fmt.Println("sending to", ugid)

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
	executableFile, err := os.Executable()
	if err != nil {
		panic(err)
	}
	configFile = filepath.Join(filepath.Dir(executableFile), "GoodMorning.ubot.json")
	config := fetchConfig()
	err = ubot.HostApp("GoodMorning", func(e *ubot.AppApi) *ubot.App {
		api = e
		s := gocron.NewScheduler(time.Local)
		at := config.At
		if at == "" {
			at = "08:00"
		}
		_, err := s.Every(1).Day().At(at).Do(goodMorning)
		ubot.AssertNoError(err)
		s.StartAsync()
		return &ubot.App{}
	})
	ubot.AssertNoError(err)
}
