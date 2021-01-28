// Copyright 2021 Solarwinds Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:generate protoc -I ./ payload.proto --go_out=./

// Package papertrailgo is a Go library package which contains code for shipping logs to papertrail
package papertrailgo

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	syslog "github.com/RackSec/srslog"
	badger "github.com/dgraph-io/badger/v3"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"
)

const (
	keyFormat                   = "TS:%d-BODY:%s"
	defaultMaxDiskUsage         = 5    // disk usage in percentage
	defaultUltimateMaxDiskUsage = 99   // usage cannot go beyond this percentage value
	defaultBatchSize            = 1000 // records
	defaultDBLocation           = "./badger"
	cleanUpInterval             = 5 * time.Second
)

var (
	defaultWorkerCount = 10
	defaultRetention   = 24 * time.Hour
)

// LoggerInterface is the interface for all Papertrail logger types
type LoggerInterface interface {
	Log(*Payload) error
	Close() error
}

// Logger is a concrete type of LoggerInterface which collects and ships logs to Papertrail
type Logger struct {
	retentionPeriod time.Duration

	db *badger.DB

	initialDiskUsage float64

	maxDiskUsage float64

	maxWorkers int

	loopFactor *loopFactor

	loopWait chan struct{}

	syslogWriter papertrailShipper
}

// NewPapertrailLogger creates a papertrail log shipper and also returns an instance of Logger
func NewPapertrailLogger(ctx context.Context, paperTrailProtocol, paperTrailHost string, paperTrailPort int, tag, dbLocation string, retention time.Duration,
	workerCount int, maxDiskUsage float64) (*Logger, error) {
	sLogWriter, err := NewPapertailShipper(paperTrailProtocol, paperTrailHost, paperTrailPort, tag)
	if err != nil {
		err = errors.Wrap(err, "error while creating a papertrail shipper instance")
		logrus.Error(err)
		return nil, err
	}
	return NewPapertrailLoggerWithShipper(ctx, dbLocation, retention, workerCount, maxDiskUsage, sLogWriter)
}

// NewPapertrailLoggerWithShipper does some ground work and returns an instance of Logger
func NewPapertrailLoggerWithShipper(ctx context.Context, dbLocation string, retention time.Duration,
	workerCount int, maxDiskUsage float64, sLogWriter papertrailShipper) (*Logger, error) {
	if retention.Seconds() <= float64(0) {
		retention = defaultRetention
	}
	if strings.TrimSpace(dbLocation) == "" {
		dbLocation = defaultDBLocation
	}
	l := logrus.New()
	l.SetLevel(logrus.GetLevel())
	opts := badger.DefaultOptions(dbLocation).WithLogger(l)
	db, err := badger.Open(opts)
	if err != nil {
		err = errors.Wrap(err, "error while opening a local db instance")
		logrus.Error(err)
		// attempting to use a different location
		dbLocation = fmt.Sprintf("%s_%s", dbLocation, uuid.NewV4().Bytes())
		opts = badger.DefaultOptions(dbLocation).WithLogger(l)
		db, err = badger.Open(opts)
		if err != nil {
			err = errors.Wrap(err, "error while opening a local db instance")
			logrus.Error(err)
			return nil, err
		}
	}

	if workerCount <= 0 {
		workerCount = defaultWorkerCount
	}

	if maxDiskUsage <= 0 {
		maxDiskUsage = defaultMaxDiskUsage
	}

	if sLogWriter == nil {
		err = errors.New("error: given syslog writer instance is nil")
		logrus.Error(err)
		return nil, err
	}

	p := &Logger{
		retentionPeriod:  retention,
		maxWorkers:       workerCount * runtime.NumCPU(),
		maxDiskUsage:     maxDiskUsage,
		loopFactor:       newLoopFactor(true),
		db:               db,
		initialDiskUsage: diskUsage(),
		loopWait:         make(chan struct{}),

		syslogWriter: sLogWriter,
	}

	go p.flushLogs()
	go p.deleteExcess()
	go p.cleanup()
	return p, nil
}

// Log method receives log messages
func (p *Logger) Log(payload *Payload) error {
	if payload == nil || payload.Log == "" {
		err := errors.New("given payload is empty")
		logrus.Error(err)
		return err
	}
	if payload.LogTime == nil {
		payload.LogTime = ptypes.TimestampNow()
	}
	data, err := proto.Marshal(payload)
	if err != nil {
		err = errors.Wrapf(err, "error marshalling payload")
		logrus.Error(err)
		return err
	}

	if len(data) > 0 {
		guuid := uuid.NewV4()
		if err := p.db.Update(func(txn *badger.Txn) error {
			logrus.Debug("log line received, marshalled and persisting to local db")
			return txn.SetEntry(badger.NewEntry([]byte(fmt.Sprintf(keyFormat, time.Now().UnixNano(), guuid)), data).WithTTL(p.retentionPeriod))
		}); err != nil {
			err = errors.Wrapf(err, "error persisting log to local db")
			logrus.Error(err)
			return err
		}
	}
	return nil
}

func (p *Logger) sendLogs(payload *Payload) error {
	logrus.Debugf("sending log to papertrail: %+v", payload)
	ts, _ := ptypes.Timestamp(payload.GetLogTime()) // we can skip err check here
	return p.syslogWriter.Write(&SyslogPacket{
		Severity: syslog.LOG_INFO,
		Hostname: payload.GetHostname(),
		Tag:      payload.GetTag(),
		Time:     ts,
		Message:  payload.GetLog(),
	})
}

