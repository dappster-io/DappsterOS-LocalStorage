package v1

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dappster-io/DappsterOS-Common/utils/logger"
	"github.com/dappster-io/DappsterOS-LocalStorage/drivers/dropbox"
	"github.com/dappster-io/DappsterOS-LocalStorage/drivers/google_drive"
	"github.com/dappster-io/DappsterOS-LocalStorage/service"
	"github.com/labstack/echo/v4"
	"go.uber.org/zap"
)

func GetRecoverStorage(ctx echo.Context) error {
	ctx.Request().Header.Add("Content-Type", "text/html; charset=utf-8")
	t := ctx.Param("type")
	currentTime := time.Now().UTC()
	currentDate := time.Now().UTC().Format("2006-01-02")
	notify := make(map[string]interface{})
	if t == "GoogleDrive" {
		add := google_drive.Addition{}
		add.Code = ctx.QueryParam("code")
		if len(add.Code) == 0 {
			ctx.String(200, `<p>Code cannot be empty</p><script>window.close()</script>`)
			notify["status"] = "fail"
			notify["message"] = "Code cannot be empty"
			logger.Error("Then code is empty: ", zap.String("code", add.Code), zap.Any("name", "google_drive"))
			service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
		}

		add.RootFolderID = "root"
		add.ClientID = google_drive.CLIENTID
		add.ClientSecret = google_drive.CLIENTSECRET

		var google_drive google_drive.GoogleDrive
		google_drive.Addition = add
		err := google_drive.Init(ctx.Request().Context())
		if err != nil {
			ctx.String(200, `<p>Initialization failure:`+err.Error()+`</p><script>window.close()</script>`)
			notify["status"] = "fail"
			notify["message"] = "Initialization failure"
			logger.Error("Then init error: ", zap.Error(err), zap.Any("name", "google_drive"))
			service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
		}

		username, err := google_drive.GetUserInfo(ctx.Request().Context())
		if err != nil {
			ctx.String(200, `<p>Failed to get user information:`+err.Error()+`</p><script>window.close()</script>`)
			notify["status"] = "fail"
			notify["message"] = "Failed to get user information"
			logger.Error("Then get user info error: ", zap.Error(err), zap.Any("name", "google_drive"))
			service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
		}
		dmap := make(map[string]interface{})
		dmap["username"] = username
		configs, err := service.MyService.Storage().GetConfig()
		if err != nil {
			ctx.String(200, `<p>Failed to get rclone config:`+err.Error()+`</p><script>window.close()</script>`)
			notify["status"] = "fail"
			notify["message"] = "Failed to get rclone config"
			logger.Error("Then get config error: ", zap.Error(err), zap.Any("name", "google_drive"))
			service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
		}
		for _, v := range configs.Remotes {
			t := service.MyService.Storage().GetAttributeValueByName(v, "type")
			username := service.MyService.Storage().GetAttributeValueByName(v, "username")
			if err != nil {
				logger.Error("then get config by name error: ", zap.Error(err), zap.Any("name", v))
				continue
			}
			if t == "drive" && username == dmap["username"] {
				ctx.String(200, `<p>The same configuration has been added</p><script>window.close()</script>`)
				err := service.MyService.Storage().CheckAndMountByName(v)
				if err != nil {
					logger.Error("check and mount by name error: ", zap.Error(err), zap.Any("name", username))
				}
				notify["status"] = "warn"
				notify["message"] = "The same configuration has been added"
				service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
			}
		}
		if len(username) > 0 {
			a := strings.Split(username, "@")
			username = a[0]
		}

		// username = fileutil.NameAccumulation(username, "/mnt")
		username += "_google_drive_" + strconv.FormatInt(time.Now().Unix(), 10)

		dmap["client_id"] = add.ClientID
		dmap["client_secret"] = add.ClientSecret
		dmap["scope"] = "drive"
		dmap["mount_point"] = "/mnt/" + username
		dmap["token"] = `{"access_token":"` + google_drive.AccessToken + `","token_type":"Bearer","refresh_token":"` + google_drive.RefreshToken + `","expiry":"` + currentDate + `T` + currentTime.Add(time.Hour*1).Add(time.Minute*50).Format("15:04:05") + `Z"}`
		service.MyService.Storage().CreateConfig(dmap, username, "drive")
		service.MyService.Storage().MountStorage("/mnt/"+username, username)
		notify := make(map[string]interface{})
		notify["status"] = "success"
		notify["message"] = "Success"
		notify["driver"] = "GoogleDrive"
		fmt.Println(service.MyService.Notify().SendNotify("dappsteros:file:recover", notify))
	} else if t == "Dropbox" {
		add := dropbox.Addition{}
		add.Code = ctx.QueryParam("code")
		if len(add.Code) == 0 {
			ctx.String(200, `<p>Code cannot be empty</p><script>window.close()</script>`)
			notify["status"] = "fail"
			notify["message"] = "Code cannot be empty"
			logger.Error("Then code is empty error: ", zap.String("code", add.Code), zap.Any("name", "dropbox"))
			service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
		}
		add.RootFolderID = ""
		add.AppKey = dropbox.APPKEY
		add.AppSecret = dropbox.APPSECRET
		var dropbox dropbox.Dropbox
		dropbox.Addition = add
		err := dropbox.Init(ctx.Request().Context())
		if err != nil {
			ctx.String(200, `<p>Initialization failure:`+err.Error()+`</p><script>window.close()</script>`)
			notify["status"] = "fail"
			notify["message"] = "Initialization failure"
			logger.Error("Then init error: ", zap.Error(err), zap.Any("name", "dropbox"))
			service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
		}
		username, err := dropbox.GetUserInfo(ctx.Request().Context())
		if err != nil {
			ctx.String(200, `<p>Failed to get user information:`+err.Error()+`</p><script>window.close()</script>`)
			notify["status"] = "fail"
			notify["message"] = "Failed to get user information"
			logger.Error("Then get user information: ", zap.Error(err), zap.Any("name", "dropbox"))
			service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
		}
		dmap := make(map[string]interface{})
		dmap["username"] = username

		configs, err := service.MyService.Storage().GetConfig()
		if err != nil {
			ctx.String(200, `<p>Failed to get rclone config:`+err.Error()+`</p><script>window.close()</script>`)
			notify["status"] = "fail"
			notify["message"] = "Failed to get rclone config"
			logger.Error("Then get config error: ", zap.Error(err), zap.Any("name", "dropbox"))
			service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
		}
		for _, v := range configs.Remotes {

			t := service.MyService.Storage().GetAttributeValueByName(v, "type")
			username := service.MyService.Storage().GetAttributeValueByName(v, "username")
			if err != nil {
				logger.Error("then get config by name error: ", zap.Error(err), zap.Any("name", v))
				continue
			}
			if t == "dropbox" && username == dmap["username"] {
				ctx.String(200, `<p>The same configuration has been added</p><script>window.close()</script>`)
				err := service.MyService.Storage().CheckAndMountByName(v)
				if err != nil {
					logger.Error("check and mount by name error: ", zap.Error(err), zap.Any("name", username))
				}

				notify["status"] = "warn"
				notify["message"] = "The same configuration has been added"
				service.MyService.Notify().SendNotify("dappsteros:file:recover", notify)
			}
		}
		if len(username) > 0 {
			a := strings.Split(username, "@")
			username = a[0]
		}
		username += "_dropbox_" + strconv.FormatInt(time.Now().Unix(), 10)

		dmap["client_id"] = add.AppKey
		dmap["client_secret"] = add.AppSecret
		dmap["token"] = `{"access_token":"` + dropbox.AccessToken + `","token_type":"bearer","refresh_token":"` + dropbox.Addition.RefreshToken + `","expiry":"` + currentDate + `T` + currentTime.Add(time.Hour*3).Add(time.Minute*50).Format("15:04:05") + `.780385354Z"}`
		dmap["mount_point"] = "/mnt/" + username
		// data.SetValue(username, "type", "dropbox")
		// data.SetValue(username, "client_id", add.AppKey)
		// data.SetValue(username, "client_secret", add.AppSecret)
		// data.SetValue(username, "mount_point", "/mnt/"+username)

		// data.SetValue(username, "token", `{"access_token":"`+dropbox.AccessToken+`","token_type":"bearer","refresh_token":"`+dropbox.Addition.RefreshToken+`","expiry":"`+currentDate+`T`+currentTime.Add(time.Hour*3).Format("15:04:05")+`.780385354Z"}`)
		// e = data.Save()
		// if e != nil {
		// 	ctx.String(200, `<p>保存配置失败:`+e.Error()+`</p>`)

		// 	return
		// }
		service.MyService.Storage().CreateConfig(dmap, username, "dropbox")
		service.MyService.Storage().MountStorage("/mnt/"+username, username)

		notify["status"] = "success"
		notify["message"] = "Success"
		notify["driver"] = "Dropbox"
		fmt.Println(service.MyService.Notify().SendNotify("dappsteros:file:recover", notify))

	}

	return ctx.String(200, `<p>Just close the page</p><script>window.close()</script>`)
}
