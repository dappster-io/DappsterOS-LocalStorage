//go:generate bash -c "mkdir -p codegen && go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 -generate types,server,spec -package codegen api/local_storage/openapi.yaml > codegen/local_storage_api.go"
//go:generate bash -c "mkdir -p codegen/message_bus && go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 -generate types,client -package message_bus https://raw.githubusercontent.com/dappster-io/DappsterOS-MessageBus/main/api/message_bus/openapi.yaml > codegen/message_bus/api.go"

package main

import (
	"context"
	_ "embed"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/coreos/go-systemd/daemon"
	"github.com/dappster-io/DappsterOS-Common/model"
	"github.com/dappster-io/DappsterOS-Common/utils/file"
	util_http "github.com/dappster-io/DappsterOS-Common/utils/http"
	"github.com/dappster-io/DappsterOS-Common/utils/logger"
	"github.com/dappster-io/DappsterOS-LocalStorage/codegen/message_bus"
	"github.com/dappster-io/DappsterOS-LocalStorage/common"
	"github.com/dappster-io/DappsterOS-LocalStorage/pkg/cache"
	"github.com/dappster-io/DappsterOS-LocalStorage/pkg/config"
	"github.com/dappster-io/DappsterOS-LocalStorage/pkg/sqlite"
	"github.com/dappster-io/DappsterOS-LocalStorage/pkg/utils/merge"
	"github.com/dappster-io/DappsterOS-LocalStorage/route"
	"github.com/dappster-io/DappsterOS-LocalStorage/service"
	"github.com/robfig/cron/v3"
	"github.com/samber/lo"
	"go.uber.org/zap"
)

const localhost = "127.0.0.1"

var (
	commit = "private build"
	date   = "private build"

	//go:embed api/index.html
	_docHTML string

	//go:embed api/local_storage/openapi.yaml
	_docYAML string

	//go:embed build/sysroot/etc/dappsteros/local-storage.conf.sample
	_confSample string
)

func init() {

	configFlag := flag.String("c", "", "config address")
	dbFlag := flag.String("db", "", "db path")

	versionFlag := flag.Bool("v", false, "version")
	initFlag := flag.Bool("init", false, "init local-storage config")

	flag.Parse()

	if *versionFlag {
		fmt.Printf("v%s\n", common.Version)
		os.Exit(0)
	}

	println("git commit:", commit)
	println("build date:", date)

	config.InitSetup(*configFlag, _confSample)

	logger.LogInit(config.AppInfo.LogPath, config.AppInfo.LogSaveName, config.AppInfo.LogFileExt)

	if len(*dbFlag) == 0 {
		*dbFlag = config.AppInfo.DBPath
	}

	sqliteDB := sqlite.GetGlobalDB(*dbFlag)

	service.MyService = service.NewService(sqliteDB)
	service.Cache = cache.Init()
	if initFlag != nil && *initFlag {
		service.MyService.Disk().CheckSerialDiskMount()
		os.Exit(0)
		return
	}
	if strings.ToLower(config.ServerInfo.EnableMergerFS) == "true" {
		if !merge.IsMergerFSInstalled() {
			config.ServerInfo.EnableMergerFS = "false"
			logger.Info("mergerfs is disabled")
		}
	}

	if strings.ToLower(config.ServerInfo.EnableMergerFS) == "true" {
		if !service.MyService.Disk().EnsureDefaultMergePoint() {
			config.ServerInfo.EnableMergerFS = "false"
			logger.Info("mergerfs is disabled")
		}
	}

	if strings.ToLower(config.ServerInfo.EnableMergerFS) == "true" {
		go service.MyService.LocalStorage().CheckMergeMount()
	}

	checkToken2_11()
	go ensureDefaultDirectories()
	//service.MyService.Disk().EnsureDefaultMergePoint()

	// service.MountLists = make(map[string]*mountlib.MountPoint)
	// configfile.Install()
	// service.MyService.Storage().CheckAndMountAll()

}

func checkToken2_11() {
	deviceTree, err := service.MyService.USB().GetDeviceTree()
	if err != nil {
		panic(err)
	}

	if service.MyService.USB().GetSysInfo().KernelArch == "aarch64" && strings.ToLower(config.ServerInfo.USBAutoMount) != "true" && strings.Contains(deviceTree, "Raspberry Pi") {
		service.MyService.USB().UpdateUSBAutoMount("False")
		service.MyService.USB().ExecUSBAutoMountShell("False")
	}
}

