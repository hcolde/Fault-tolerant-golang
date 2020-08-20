package faultTolerant

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

type Bits struct {
	Delta int
	Mac   int
	Callback int
	Sequence int
	Host string
	DB string
}

var (
	startTime int64   // 启动时间
	lastTime  int64   // 最后一次调用时间
	machine   Machine // 机器实例
	sequence  int64   // 并发自增数
	callback  int64   // 时间回拨次数
	running   int32   // 是否有任务
	bits Bits
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

	if delta >= 1 << bits.Delta {
		return -1, errors.New(fmt.Sprintf("delta time is out of 2^%d", bits.Delta))
	} else if callback >= 1 << bits.Callback {
		return -1, errors.New(fmt.Sprintf("callback times is out of 2^%d", bits.Callback))
	} else if sequence >= 1 << bits.Sequence {
		return -1, errors.New(fmt.Sprintf("sequence number is out of 2^%d", bits.Sequence))
	}

	return delta << (bits.Mac + bits.Callback + bits.Sequence) +
		machine.ID << (bits.Callback + bits.Sequence) +
		callback << bits.Sequence + sequence, nil
}

func initMachine() error {
	db, err := gorm.Open("mysql", bits.DB)
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

	c.JSON(200, gin.H{"id": id})
}

func router() *gin.Engine {
	route := gin.New()
	route.GET("/id", getID)
	return route
}

func Run(param Bits) error {
	if param.Delta + param.Mac + param.Callback + param.Sequence != 63 {
		return errors.New("please don't embarrass me")
	}

	bits = param

	if err := initMachine(); err != nil {
		return err
	}
	if machine.ID >= 1 << bits.Mac {
		return errors.New(fmt.Sprintf("machine id (%d) is out of limited(%d)", machine.ID, bits.Mac))
	}

	route := router()
	if err := route.Run(bits.Host); err != nil {
		return err
	}
	return nil
}