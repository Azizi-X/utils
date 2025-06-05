package analytics

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/Azizi-X/utils"
	"github.com/buger/jsonparser"
)

const timeout = 10 * time.Second

var defaultHTTP = http.Client{Timeout: timeout, Transport: &http.Transport{
	ResponseHeaderTimeout: timeout,
}}

type Handle struct {
	Type       string `json:"type"`
	Properties any    `json:"properties"`
	Raw        []byte `json:"raw"`
}

type Event struct {
	Type       string `json:"type"`
	Properties any    `json:"properties"`
}

type Sending struct {
	Token  string  `json:"token,omitempty"`
	Events []Event `json:"events"`
}

type Analytics struct {
	debounce func(f func())
	callback func(handle Handle) error
	onFlush  func(sending Sending) error
	backend  string
	sending  Sending
	mu       sync.Mutex
}

func (a *Analytics) Flush() error {
	a.mu.Lock()

	defer func() {
		a.sending.Events = nil
		a.mu.Unlock()
	}()

	if a.backend == "" && a.onFlush == nil {
		return errors.New("backend is not set")
	} else if len(a.sending.Events) == 0 {
		return nil
	}

	if a.onFlush != nil {
		err := a.onFlush(a.sending)

		if err != nil {
			return err
		}

		if a.backend == "" {
			return nil
		}
	}

	payload, err := json.Marshal(a.sending)
	if err != nil {
		return err
	}

	resp, err := defaultHTTP.Post(a.backend, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return nil
}

func (a *Analytics) Publish(t string, properties any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.backend == "" && a.onFlush == nil {
		return errors.New("backend is not set")
	} else if a.debounce == nil {
		return errors.New("debounce is not set")
	}

	a.sending.Events = append(a.sending.Events, Event{
		Type:       t,
		Properties: properties,
	})

	a.debounce(func() {
		a.Flush()
	})

	return nil
}

func (a *Analytics) SetOnFlush(fn func(sending Sending) error) *Analytics {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.onFlush = fn
	return a
}

func (a *Analytics) SetBackend(backend string) *Analytics {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.backend = backend
	return a
}

func (a *Analytics) SetToken(token string) *Analytics {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.sending.Token = token
	return a
}

func (a *Analytics) SetDebounce(after time.Duration) *Analytics {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.debounce = utils.NewDebouncer(after)
	return a
}

func (a *Analytics) SetCallback(callback func(handle Handle) error) *Analytics {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.callback = callback
	return a
}

func (a *Analytics) Handle(data []byte) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.callback == nil {
		return errors.New("callback is not set")
	}

	var err error

	if _, err := jsonparser.ArrayEach(data, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			return
		}

		t, _ := jsonparser.GetString(value, "type")
		properties, _, _, _ := jsonparser.Get(value, "properties")

		go a.callback(Handle{
			Type:       t,
			Properties: properties,
			Raw:        value,
		})

	}, "events"); err != nil {
		return err
	}

	return err
}

func NewAnalytics() *Analytics {
	return &Analytics{}
}
