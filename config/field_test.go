package config_test

import (
	"testing"

	"github.com/drykit-go/testx"

	"github.com/benchttp/runner/config"
)

func TestIsField(t *testing.T) {
	testx.Table(config.IsField).Cases([]testx.Case{
		{In: config.FieldMethod, Exp: true},
		{In: config.FieldURL, Exp: true},
		{In: config.FieldTimeout, Exp: true},
		{In: config.FieldConcurrency, Exp: true},
		{In: config.FieldRequests, Exp: true},
		{In: config.FieldGlobalTimeout, Exp: true},
		{In: "notafield", Exp: false},
	}).Run(t)
}
