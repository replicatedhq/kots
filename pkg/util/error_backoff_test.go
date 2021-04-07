package util

import (
	"testing"
	"time"

	"github.com/pkg/errors"
	"gotest.tools/assert"
)

func TestErrorBackoff_OnError(t *testing.T) {
	type fields struct {
		MinPeriod time.Duration
		MaxPeriod time.Duration
	}
	type event struct {
		err error
		dur time.Duration
	}
	tests := []struct {
		name    string
		fields  fields
		events  []event
		expects []int
	}{
		{
			name: "basic",
			fields: fields{
				MinPeriod: 4 * time.Millisecond,
				MaxPeriod: 40 * time.Millisecond,
			},
			events: []event{
				{errors.New("error 1"), 2 * time.Millisecond},
				{errors.New("error 1"), 2 * time.Millisecond},
				{errors.New("error 1"), 2 * time.Millisecond},
				{errors.New("error 1"), 2 * time.Millisecond},
				{errors.New("error 1"), 2 * time.Millisecond},
				{errors.New("error 2"), 2 * time.Millisecond},
				{errors.New("error 1"), 2 * time.Millisecond},
				{errors.New("error 2"), 2 * time.Millisecond},
				{errors.New("error 3"), 2 * time.Millisecond},
				{errors.New("error 3"), 2 * time.Millisecond},
				{errors.New("error 3"), 2 * time.Millisecond},
			},
			expects: []int{0, 2, 4, 5, 6, 7, 8, 10},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ErrorBackoff{
				MinPeriod: tt.fields.MinPeriod,
				MaxPeriod: tt.fields.MaxPeriod,
			}
			actual := []int{}
			fn := func(i int) func() {
				return func() {
					actual = append(actual, i)
				}
			}
			for i, event := range tt.events {
				r.OnError(event.err, fn(i))
				time.Sleep(event.dur)
			}
			assert.DeepEqual(t, tt.expects, actual)
		})
	}
}
