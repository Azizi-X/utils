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

type Event struct {
	Type       string `json:"type"`
	Properties any    `json:"properties"`
}

type Analytics struct {
	debounce func(f func())
	callback func(t string, properties []byte, raw []byte) error
	onFlush  func(events []Event)
	backend  string
	pending  []Event
	mu       sync.Mutex
}

func (a *Analytics) Flush() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	pending := a.pending
	a.pending = nil

	if a.backend == "" && a.onFlush == nil {
		return errors.New("backend is not set")
	} else if len(pending) == 0 {
		return nil
	}

	if a.onFlush != nil {
		go a.onFlush(pending)

		if a.backend == "" {
			return nil
		}
	}

	payload, err := json.Marshal(pending)
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

func (a *Analytics) Public(t string, properties any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.backend == "" && a.onFlush == nil {
		return errors.New("backend is not set")
	} else if a.debounce == nil {
		return errors.New("debounce is not set")
	}

	a.pending = append(a.pending, Event{
		Type:       t,
		Properties: properties,
	})

	a.debounce(func() {
		a.Flush()
	})

	return nil
}

func (a *Analytics) SetOnFlush(fn func(events []Event)) *Analytics {
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

func (a *Analytics) WithDebounce(after time.Duration) *Analytics {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.debounce = utils.NewDebouncer(after)
	return a
}

func (a *Analytics) SetCallback(callback func(t string, properties []byte, raw []byte) error) *Analytics {
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

		go a.callback(t, properties, value)

	}, "events"); err != nil {
		return err
	}

	return err
}

func NewAnalytics() *Analytics {
	return &Analytics{}
}
