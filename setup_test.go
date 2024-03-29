package health

import (
	"testing"

	"github.com/coredns/caddy"
)

func TestSetupHealth(t *testing.T) {
	tests := []struct {
		input     string
		shouldErr bool
	}{
		{`health`, false},
		{`health localhost:1234`, false},
		{`health localhost:1234 {
			lameduck 4s
			lookup foo
}`, false},
		{`health localhost:1234 {
			lameduck 4s
			lookup foo bar baz
}`, false},
		{`health bla:a`, false},

		{`health bla`, true},
		{`health bla bla`, true},
		{`health localhost:1234 {
			lameduck a
			lookup
}`, true},
		{`health localhost:1234 {
			lamedudk 4
} `, true},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.input)
		_, _, _, err := parse(c)

		if test.shouldErr && err == nil {
			t.Errorf("Test %d: Expected error but found none for input %s", i, test.input)
		}

		if err != nil {
			if !test.shouldErr {
				t.Errorf("Test %d: Expected no error but found one for input %s. Error was: %v", i, test.input, err)
			}
		}
	}
}
