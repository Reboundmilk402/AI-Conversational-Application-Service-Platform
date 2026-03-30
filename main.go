package main

import (
	"GopherAI/common/aihelper"
	"GopherAI/common/mysql"
	"GopherAI/common/rabbitmq"
	"GopherAI/common/redis"
	"GopherAI/config"
	"GopherAI/dao/message"
	sessiondao "GopherAI/dao/session"
	"GopherAI/router"
	"fmt"
	"log"
)

func StartServer(addr string, port int) error {
	r := router.InitRouter()
	return r.Run(fmt.Sprintf("%s:%d", addr, port))
}

func readDataFromDB() error {
	manager := aihelper.GetGlobalManager()
	msgs, err := message.GetAllMessages()
	if err != nil {
		return err
	}

	for i := range msgs {
		m := &msgs[i]

		sessionRecord, err := sessiondao.GetSessionByID(m.SessionID)
		if err != nil {
			log.Printf("[readDataFromDB] failed to load session=%s: %v", m.SessionID, err)
			continue
		}

		modelType := sessionRecord.ModelType
		if modelType == "" {
			modelType = "1"
		}

		helper, err := manager.GetOrCreateAIHelper(
			m.UserName,
			m.SessionID,
			modelType,
			aihelper.BuildModelConfig(modelType),
		)
		if err != nil {
			log.Printf("[readDataFromDB] failed to create helper for user=%s session=%s: %v", m.UserName, m.SessionID, err)
			continue
		}

		log.Println("readDataFromDB init:", helper.SessionID)
		helper.AddMessage(m.Content, m.UserName, m.IsUser, false)
	}

	log.Println("AIHelperManager init success")
	return nil
}

func main() {
	conf := config.GetConfig()
	host := conf.MainConfig.Host
	port := conf.MainConfig.Port

	if err := mysql.InitMysql(); err != nil {
		log.Println("InitMysql error, " + err.Error())
		return
	}

	readDataFromDB()

	redis.Init()
	log.Println("redis init success")
	rabbitmq.InitRabbitMQ()
	log.Println("rabbitmq init success")

	if err := StartServer(host, port); err != nil {
		panic(err)
	}
}
