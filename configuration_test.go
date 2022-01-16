package application

import (
	"testing"
	"time"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/portcullis/application/confighcl"
)

func TestHCL(t *testing.T) {
	tests := []struct {
		Name  string
		Input string
		Value interface{}
	}{
		{
			Name:  "basic",
			Input: `hello = "world"`,
			Value: &struct {
				Hello string `config:"hello,optional"`
			}{},
		},
		{
			Name:  "Duration",
			Input: `timeout = "5s"`,
			Value: &struct {
				Timeout time.Duration `config:"timeout,optional"`
			}{},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			file, diags := hclsyntax.ParseConfig([]byte(test.Input), "test.hcl", hcl.Pos{Line: 1, Column: 1})
			if diags.HasErrors() {
				t.Fatalf(diags.Error())
			}

			diags = confighcl.DecodeBody(file.Body, &hcl.EvalContext{}, test.Value)
			if diags.HasErrors() {
				t.Fatalf(diags.Error())
			}

			t.Logf("%+v", test.Value)
		})
	}
}
