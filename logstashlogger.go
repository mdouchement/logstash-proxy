package main

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"gopkg.in/elazarl/goproxy.v1"
	"gopkg.in/elazarl/goproxy.v1/transport"
	"net/http"
	"path"
	"time"
	// "io"
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

				bodyPath := path.Join(logger.path, fmt.Sprintf("%v_resp", fields["session"]))
				buf := []byte{}
				if _, err2 := NewFileStream(bodyPath).Read(buf); err2 != nil {
					// if _, err2 := io.LimitReader(bodyReader, 8192).Read(buf); err2 != nil {
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
		// logger.errch <- f.Close()
	}()

	return logger, nil
}

func (logger *LogstashLogger) LogResp(resp *http.Response, ctx *goproxy.ProxyCtx) {
	body := path.Join(logger.path, fmt.Sprintf("%d_resp", ctx.Session))
	from := ""
	if ctx.UserData != nil {
		from = ctx.UserData.(*transport.RoundTripDetails).TCPAddr.String()
	}
	if resp == nil {
		resp = emptyResp
	} else {
		resp.Body = NewTeeReadCloser(resp.Body, NewFileStream(body))
	}
	logger.LogMeta(&Meta{
		resp: resp,
		err:  ctx.Error,
		t:    time.Now(),
		sess: ctx.Session,
		from: from})
}

func (logger *LogstashLogger) LogReq(req *http.Request, ctx *goproxy.ProxyCtx) {
	body := path.Join(logger.path, fmt.Sprintf("%d_req", ctx.Session))
	if req == nil {
		req = emptyReq
	} else {
		req.Body = NewTeeReadCloser(req.Body, NewFileStream(body))
	}
	logger.LogMeta(&Meta{
		req:  req,
		err:  ctx.Error,
		t:    time.Now(),
		sess: ctx.Session,
		from: req.RemoteAddr})
}

func (logger *LogstashLogger) LogMeta(m *Meta) {
	logger.c <- m
}

func (logger *LogstashLogger) Close() error {
	close(logger.c)
	return <-logger.errch
}
