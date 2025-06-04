package analytics

import (
	"time"

	"github.com/Azizi-X/utils"
	"github.com/buger/jsonparser"
)

type analytics struct {
	debounce func(f func())
	callback func(t string, properties []byte, raw []byte)
}

func (a *analytics) WithDebounce(after time.Duration) *analytics {
	a.debounce = utils.NewDebouncer(after)
	return a
}

func (a *analytics) SetCallback(callback func(t string, properties []byte, raw []byte)) *analytics {
	a.callback = callback
	return a
}

func (a *analytics) Handle(data []byte) error {
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

func NewAnalytics() *analytics {
	return &analytics{}
}
