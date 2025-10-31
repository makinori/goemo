package emocache

import (
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

type Data[T any] struct {
	Key      string
	CronSpec string
	Current  T
	Updated  time.Time
	// do not use. this gets fresh data. use Current
	Retrieve func() (T, error)
}

type DataInterface interface {
	init(c *cron.Cron)
}

func (data *Data[T]) getFresh() {
	// parse cron spec so we can get an expire time
	schedule, err := cron.ParseStandard(data.CronSpec)
	if err != nil {
		slog.Error("failed to parse cron spec", "err", err.Error())
		os.Exit(1)
		// exit program entirely
	}

	expires := schedule.Next(time.Now())

	// get data
	freshData, err := data.Retrieve()
	if err != nil {
		slog.Error(
			"failed to get data",
			"key", data.Key, "err", err.Error(),
		)
		return
	}

	data.Current = freshData
	data.Updated = time.Now()

	err = setCache(data.Key, cacheData[T]{
		Data:    data.Current,
		Updated: data.Updated,
		Expires: expires,
	})
	if err != nil {
		slog.Error(
			"failed to set cache",
			"key", data.Key, "err", err.Error(),
		)
	}
}

func (data *Data[T]) init(c *cron.Cron) {
	// try from cache
	cache, err := getCache[T](data.Key)
	if err != nil {
		// if expires will also error
		slog.Info("fetching fresh", "key", data.Key)
		data.getFresh()
	} else {
		slog.Info("already cached", "key", data.Key)
		data.Current = cache.Data
		data.Updated = cache.Updated
	}

	// setup cron
	c.AddFunc(data.CronSpec, func() {
		data.getFresh()
	})

	// slog.Println("starting cron for " + cachedData.Key)
}

func Init(cacheDir string, dataInterfaces []DataInterface) {
	currentCacheDir = cacheDir

	c := cron.New()

	var wg sync.WaitGroup

	for _, data := range dataInterfaces {
		wg.Go(func() {
			data.init(c)
		})
	}

	wg.Wait()

	c.Start()
}
