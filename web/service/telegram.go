package service

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"time"
	"x-ui/logger"
	"x-ui/util/common"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
)

// This should be global variable, and only one instance
var botInstace *tgbotapi.BotAPI

// Structural types can be accessed by other bags
type TelegramService struct {
	xrayService    XrayService
	serverService  ServerService
	inboundService InboundService
	settingService SettingService
}

func (s *TelegramService) GetsystemStatus() string {
	var status string
	// get hostname
	name, err := os.Hostname()
	if err != nil {
		fmt.Println("get hostname error: ", err)
		return ""
	}

	status = fmt.Sprintf("😊 Host Name: %s\r\n", name)
	status += fmt.Sprintf("🔗 System: %s\r\n", runtime.GOOS)
	status += fmt.Sprintf("⬛ CPU Load: %s\r\n", runtime.GOARCH)

	avgState, err := load.Avg()
	if err != nil {
		logger.Warning("get load avg failed: ", err)
	} else {
		status += fmt.Sprintf("⭕ System load: %.2f, %.2f, %.2f\r\n", avgState.Load1, avgState.Load5, avgState.Load15)
	}

	upTime, err := host.Uptime()
	if err != nil {
		logger.Warning("get uptime failed: ", err)
	} else {
		status += fmt.Sprintf("⏳ Operation hours: %s\r\n", common.FormatTime(upTime))
	}

	// xray version
	status += fmt.Sprintf("🟡 Current XRay kernel version: %s\r\n", s.xrayService.GetXrayVersion())

	// ip address
	var ip string
	ip = common.GetMyIpAddr()
	status += fmt.Sprintf("🆔 IP Address: %s\r\n \r\n", ip)

	// get traffic
	inbouds, err := s.inboundService.GetAllInbounds()
	if err != nil {
		logger.Warning("StatsNotifyJob run error: ", err)
	}

	for _, inbound := range inbouds {
		status += fmt.Sprintf("😎 Node name: %s\r\nport: %d\r\nUplink Traffic↑: %s\r\nDownlink Traffic↓: %s\r\nTotal traffic: %s\r\n", inbound.Remark, inbound.Port, common.FormatTraffic(inbound.Up), common.FormatTraffic(inbound.Down), common.FormatTraffic((inbound.Up + inbound.Down)))
		if inbound.ExpiryTime == 0 {
			status += fmt.Sprintf("⌚ Understanding time: indefinitely\r\n \r\n")
		} else {
			status += fmt.Sprintf("❗ Expire date: %s\r\n \r\n", time.Unix((inbound.ExpiryTime/1000), 0).Format("2006-01-02 15:04:05"))
		}
	}
	return status
}

