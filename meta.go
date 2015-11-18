package main

import (
	"github.com/Sirupsen/logrus"
	"io"
	"net/http"
	"net/http/httputil"
	"time"
)

type Meta struct {
	req      *http.Request
	resp     *http.Response
	err      error
	t        time.Time
	sess     int64
	bodyPath string
	from     string
}

func (m *Meta) WriteTo(w io.Writer) (nr int64, err error) {
	if m.req != nil {
		fprintf(&nr, &err, w, "Type: request\r\n")
	} else if m.resp != nil {
		fprintf(&nr, &err, w, "Type: response\r\n")
	}
	fprintf(&nr, &err, w, "ReceivedAt: %v\r\n", m.t)
	fprintf(&nr, &err, w, "Session: %d\r\n", m.sess)
	fprintf(&nr, &err, w, "From: %v\r\n", m.from)
	if m.err != nil {
		// note the empty response
		fprintf(&nr, &err, w, "Error: %v\r\n\r\n\r\n\r\n", m.err)
	} else if m.req != nil {
		fprintf(&nr, &err, w, "\r\n")
		buf, err2 := httputil.DumpRequest(m.req, false)
		if err2 != nil {
			return nr, err2
		}
		write(&nr, &err, w, buf)
	} else if m.resp != nil {
		fprintf(&nr, &err, w, "\r\n")
		buf, err2 := httputil.DumpResponse(m.resp, false)
		if err2 != nil {
			return nr, err2
		}
		write(&nr, &err, w, buf)
	}
	return
}

func (m *Meta) ConvertToFields() (logrus.Fields, error) {
	t := ""
	if m.req != nil {
		t = "request"
	} else if m.resp != nil {
		t = "response"
	}

	headers := ""
	e := ""
	if m.err != nil {
		// note the empty response
		e = sprintf("%v", m.err)
	} else if m.req != nil {
		buf, err := httputil.DumpRequest(m.req, false)
		if err != nil {
			return nil, err
		}
		headers = string(buf)
	} else if m.resp != nil {
		buf, err := httputil.DumpResponse(m.resp, false)
		if err != nil {
			return nil, err
		}
		headers = string(buf)
	}

	fields := logrus.Fields{
		"type":        t,
		"received_at": sprintf("%v", m.t),
		"session":     sprintf("%d", m.sess),
		"from":        sprintf("%v", m.from),
		"error":       e,
		"headers":     headers,
	}

	return fields, nil
}
