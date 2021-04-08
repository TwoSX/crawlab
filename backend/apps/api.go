package apps

import (
	"context"
	"fmt"
	"github.com/apex/log"
	"github.com/crawlab-team/crawlab-core/config"
	"github.com/crawlab-team/crawlab-core/controllers"
	"github.com/crawlab-team/crawlab-core/middlewares"
	"github.com/crawlab-team/crawlab-core/models"
	"github.com/crawlab-team/crawlab-core/routes"
	"github.com/crawlab-team/crawlab-db/mongo"
	"github.com/crawlab-team/crawlab-db/redis"
	"github.com/crawlab-team/go-trace"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Api struct {
	app *gin.Engine
}

func (app *Api) Init() {
	// initialize config
	_ = app.initModule("config", config.InitConfig)

	// initialize mongo
	_ = app.initModule("mongo", mongo.InitMongo)

	// initialize redis
	_ = app.initModule("redis", redis.InitRedis)

	// initialize model services
	_ = app.initModule("modeServices", models.InitModelServices)

	// initialize controllers
	_ = app.initModule("controllers", controllers.InitControllers)

	// initialize middlewares
	_ = app.initModuleWithApp("middlewares", middlewares.InitMiddlewares)

	// initialize routes
	_ = app.initModuleWithApp("routes", routes.InitRoutes)
}

func (app *Api) Run() {
	host := viper.GetString("server.host")
	port := viper.GetString("server.port")
	address := net.JoinHostPort(host, port)
	srv := &http.Server{
		Handler: app.app,
		Addr:    address,
	}
	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				log.Error("run server error:" + err.Error())
			} else {
				log.Info("server graceful down")
			}
		}
	}()
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	ctx2, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx2); err != nil {
		log.Error("run server error:" + err.Error())
	}
}

func (app *Api) initModuleWithApp(name string, fn func(app *gin.Engine) error) (err error) {
	return app.initModule(name, func() error {
		return fn(app.app)
	})
}

func (app *Api) initModule(name string, fn func() error) (err error) {
	if err := fn(); err != nil {
		log.Error(fmt.Sprintf("init %s error: %s", name, err.Error()))
		_ = trace.TraceError(err)
		panic(err)
	}
	log.Info(fmt.Sprintf("initialized %s successfully", name))
	return nil
}

func NewApi() *Api {
	app := gin.New()
	return &Api{
		app: app,
	}
}
