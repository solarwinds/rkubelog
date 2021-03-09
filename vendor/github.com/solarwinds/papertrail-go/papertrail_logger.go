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
	"syscall"
	"time"

	syslog "github.com/RackSec/srslog"
	"github.com/golang/protobuf/ptypes"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	"github.com/sirupsen/logrus"
	bolt "go.etcd.io/bbolt"
	"google.golang.org/protobuf/proto"
)

const (
	keyFormat                   = "TS:%d-BODY:%s"
	defaultMaxDiskUsage         = 5    // disk usage in percentage
	defaultUltimateMaxDiskUsage = 99   // usage cannot go beyond this percentage value
	defaultBatchSize            = 1000 // records
	defaultDBLocation           = "./db"
	cleanUpInterval             = 1 * time.Minute
)

var (
	defaultWorkerCount = 10
	defaultRetention   = 24 * time.Hour
	defaultBucketName  = []byte("rKubeLog")
)

// LoggerInterface is the interface for all Papertrail logger types
type LoggerInterface interface {
	Log(*Payload) error
	Close() error
}

// Logger is a concrete type of LoggerInterface which collects and ships logs to Papertrail
type Logger struct {
	retentionPeriod time.Duration

	db *bolt.DB

	initialDiskUsage float64

	maxDiskUsage float64

	maxWorkers int

	loopFactor *loopFactor

	loopWait chan struct{}

	dbLocation string

	syslogWriter papertrailShipper

	writeChan chan []byte
}

type kv struct {
	k []byte
	v []byte
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
	db, err := openDB(dbLocation)
	if err != nil {
		// attempting to use a different location
		dbLocation = fmt.Sprintf("%s_%s", dbLocation, uuid.NewV4().Bytes())
		db, err = openDB(dbLocation)
		if err != nil {
			return nil, err
		}
	}

	if err := db.Update(func(t *bolt.Tx) error {
		_, err := t.CreateBucketIfNotExists(defaultBucketName)
		if err != nil {
			return err
		}
		return nil
	}); err != nil {
		err = errors.Wrap(err, "error creating a bucket")
		logrus.Error(err)
		return nil, err
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
		dbLocation:       dbLocation,
		syslogWriter:     sLogWriter,
		writeChan:        make(chan []byte, 1000),
	}

	go p.flushLogs()
	go p.deleteExcess()
	go p.write()
	// go p.cleanup()
	return p, nil
}

func openDB(dbLocation string) (*bolt.DB, error) {
	db, err := bolt.Open(dbLocation, 0666, nil)
	if err != nil {
		err = errors.Wrap(err, "error while opening a local db instance")
		logrus.Error(err)
		return nil, err
	}
	// db.MaxBatchSize = 100
	return db, nil
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
	p.writeChan <- data
	return nil
}

// write method persists log messages
func (p *Logger) write() {
	for data := range p.writeChan {
		if len(data) > 0 {
			guuid := uuid.NewV4()
			// if err := p.db.Update(func(txn *badger.Txn) error {
			// 	logrus.Debug("log line received, marshalled and persisting to local db")
			// 	return txn.SetEntry(badger.NewEntry([]byte(fmt.Sprintf(keyFormat, time.Now().UnixNano(), guuid)), data).WithTTL(p.retentionPeriod))
			// })
			retryCount := 3
		RETRY:
			if retryCount > 0 {
				ts := time.Now()
				if err := p.db.Update(func(t *bolt.Tx) error {
					var err error
					b := t.Bucket(defaultBucketName)
					if b == nil {
						b, err = t.CreateBucketIfNotExists(defaultBucketName)
						if err != nil {
							return err
						}
					}
					return b.Put([]byte(fmt.Sprintf(keyFormat, time.Now().UnixNano(), guuid)), data)
				}); err != nil {
					if errors.Is(err, bolt.ErrDatabaseNotOpen) || errors.Is(err, bolt.ErrTimeout) {
						p.db, err = openDB(p.dbLocation)
						if err != nil {
							logrus.Error(err)
							continue
						}
						retryCount--
						logrus.Debugf("retrying.... count: %d", retryCount)
						goto RETRY
					}
					err = errors.Wrapf(err, "error persisting log to local db")
					logrus.Error(err)
					continue
				}
				logrus.Debugf("time taken for write: %f seconds", time.Since(ts).Seconds())
			} else {
				logrus.Error("Log write - retrying count exceeded... giving up")
			}
		}
	}
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
		retryCount := 3
	RETRY:
		if retryCount > 0 {
			if err := p.db.Update(func(t *bolt.Tx) error {
				var err error
				b := t.Bucket(defaultBucketName)
				if b == nil {
					b, err = t.CreateBucketIfNotExists(defaultBucketName)
					if err != nil {
						return err
					}
				}

				c := b.Cursor()
				for k, v := c.First(); k != nil; k, v = c.Next() {
					kk := make([]byte, len(k))
					vv := make([]byte, len(v))
					copy(kk, k)
					copy(vv, v)
					payload := &Payload{}
					if err := proto.Unmarshal(vv, payload); err != nil {
						err = errors.Wrap(err, "unmarshal error")
						logrus.Error(err)
						continue
					}
					if err := p.sendLogs(payload); err != nil {
						err = errors.Wrapf(err, "error sending log with key: %s, which will be reattempted later", kk)
						logrus.Error(err)
						continue
					}

					logrus.Debugf("flushLogs, delete key: %s", kk)
					if err := b.Delete(kk); err != nil {
						err = errors.Wrapf(err, "error delete key: %s, which will be reattempted later", kk)
						logrus.Error(err)
					}
				}

				return nil

			}); err != nil {
				if errors.Is(err, bolt.ErrDatabaseNotOpen) || errors.Is(err, bolt.ErrTimeout) {
					p.db, err = openDB(p.dbLocation)
					if err != nil {
						logrus.Warn(err)
						time.Sleep(50 * time.Millisecond)
						continue
					}
					retryCount--
					logrus.Debugf("DB iterate - retrying.... count: %d", retryCount)
					goto RETRY
				}
				err = errors.Wrapf(err, "flush logs - iterator error")
				logrus.Warn(err)
			}
		}
		time.Sleep(50 * time.Millisecond)
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

			if err := p.db.Batch(func(t *bolt.Tx) error {
				b, err := t.CreateBucketIfNotExists(defaultBucketName)
				if err != nil {
					return err
				}

				c := b.Cursor()
				for k, _ := c.First(); k != nil; k, _ = c.Next() {
					if err := b.Delete(k); err != nil {
						err = errors.Wrapf(err, "deleteExcess - Error while deleting")
						logrus.Warn(err)
					}

					iterations--
					if iterations < 0 {
						break
					}
				}
				return nil
			}); err != nil {
				err = errors.Wrapf(err, "deleteExcess - Batch Error while deleting")
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
	time.Sleep(time.Second)
	close(p.writeChan)
	_ = p.syslogWriter.Close()
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
