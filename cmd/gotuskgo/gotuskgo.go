package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
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
	db, err := gorm.Open("sqlite3", cfg.Database.Path)
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
	if err != nil && err != bot.ErrTelegramInit {
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
