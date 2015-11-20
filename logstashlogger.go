package main

import (
	log "github.com/Sirupsen/logrus"
	"gopkg.in/elazarl/goproxy.v1"
	"gopkg.in/elazarl/goproxy.v1/transport"
	"io"
	"net/http"
	"time"
)

type LogstashLogger struct {
	path  string
	c     chan *Meta
	errch chan error
}

func NewLogstashLogger(basepath string) (*LogstashLogger, error) {
	logger := &LogstashLogger{basepath, make(chan *Meta), make(chan error)}

	go func() {
		for m := range logger.c {

			if fields, err := m.ConvertToFields(); err != nil {
				log.Info("Can't write meta", err)
			} else {

				// Fix body streaming. Using a file causes `no such file or directory' because the other routine does not write the file yet`
				// User another kind of stream

				buf := []byte{}
				log.Info("HERE ===== 1")
				if _, err2 := m.Body().Read(buf); err2 != nil {
				// if _, err2 := io.LimitReader(m.Body(), 8192).Read(buf); err2 != nil {
					log.Info("==================Empty ", err2)
					log.WithFields(fields).Info("")
				} else {
					log.Info("==================bodyReader")
					time.Sleep(5 * time.Second)
					log.Info(buf)
					log.WithFields(fields).Info(string(buf))
				}
			}
		}
		log.Info("HERE ===== 2")
		// logger.errch <- f.Close()
	}()

	return logger, nil
}

func (logger *LogstashLogger) LogResp(resp *http.Response, ctx *goproxy.ProxyCtx) {
	r, w := io.Pipe()
	from := ""
	if ctx.UserData != nil {
		from = ctx.UserData.(*transport.RoundTripDetails).TCPAddr.String()
	}
	if resp == nil {
		resp = emptyResp
	} else {
		resp.Body = NewTeeReadCloser(resp.Body, NewLinkedStream(r, w))
	}
	logger.LogMeta(&Meta{
		resp: resp,
		err:  ctx.Error,
		t:    time.Now(),
		sess: ctx.Session,
		from: from,
		body: r})
}

func (logger *LogstashLogger) LogReq(req *http.Request, ctx *goproxy.ProxyCtx) {
	r, w := io.Pipe()
	if req == nil {
		req = emptyReq
	} else {
		req.Body = NewTeeReadCloser(req.Body, NewLinkedStream(r, w))
	}
	logger.LogMeta(&Meta{
		req:  req,
		err:  ctx.Error,
		t:    time.Now(),
		sess: ctx.Session,
		from: req.RemoteAddr,
		body: r})
}

func (logger *LogstashLogger) LogMeta(m *Meta) {
	logger.c <- m
}

func (logger *LogstashLogger) Close() error {
	close(logger.c)
	return <-logger.errch
}
