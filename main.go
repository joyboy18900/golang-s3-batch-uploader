package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"golang-s3-batch-uploader/handler"
	"golang-s3-batch-uploader/logs"
	"golang-s3-batch-uploader/repository"
	"golang-s3-batch-uploader/service"

	"github.com/gofiber/fiber/v2"
	"github.com/spf13/viper"
)

func main() {
	initConfig()

	ctx := context.Background()
	uploader, err := repository.NewUploaderS3(
		ctx,
		viper.GetString("s3.region"),
		viper.GetString("s3.endpoint"),
		viper.GetString("s3.bucket"),
		viper.GetBool("s3.auto_create_bucket"),
	)
	if err != nil {
		logs.Error(err)
		os.Exit(1)
	}

	batchSvc := service.NewBatchService(uploader, viper.GetInt("batch.worker_count"))
	batchHdlr := handler.NewBatchHandler(batchSvc)

	app := fiber.New()
	app.Post("/batches", batchHdlr.Run)

	port := viper.GetString("app.port")
	logs.Info("server started on port " + port)
	if err := app.Listen(":" + port); err != nil {
		logs.Error(err)
		os.Exit(1)
	}
}

func initConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("read config: %w", err))
	}
}