// This should be run in a routine
func (p *Logger) flushLogs() {
	defer func() {
		p.loopWait <- struct{}{}
	}()
	for p.loopFactor.getBool() {
		hose := make(chan []byte, p.maxWorkers)
		wg := new(sync.WaitGroup)

		// workers
		for i := 0; i < p.maxWorkers; i++ {
			go p.flushWorker(hose, wg)
		}

		err := p.db.View(func(txn *badger.Txn) error {
			opts := badger.DefaultIteratorOptions
			opts.PrefetchValues = false
			it := txn.NewIterator(opts)
			defer it.Close()
			for it.Rewind(); it.Valid(); it.Next() {
				item := it.Item()
				k := make([]byte, len(item.Key()))
				copy(k, item.Key())
				wg.Add(1)
				hose <- k
			}
			return nil
		})
		if err != nil {
			err = errors.Wrapf(err, "flush logs - Error reading keys from db")
			logrus.Warn(err)
		}
		wg.Wait()
		close(hose)
		time.Sleep(50 * time.Millisecond)
	}
}

func (p *Logger) flushWorker(hose chan []byte, wg *sync.WaitGroup) {
	for key := range hose {
		err := p.db.Update(func(txn *badger.Txn) error {
			item, err := txn.Get(key)
			if err != nil {
				if err == badger.ErrKeyNotFound {
					return nil
				}
				return err
			}
			var val []byte
			val, err = item.ValueCopy(val)
			if err != nil {
				if err == badger.ErrKeyNotFound {
					return nil
				}
				return err
			}
			payload := &Payload{}
			if err := proto.Unmarshal(val, payload); err != nil {
				err = errors.Wrap(err, "unmarshal error")
				return err
			}
			if err := p.sendLogs(payload); err != nil {
				err = errors.Wrapf(err, "error sending log with key: %s, which will be reattempted later", key)
				return err
			}

			logrus.Debugf("flushLogs, delete key: %s", key)
			err = txn.Delete(key)
			if err != nil {
				return err
			}
			return nil
		})
		if err != nil {
			err = errors.Wrapf(err, "Error while deleting key: %s", key)
			logrus.Warn(err)
		}
		wg.Done()
	}
}

func (p *Logger) deleteExcess() {
	for p.loopFactor.getBool() {
		currentUsage := diskUsage()
		// if p.log.VerbosityLevel(config.DebugLevel) {
		// 	p.log.Infof("Current disk usage: %.2f %%", currentUsage)
		// 	p.log.Infof("DB folder size: %.2f MB", computeDirectorySizeInMegs(dbLocation))
		// }
		if currentUsage > p.initialDiskUsage+p.maxDiskUsage || currentUsage > defaultUltimateMaxDiskUsage {
			// delete from beginning
			iterations := defaultBatchSize
			err := p.db.View(func(txn *badger.Txn) error {
				opts := badger.DefaultIteratorOptions
				opts.PrefetchValues = false
				it := txn.NewIterator(opts)
				defer it.Close()
				for it.Rewind(); it.Valid(); it.Next() {
					item := it.Item()
					k := make([]byte, len(item.Key()))
					copy(k, item.Key())
					_ = txn.Delete(k)
					iterations--
					if iterations < 0 {
						break
					}
				}
				return nil
			})
			if err != nil {
				err = errors.Wrapf(err, "deleteExcess - Error while deleting")
				logrus.Warn(err)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
}

// Close - closes the Logger instance
func (p *Logger) Close() error {
	logrus.Info("closing papertrail logger instance")
	p.loopFactor.setBool(false)
	defer func() {
		close(p.loopWait)
		logrus.Info("papertrail instance closed")
	}()
	_ = p.syslogWriter.Close()
	// if err := p.syslogWriter.Close(); err != nil {
	// 	err = errors.Wrapf(err, "error while closing syslog writer")
	// 	logrus.Error(err)
	// 	return err
	// }

	time.Sleep(time.Second)
	if p.db != nil {
		if err := p.db.Close(); err != nil {
			err = errors.Wrapf(err, "error while closing DB")
			logrus.Error(err)
			return err
		}
	}
	<-p.loopWait
	return nil
}

func (p *Logger) cleanup() {
	for p.loopFactor.getBool() {
		if p.db != nil {
			logrus.Debug("cleanup - running GC")
			//_ = p.db.PurgeOlderVersions()
			_ = p.db.RunValueLogGC(0.99)
		}
		time.Sleep(cleanUpInterval)
	}
}

func diskUsage() float64 {
	var stat syscall.Statfs_t
	wd, _ := os.Getwd()
	_ = syscall.Statfs(wd, &stat)
	avail := stat.Bavail * uint64(stat.Bsize)
	used := stat.Blocks * uint64(stat.Bsize)
	return (float64(used) / float64(used+avail)) * 100
}

//func computeDirectorySizeInMegs(fullPath string) float64 {
//	var sizeAccumulator int64
//	filepath.Walk(fullPath, func(path string, file os.FileInfo, err error) error {
//		if !file.IsDir() {
//			atomic.AddInt64(&sizeAccumulator, file.Size())
//		}
//		return nil
//	})
//	return float64(atomic.LoadInt64(&sizeAccumulator)) / (1024 * 1024)
//}
