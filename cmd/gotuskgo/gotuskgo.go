package main

import (
	"os"
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/wallnutkraken/gotuskgo/bot"
	"github.com/wallnutkraken/gotuskgo/controlpanel/panel"
	"github.com/wallnutkraken/gotuskgo/server"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/dbwrap"
	"github.com/wallnutkraken/gotuskgo/tuskbrain/settings"
)

func main() {
	// Load settings
	cfg, err := settings.Load()
	if err != nil {
		// Loading failed, use default
		cfg = settings.Default
		// Then save it. If THAT fails, panic.
		if err := settings.Save(cfg); err != nil {
			panic("Error saving default settings: " + err.Error())
		}
	}
	// Connect to the database
	db, err := gorm.Open("mysql", fmt.Sprintf("%s:%s@tcp(db:3306)/%s?charset=utf8mb4", os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASSWORD"), os.Getenv("MYSQL_DATABASE")))
	if err != nil {
		panic("Failed connecting to the database " + err.Error())
	}
	// Create the gorm wrapper
	wrapper := dbwrap.New(db)
	// Start automigrate
	if err := wrapper.AutoMigrate(); err != nil {
		panic("Failed running AutoMigrate: " + err.Error())
	}

	// Create an instance of the server
	serv, err := server.New(cfg, wrapper)
	if err != nil && err != bot.ErrServiceInit {
		panic("Error starting server: " + err.Error())
	}
	// And have it run on a separate goroutine
	go serv.Start()
	// And of the gRPC control panel
	cpanel := panel.New(cfg.GRPC, serv, wrapper)
	if err := cpanel.ListenAndServe(); err != nil {
		panic("ListenAndServe error: " + err.Error())
	}
}