func ensureDefaultDirectories() {
	sysType := runtime.GOOS
	var dirArray []string
	if sysType == "linux" {
		dirArray = []string{"/DATA/AppData", "/DATA/Documents", "/DATA/Downloads", "/DATA/Gallery", "/DATA/Media/Movies", "/DATA/Media/TV Shows", "/DATA/Media/Music"}
	}

	if sysType == "windows" {
		dirArray = []string{"C:\\DappsterOS\\DATA\\AppData", "C:\\DappsterOS\\DATA\\Documents", "C:\\DappsterOS\\DATA\\Downloads", "C:\\DappsterOS\\DATA\\Gallery", "C:\\DappsterOS\\DATA\\Media/Movies", "C:\\DappsterOS\\DATA\\Media\\TV Shows", "C:\\DappsterOS\\DATA\\Media\\Music"}
	}

	if sysType == "darwin" {
		dirArray = []string{"./DappsterOS/DATA/AppData", "./DappsterOS/DATA/Documents", "./DappsterOS/DATA/Downloads", "./DappsterOS/DATA/Gallery", "./DappsterOS/DATA/Media/Movies", "./DappsterOS/DATA/Media/TV Shows", "./DappsterOS/DATA/Media/Music"}
	}

	for _, v := range dirArray {
		if err := file.IsNotExistMkDir(v); err != nil {
			logger.Error("ensureDefaultDirectories", zap.Error(err))
		}
	}
}

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitorUEvent(ctx)

	go sendStorageStats()

	crontab := cron.New(cron.WithSeconds())
	if _, err := crontab.AddFunc("@every 5s", sendStorageStats); err != nil {
		logger.Error("crontab add func error", zap.Error(err))
	}

	crontab.Start()
	defer crontab.Stop()

	listener, err := net.Listen("tcp", net.JoinHostPort(localhost, "0"))
	if err != nil {
		panic(err)
	}

	// register at gateway
	apiPaths := []string{
		"/v1/usb",
		"/v1/disks",
		"/v1/storage",
		// "/v1/cloud",
		// "/v1/recover",
		// "/v1/driver",
		route.V2APIPath,
		route.V2DocPath,
	}
	for _, apiPath := range apiPaths {
		err = service.MyService.Gateway().CreateRoute(&model.Route{
			Path:   apiPath,
			Target: "http://" + listener.Addr().String(),
		})

		if err != nil {
			panic(err)
		}
	}
	go RegMsg()
	go service.MyService.Disk().InitCheck()
	v1Router := route.InitV1Router()
	v2Router := route.InitV2Router()
	v2DocRouter := route.InitV2DocRouter(_docHTML, _docYAML)

	mux := &util_http.HandlerMultiplexer{
		HandlerMap: map[string]http.Handler{
			"v1":  v1Router,
			"v2":  v2Router,
			"doc": v2DocRouter,
		},
	}

	if supported, err := daemon.SdNotify(false, daemon.SdNotifyReady); err != nil {
		logger.Error("Failed to notify systemd that local storage service is ready", zap.Error(err))
	} else if supported {
		logger.Info("Notified systemd that local storage service is ready")
	} else {
		logger.Info("This process is not running as a systemd service.")
	}

	logger.Info("LocalStorage service is listening...", zap.Any("address", listener.Addr().String()))

	server := &http.Server{
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	err = server.Serve(listener)
	if err != nil {
		panic(err)
	}
}
func RegMsg() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var events []message_bus.EventType
	events = append(events, message_bus.EventType{Name: common.ServiceName + ":merge_status", SourceID: common.ServiceName, PropertyTypeList: []message_bus.PropertyType{}})
	events = append(events, message_bus.EventType{Name: common.ServiceName + ":storage_status", SourceID: common.ServiceName, PropertyTypeList: []message_bus.PropertyType{}})
	// register at message bus
	for i := 0; i < 10; i++ {
		response, err := service.MyService.MessageBus().RegisterEventTypesWithResponse(context.Background(), events)
		if err != nil {
			logger.Error("error when trying to register one or more event types - some event type will not be discoverable", zap.Error(err))
		}
		if response != nil && response.StatusCode() != http.StatusOK {
			logger.Error("error when trying to register one or more event types - some event type will not be discoverable", zap.String("status", response.Status()), zap.String("body", string(response.Body)))
		}
		if response.StatusCode() == http.StatusOK {
			break
		}
		time.Sleep(time.Second)
	}
	// register at message bus
	for devtype, eventTypesByAction := range common.EventTypes {
		response, err := service.MyService.MessageBus().RegisterEventTypesWithResponse(ctx, lo.Values(eventTypesByAction))
		if err != nil {
			logger.Error("error when trying to register one or more event types - some event type will not be discoverable", zap.Error(err), zap.String("devtype", devtype))
		}

		if response != nil && response.StatusCode() != http.StatusOK {
			logger.Error("error when trying to register one or more event types - some event type will not be discoverable", zap.String("status", response.Status()), zap.String("body", string(response.Body)), zap.String("devtype", devtype))
		}
	}

}
