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
				MinPeriod: 30 * time.Millisecond,
				MaxPeriod: 400 * time.Millisecond,
			},
			events: []event{
				0:  {errors.New("error 1"), 20 * time.Millisecond},
				1:  {errors.New("error 1"), 20 * time.Millisecond},
				2:  {errors.New("error 1"), 20 * time.Millisecond},
				3:  {errors.New("error 1"), 20 * time.Millisecond},
				4:  {errors.New("error 1"), 20 * time.Millisecond},
				5:  {errors.New("error 2"), 20 * time.Millisecond},
				6:  {errors.New("error 1"), 20 * time.Millisecond},
				7:  {errors.New("error 2"), 20 * time.Millisecond},
				8:  {errors.New("error 3"), 20 * time.Millisecond},
				9:  {errors.New("error 3"), 20 * time.Millisecond},
				10: {errors.New("error 3"), 20 * time.Millisecond},
			},
			expects: []int{0, 2, 5, 6, 7, 8, 10},
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