func (s *TelegramService) StartRun() {
	logger.Info("telegram service ready to run")
	s.settingService = SettingService{}
	tgBottoken, err := s.settingService.GetTgBotToken()

	if err != nil || tgBottoken == "" {
		logger.Infof("⚠ Telegram service start run failed, GetTgBotToken fail, err: %v, tgBottoken: %s", err, tgBottoken)
		return
	}
	logger.Infof("TelegramService GetTgBotToken:%s", tgBottoken)

	botInstace, err = tgbotapi.NewBotAPI(tgBottoken)

	if err != nil {
		logger.Infof("⚠ Telegram service start run failed, NewBotAPI fail: %v, tgBottoken: %s", err, tgBottoken)
		return
	}
	botInstace.Debug = false
	fmt.Printf("Authorized on account %s", botInstace.Self.UserName)

	// get all my commands
	commands, err := botInstace.GetMyCommands()
	if err != nil {
		logger.Warning("⚠ Telegram service start run error, GetMyCommandsfail: ", err)
	}

	for _, command := range commands {
		fmt.Printf("Command %s, Description: %s \r\n", command.Command, command.Description)
	}

	// get update
	chanMessage := tgbotapi.NewUpdate(0)
	chanMessage.Timeout = 60

	updates := botInstace.GetUpdatesChan(chanMessage)

	for update := range updates {
		if update.Message == nil {
			// NOTE:may there are different bot instance,we could use different bot endApiPoint
			updates.Clear()
			continue
		}

		if !update.Message.IsCommand() {
			continue
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")

		// Extract the command from the Message.
		switch update.Message.Command() {
		case "delete":
			inboundPortStr := update.Message.CommandArguments()
			inboundPortValue, err := strconv.Atoi(inboundPortStr)

			if err != nil {
				msg.Text = "🔴 Invalid entry port, please check"
				break
			}

			//logger.Infof("Will delete port:%d inbound", inboundPortValue)
			error := s.inboundService.DelInboundByPort(inboundPortValue)
			if error != nil {
				msg.Text = fmt.Sprintf("⚠ Deleting the inbound to port %d  failed", inboundPortValue)
			} else {
				msg.Text = fmt.Sprintf("✅ The inbound of the port has been successfully deleted", inboundPortValue)
			}

		case "restart":
			err := s.xrayService.RestartXray(true)
			if err != nil {
				msg.Text = fmt.Sprintln("⚠ Restart XRAY service failed, err: ", err)
			} else {
				msg.Text = "✅ Successfully restarted XRAY service"
			}

		case "disable":
			inboundPortStr := update.Message.CommandArguments()
			inboundPortValue, err := strconv.Atoi(inboundPortStr)
			if err != nil {
				msg.Text = "🔴 Invalid inbound port, please check"
				break
			}
			//logger.Infof("Will delete port:%d inbound", inboundPortValue)
			error := s.inboundService.DisableInboundByPort(inboundPortValue)
			if error != nil {
				msg.Text = fmt.Sprintf("⚠ Disabling the inbound to port %d  failed, err: %s", inboundPortValue, error)
			} else {
				msg.Text = fmt.Sprintf("✅ The inbound of the port %d successfully disabled", inboundPortValue)
			}

		case "enable":
			inboundPortStr := update.Message.CommandArguments()
			inboundPortValue, err := strconv.Atoi(inboundPortStr)
			if err != nil {
				msg.Text = "🔴 Invalid entry port, please check"
				break
			}
			//logger.Infof("Will delete port:%d inbound", inboundPortValue)
			error := s.inboundService.EnableInboundByPort(inboundPortValue)
			if error != nil {
				msg.Text = fmt.Sprintf("⚠ Enabling the inbound to ports %d failed, err: %s", inboundPortValue, error)
			} else {
				msg.Text = fmt.Sprintf("✅ The inbound of the port %d has been successfully enabled ", inboundPortValue)
			}

		case "clear":
			inboundPortStr := update.Message.CommandArguments()
			inboundPortValue, err := strconv.Atoi(inboundPortStr)
			if err != nil {
				msg.Text = "🔴 Invalid entry port, please check"
				break
			}
			error := s.inboundService.ClearTrafficByPort(inboundPortValue)
			if error != nil {
				msg.Text = fmt.Sprintf("⚠ Resting the inbound to port %d failed, err: %s", inboundPortValue, error)
			} else {
				msg.Text = fmt.Sprintf("✅ Resetting the inbound to port %d succeed", inboundPortValue)
			}

		case "clearall":
			error := s.inboundService.ClearAllInboundTraffic()
			if error != nil {
				msg.Text = fmt.Sprintf("⚠ Failure to clean up all inbound traffic, err: %s", error)
			} else {
				msg.Text = fmt.Sprintf("✅ All inbound traffic has been successfully cleaned up")
			}
        // DEPRIATED. UPDATING KERNAL INTO ANY UNSUPPORTED VERSIONS MAY BREAK THE OS
		// case "version":
		//	versionStr := update.Message.CommandArguments()
		//	currentVersion, _ := s.serverService.GetXrayVersions()
		//	if currentVersion[0] == versionStr {
		//		msg.Text = fmt.Sprint("Can't update the same version as the local X-UI XRAY kernel")
		//	}
		//	error := s.serverService.UpdateXray(versionStr)
		//	if error != nil {
		//		msg.Text = fmt.Sprintf("XRAY kernel version upgrade to %s failed, err: %s", versionStr, error)
		//	} else {
		//		msg.Text = fmt.Sprintf("XRAY kernel version upgrade to %s succeed", versionStr)
		//	}
		case "github":
			msg.Text = `👩🏻‍💻 Here's the link to the project: https://github.com/NidukaAkalanka/x-ui-english/
             
            🖋 Author's Note on V0.2: 
            😶 My schedule is becoming tight so I may not be able to update the project frequently. I'm looking for a contributor who is familiar with Go Telegram Bot API, which is at https://go-telegram-bot-api.dev/ to further improve this Bot. (As you can feel, it's lacking the most user-friendly features like Buttons, Emojis...) If you are interested, please fork the repository and submit a pull request with your changes committed.`

		case "status":
			msg.Text = s.GetsystemStatus()

		case "start":
			msg.Text = `😁 Hi there! 
            💖Welcome to use the X-UI panel Telegram Bot! please send /help to see what can I do`
        case "author":
            msg.Text = `
            👦🏻 Author  : Niduka Akalanka
            📍 Github  : https://github.com/NidukaAkalanka
            📞 Telegram: @NidukaAkalanka (Contact for any issues. Please be patient. As I am a student, I may not be able to reply immediately.)
            📧 Email   : admin@itsmeniduka.engineer
            `
		default:
			msg.Text = `⭐ X-UI 0.2 Telegram Bot Commands Menu ⭐

 			
| /help 		    
|-🆘 Get the help information of BOT (this menu)
| 
| /delete [PORT] 
|-♻ Delete the node of the corresponding port
| 
| /restart 
|-🔁 Restart XRAY service
| 
| /status
|-✔ Get the current system state
| 
| /enable [PORT]
|-🧩 Open the node of the corresponding port
|
| /disable [PORT]
|-🚫 Turn off the corresponding port node
|
| /clear [PORT]
|-🧹 Clean up the node traffic of the corresponding port
|
| /clearall 
|-🆕 Clean up all inbound traffics and count from 0
|
| /github
|-✍🏻 Get the project link
|
| /author
|-👦🏻 Get the author's information
`
		}

		if _, err := botInstace.Send(msg); err != nil {
			log.Panic(err)
		}
	}

}

func (s *TelegramService) SendMsgToTgbot(msg string) {
	logger.Info("SendMsgToTgbot entered")
	tgBotid, err := s.settingService.GetTgBotChatId()
	if err != nil {
		logger.Warning("sendMsgToTgbot failed, GetTgBotChatId fail:", err)
		return
	}
	if tgBotid == 0 {
		logger.Warning("sendMsgToTgbot failed, GetTgBotChatId fail")
		return
	}

	info := tgbotapi.NewMessage(int64(tgBotid), msg)
	if botInstace != nil {
		botInstace.Send(info)
	} else {
		logger.Warning("bot instance is nil")
	}
}

// NOTE:This function can't be called repeatly
func (s *TelegramService) StopRunAndClose() {
	if botInstace != nil {
		botInstace.StopReceivingUpdates()
	}
}
