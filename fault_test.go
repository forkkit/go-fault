package fault

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNewFault tests NewFault().
func TestNewFault(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		giveInjector Injector
		giveOptions  []FaultOption
		wantFault    *Fault
		wantErr      error
	}{
		{
			name:         "valid",
			giveInjector: newTestInjector(false),
			giveOptions: []FaultOption{
				WithEnabled(true),
				WithInjectPercent(1.0),
				WithPathBlacklist([]string{"/donotinject"}),
				WithPathWhitelist([]string{"/onlyinject"}),
				WithRandSeed(100),
			},
			wantFault: &Fault{
				enabled: true,
				injector: &testInjector{
					resp500: false,
				},
				injectPercent: 1.0,
				pathBlacklist: map[string]bool{
					"/donotinject": true,
				},
				pathWhitelist: map[string]bool{
					"/faultenabled": true,
				},
				randSeed: 100,
				rand:     rand.New(rand.NewSource(100)),
			},
			wantErr: nil,
		},
		// {
		// 	name: "invalid injector",
		// 	give: Options{
		// 		Injector:          nil,
		// 		PercentOfRequests: 1.0,
		// 	},
		// 	wantFault: nil,
		// 	wantErr:   ErrNilInjector,
		// },
		// {
		// 	name: "invalid percent",
		// 	give: Options{
		// 		Injector:          newTestInjector(false),
		// 		PercentOfRequests: 1.1,
		// 	},
		// 	wantFault: nil,
		// 	wantErr:   ErrInvalidPercent,
		// },
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f, err := NewFault(tt.giveInjector, tt.giveOptions...)

			assert.Equal(t, tt.wantErr, err)
			assert.Equal(t, tt.wantFault, f)
		})
	}
}

// TestFaultHandler tests Fault.Handler.
func TestFaultHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		give     *Fault
		wantCode int
		wantBody string
	}{
		{
			name:     "nil",
			give:     nil,
			wantCode: testHandlerCode,
			wantBody: testHandlerBody,
		},
		{
			name:     "empty",
			give:     &Fault{},
			wantCode: testHandlerCode,
			wantBody: testHandlerBody,
		},
		{
			name: "nil injector",
			give: &Fault{
				opt: Options{
					Enabled:           true,
					Injector:          nil,
					PercentOfRequests: 1.0,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			},
			wantCode: testHandlerCode,
			wantBody: testHandlerBody,
		},
		{
			name: "not enabled",
			give: &Fault{
				opt: Options{
					Enabled: false,
					Injector: &testInjector{
						resp500: false,
					},
					PercentOfRequests: 1.0,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			},
			wantCode: testHandlerCode,
			wantBody: testHandlerBody,
		},
		{
			name: "zero percent",
			give: &Fault{
				opt: Options{
					Enabled: true,
					Injector: &testInjector{
						resp500: false,
					},
					PercentOfRequests: 0.0,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			},
			wantCode: testHandlerCode,
			wantBody: testHandlerBody,
		},
		{
			name: "100 percent 500s",
			give: &Fault{
				opt: Options{
					Enabled: true,
					Injector: &testInjector{
						resp500: true,
					},
					PercentOfRequests: 1.0,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			},
			wantCode: http.StatusInternalServerError,
			wantBody: http.StatusText(http.StatusInternalServerError),
		},
		{
			name: "100 percent 500s with blacklist root",
			give: &Fault{
				opt: Options{
					Enabled: true,
					Injector: &testInjector{
						resp500: true,
					},
					PercentOfRequests: 1.0,
					PathBlacklist: []string{
						"/",
					},
				},
				pathBlacklist: map[string]bool{
					"/": true,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			},
			wantCode: testHandlerCode,
			wantBody: testHandlerBody,
		},
		{
			name: "100 percent 500s with whitelist root",
			give: &Fault{
				opt: Options{
					Enabled: true,
					Injector: &testInjector{
						resp500: true,
					},
					PercentOfRequests: 1.0,
					PathWhitelist: []string{
						"/",
					},
				},
				pathWhitelist: map[string]bool{
					"/": true,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			},
			wantCode: http.StatusInternalServerError,
			wantBody: http.StatusText(http.StatusInternalServerError),
		},
		{
			name: "100 percent 500s with whitelist other",
			give: &Fault{
				opt: Options{
					Enabled: true,
					Injector: &testInjector{
						resp500: true,
					},
					PercentOfRequests: 1.0,
					PathWhitelist: []string{
						"/onlyinject",
					},
				},
				pathWhitelist: map[string]bool{
					"/onlyinject": true,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			},
			wantCode: testHandlerCode,
			wantBody: testHandlerBody,
		},
		{
			name: "100 percent 500s with whitelist and blacklist root",
			give: &Fault{
				opt: Options{
					Enabled: true,
					Injector: &testInjector{
						resp500: true,
					},
					PercentOfRequests: 1.0,
					PathBlacklist: []string{
						"/",
					},
					PathWhitelist: []string{
						"/",
					},
				},
				pathBlacklist: map[string]bool{
					"/": true,
				},
				pathWhitelist: map[string]bool{
					"/": true,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			},
			wantCode: testHandlerCode,
			wantBody: testHandlerBody,
		},
		{
			name: "100 percent",
			give: &Fault{
				opt: Options{
					Enabled: true,
					Injector: &testInjector{
						resp500: false,
					},
					PercentOfRequests: 1.0,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			},
			wantCode: testHandlerCode,
			wantBody: testHandlerBody,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rr := testRequest(t, tt.give)

			assert.Equal(t, tt.wantCode, rr.Code)
			assert.Equal(t, tt.wantBody, strings.TrimSpace(rr.Body.String()))
		})
	}
}

// TestFaultPercentDo tests the internal Fault.percentDo.
func TestFaultPercentDo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		givePercent float32
		wantPercent float32
		wantRange   float32
	}{
		{-1.0, 0.0, 0.0},
		{},
		{0.0, 0.0, 0.0},
		{0.0001, 0.0001, 0.005},
		{0.3298, 0.3298, 0.005},
		{0.75, 0.75, 0.005},
		{1.0, 1.0, 0.0},
		{1.1, 0.0, 0.0},
		{10000.1, 0.0, 0.0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(fmt.Sprintf("%g", tt.givePercent), func(t *testing.T) {
			t.Parallel()

			f := &Fault{
				opt: Options{
					PercentOfRequests: tt.givePercent,
				},
				rand: rand.New(rand.NewSource(defaultRandSeed)),
			}

			var errorC, totalC float32
			for totalC <= 100000 {
				result := f.percentDo()
				if result {
					errorC++
				}
				totalC++
			}

			minP := tt.wantPercent - tt.wantRange
			per := errorC / totalC
			maxP := tt.wantPercent + tt.wantRange

			assert.GreaterOrEqual(t, per, minP)
			assert.LessOrEqual(t, per, maxP)
		})
	}
}
