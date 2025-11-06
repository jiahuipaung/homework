package main

import (
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"

	"github.com/flashcatcloud/fc-stash/app/stash/srvs"
	"github.com/flashcatcloud/fc-stash/config"
	"github.com/flashcatcloud/fc-stash/handler/plugin"
	"github.com/flashcatcloud/fc-stash/handler/plugin/enrich/g_account"
	"github.com/flashcatcloud/fc-stash/handler/plugin/hundsun"
	"github.com/flashcatcloud/fc-stash/handler/plugin/task"
	"github.com/flashcatcloud/fc-stash/transform/field_format"
	"github.com/flashcatcloud/fc-stash/transform/text_parser"
	"github.com/flashcatcloud/go-pkg/srv"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	cfg         = kingpin.Flag("config", "path to config file").Short('c').String()
	logDir      = kingpin.Flag("logDir", "path to the logs dir").Short('l').String()
	showVersion = kingpin.Flag("version", "show build version").Short('v').Bool()
)

func main() {
	kingpin.Parse()

	if *showVersion {
		fmt.Println(srv.VERSION)
		os.Exit(0)
	}

	srv.MODULE = "stash"
	config.MustLoad(*cfg, *logDir)

	srv.Inits(srv.MODULE)
	defer srv.Shutdown()

	_ = field_format.ResetUriPathPatternFile(config.C.Stash.GrokPattern, config.C.Stash.ReplacePattern)

	// 初始化geoip词表
	if err := text_parser.InitGeoIPParser(config.C.Stash.GeoIPFile); err != nil {
		log.Println("init geoip reader failed:", err)
	}

	// pprof server
	if config.C.HTTP.PProf.Enabled {
		go srv.StartPProf(config.C.HTTP.PProf.Port, config.C.Srv.Mode)
		defer srv.StopPProf()
	}

	// http server
	restSrv := srv.New(
		config.C.HTTP.Port,
		config.C.HTTP.IsTLS,
		config.C.Srv.Mode,
	)
	buildRestRoutes(restSrv)
	go restSrv.Start()
	defer restSrv.Stop()

	if len(config.C.Stash.InsightService.Webapis) == 0 && len(config.C.Stash.N9eService.Webapis) == 0 {
		log.Fatalln("etc/stash.yml no insight service or n9e service")
	}
	// common workers
	if !config.C.Stash.DisableCommonLog {
		srv.RegistWorker("pipeline_worker_manage",
			srvs.PipelineWorkerManageStart, srvs.PipelineWorkerManageStop)
		go srv.StartWorkers()
		defer srv.StopWorkers()
	}

	// plugin worker
	var hundsuns []config.PluginPipelineConfig
	var enrichs []config.PluginPipelineConfig
	for i := range config.C.Stash.Plugins {
		conf := config.C.Stash.Plugins[i]
		if conf.Type == plugin.PluginTypeHundsumRequestLog {
			hundsuns = append(hundsuns, conf)
		}
		if conf.Type == plugin.PluginTypeEnrichGAccount {
			enrichs = append(enrichs, conf)
		}
	}
	if len(hundsuns) > 0 {
		// 启动失败只报错, 不阻塞程序
		if err := hundsun.StartPipelineWorker(hundsuns); err != nil {
			log.Println("start plugin pipiline failed:" + err.Error())
		}
		defer hundsun.StopPipelineWorker()
	}
	if len(enrichs) > 0 {
		// 启动失败只报错, 不阻塞程序
		if err := g_account.StartPipelineWorker(enrichs); err != nil {
			log.Println("start plugin pipiline failed:" + err.Error())
		}
		defer g_account.StopPipelineWorker()
	}
	if len(hundsuns) > 0 || len(enrichs) > 0 {
		// 过期清理
		var opts []config.PluginPipelineConfig
		if len(hundsuns) > 0 {
			opts = append(opts, hundsuns...)
		}
		if len(enrichs) > 0 {
			opts = append(opts, enrichs...)
		}
		// TODO: 切换到pipeline_handler的逻辑
		go task.StartElasticRolloverCron(opts)
	}
	// wait for stop signal
	<-srv.QuitSignal()
}
