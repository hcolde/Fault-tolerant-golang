package main

import (
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jinzhu/gorm"
	"log"
	"os"
	"sync/atomic"
	"time"
)

type Machine struct {
	ID   int64 `gorm:'AUTO_INCREMENT'`
	Name string
	LaunchDate int64
}

var (
	startTime int64   // 启动时间
	lastTime  int64   // 最后一次调用时间
	machine   Machine // 机器实例
	sequence  int64   // 并发自增数
	callback  int64   // 时间回拨次数
	running   int32   // 是否有任务
	//collate sync.Map

	BITS = []int{41, 9, 3, 10} // 增量时间41 机器ID9 时间回拨次数3 递增序列10

	server = ":12138" // ip:端口
	databaseInfo = "账号:密码@tcp(127.0.0.1:3306)/数据库?charset=utf8&parseTime=True&loc=Local" // Mysql数据库
)

func init() {
	startTime = time.Now().UnixNano() / 1e6
}

func GeneralID() (int64, error) {
	for atomic.CompareAndSwapInt32(&running, 0, 1) == false {}
	defer atomic.StoreInt32(&running, 0)

	now := time.Now().UnixNano() / 1e6
	delta := now - startTime
	delta = (delta >> 63) ^ delta - (delta >> 63)

	if now == lastTime { // 并发
		sequence++
	} else {
		if now < lastTime { // 时间回拨
			callback++
		}
		sequence = 0
		lastTime = now
	}

	if delta >= 1 << BITS[0] {
		return -1, errors.New(fmt.Sprintf("delta time is out of 2^%d", BITS[0]))
	} else if callback >= 1 << 3 {
		return -1, errors.New(fmt.Sprintf("callback times is out of 2^%d", BITS[2]))
	} else if sequence >= 1 << 10 {
		return -1, errors.New(fmt.Sprintf("sequence number is out of 2^%d", BITS[3]))
	}

	return delta << (BITS[3] + BITS[2] + BITS[1]) + machine.ID << (BITS[3] + BITS[2]) + callback << BITS[3] + sequence, nil
}

func initMachine() error {
	db, err := gorm.Open("mysql", databaseInfo)
	if err != nil {
		return err
	}
	defer db.Close()

	db.SingularTable(true)

	machine.Name, _ = os.Hostname()
	machine.LaunchDate = startTime
	db.Create(&machine)
	return nil
}

func getID(c *gin.Context) {
	id, err := GeneralID()
	if err != nil {
		log.Println(err.Error())
	}

	//if id == -1 {
	//	log.Println(id)
	//}
	//
	//if id != -1 {
	//	if v, ok := collate.LoadOrStore(id, true); ok {
	//		log.Println(id, v)
	//	}
	//}

	c.JSON(200, gin.H{"id": id})
}

func router() *gin.Engine {
	route := gin.New()
	route.GET("/id", getID)
	return route
}

func main() {
	if err := initMachine(); err != nil {
		log.Fatal(err.Error())
		return
	}
	if machine.ID > 512 {
		log.Fatal(fmt.Sprintf("Machine id (%d) is out of limited(%d)", machine.ID, BITS[1]))
		return
	}

	route := router()
	if err := route.Run(server); err != nil {
		log.Fatal(err.Error())
		return
	}
}