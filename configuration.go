package application

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/json"
	"github.com/portcullis/application/confighcl"
	"github.com/zclconf/go-cty/cty"
	"github.com/zclconf/go-cty/cty/function"
)

var (
	// ErrInvalidObjectPath is returned when a key cannot be converted into
	// a nested object path like "foo...bar", ".foo", or "foo."
	ErrInvalidObjectPath = errors.New("invalid object path")
)

// Configuration for the application based on github.com/hashicorp/hcl/v2
type Configuration struct{}

// DecodeFile will open and decode the provided file, returning an error when parsing fails
func (c Configuration) DecodeFile(ctx context.Context, filename string) hcl.Diagnostics {
	src, err := ioutil.ReadFile(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return hcl.Diagnostics{
				{
					Severity: hcl.DiagError,
					Summary:  "Configuration file not found",
					Detail:   fmt.Sprintf("The configuration file %s does not exist.", filename),
				},
			}
		}

		return hcl.Diagnostics{
			{
				Severity: hcl.DiagError,
				Summary:  "Failed to read configuration",
				Detail:   fmt.Sprintf("Can't read %s: %s.", filename, err),
			},
		}
	}

	return c.Decode(ctx, filename, src)
}

func (c Configuration) Decode(ctx context.Context, filename string, src []byte) hcl.Diagnostics {
	var file *hcl.File
	var diags hcl.Diagnostics

	switch suffix := strings.ToLower(filepath.Ext(filename)); suffix {
	case ".hcl":
		file, diags = hclsyntax.ParseConfig(src, filename, hcl.Pos{Line: 1, Column: 1})
	case ".json":
		file, diags = json.Parse(src, filename)
	default:
		diags = diags.Append(&hcl.Diagnostic{
			Severity: hcl.DiagError,
			Summary:  "Unsupported file format",
			Detail:   fmt.Sprintf("Cannot read from %s: unrecognized file format suffix %q.", filename, suffix),
		})
		return diags
	}
	if diags.HasErrors() {
		return diags
	}

	evalContext := c.EvalContext(ctx)

	target := struct {
		Configuration hcl.Body `config:",remain"`
	}{}

	diags = confighcl.DecodeBody(file.Body, evalContext, &target)
	if diags.HasErrors() {
		return diags
	}

	app := FromContext(ctx)
	if app != nil {

		// loop through the modules, see if configurable, then apply the configs
		app.Controller.Range(func(name string, m Module) bool {
			cfgr, ok := m.(Configurable)
			if !ok {
				return true
			}

			v, err := cfgr.Config()
			if err != nil {
				diags = diags.Append(&hcl.Diagnostic{
					Severity: hcl.DiagError,
					Summary:  "Failed to retrieve module config",
					Detail:   fmt.Sprintf("Cannot read config from module %s: %v.", name, err),
				})
				return false
			}

			target.Configuration, diags = confighcl.DecodeLeftoverBody(target.Configuration, evalContext, v)

			if diags.HasErrors() {
				return false
			}

			if notifier, ok := m.(ConfigurableNotify); ok {
				if err := notifier.ConfigSet(v); err != nil {
					diags = diags.Append(&hcl.Diagnostic{
						Severity: hcl.DiagError,
						Summary:  "Failed to notify module config",
						Detail:   fmt.Sprintf("Module %s returned an error on ConfigSet: %v.", name, err),
					})
					return false
				}
			}

			return true
		})

		if diags.HasErrors() {
			return diags
		}
	}

	return confighcl.DecodeBody(target.Configuration, evalContext, new(struct{}))
}

// EvalContext returns the hcl.EvalContext for loading hcl files
func (Configuration) EvalContext(ctx context.Context) *hcl.EvalContext {
	var result hcl.EvalContext

	// functions
	result.Functions = map[string]function.Function{}

	// variables
	allMap := map[string]interface{}{}

	if app := FromContext(ctx); app != nil {
		_ = addNestedKey(allMap, "application.name", app.Name)
		_ = addNestedKey(allMap, "application.version", app.Version)
	}

	var err error
	// if we put in something bad, panic
	if result.Variables, err = ctyify(allMap); err != nil {
		panic(err)
	}

	return &result
}

// addNestedKey expands keys into their nested form:
//
//	k="foo.bar", v="quux" -> {"foo": {"bar": "quux"}}
//
// Existing keys are overwritten. Map values take precedence over primitives.
//
// If the key has dots but cannot be converted to a valid nested data structure
// (eg "foo...bar", "foo.", or non-object value exists for key), an error is
// returned.
func addNestedKey(dst map[string]interface{}, k, v string) error {
	// createdParent and Key capture the parent object of the first created
	// object and the first created object's key respectively. The cleanup
	// func deletes them to prevent side-effects when returning errors.
	var createdParent map[string]interface{}
	var createdKey string
	cleanup := func() {
		if createdParent != nil {
			delete(createdParent, createdKey)
		}
	}

	segments := strings.Split(k, ".")
	for _, newKey := range segments[:len(segments)-1] {
		if newKey == "" {
			// String either begins with a dot (.foo) or has at
			// least two consecutive dots (foo..bar); either way
			// it's an invalid object path.
			cleanup()
			return ErrInvalidObjectPath
		}

		var target map[string]interface{}
		if existingI, ok := dst[newKey]; ok {
			if existing, ok := existingI.(map[string]interface{}); ok {
				// Target already exists
				target = existing
			} else {
				// Existing value is not a map. Maps should
				// take precedence over primitive values (eg
				// overwrite attr.driver.qemu = "1" with
				// attr.driver.qemu.version = "...")
				target = make(map[string]interface{})
				dst[newKey] = target
			}
		} else {
			// Does not exist, create
			target = make(map[string]interface{})
			dst[newKey] = target

			// If this is the first created key, capture it for
			// cleanup if there is an error later.
			if createdParent == nil {
				createdParent = dst
				createdKey = newKey
			}
		}

		// Descend into new m
		dst = target
	}

	// See if the final segment is a valid key
	newKey := segments[len(segments)-1]
	if newKey == "" {
		// String ends in a dot
		cleanup()
		return ErrInvalidObjectPath
	}

	if existingI, ok := dst[newKey]; ok {
		if _, ok := existingI.(map[string]interface{}); ok {
			// Existing value is a map which takes precedence over
			// a primitive value. Drop primitive.
			return nil
		}
	}
	dst[newKey] = v
	return nil
}

// ctyify converts nested map[string]interfaces to a map[string]cty.Value. An
// error is returned if an unsupported type is encountered.
//
// Currently only strings, cty.Values, and nested maps are supported.
func ctyify(src map[string]interface{}) (map[string]cty.Value, error) {
	dst := make(map[string]cty.Value, len(src))

	for k, vI := range src {
		switch v := vI.(type) {
		case string:
			dst[k] = cty.StringVal(v)

		case cty.Value:
			dst[k] = v

		case map[string]interface{}:
			o, err := ctyify(v)
			if err != nil {
				return nil, err
			}
			dst[k] = cty.ObjectVal(o)

		default:
			return nil, fmt.Errorf("key %q has invalid type %T", k, v)
		}
	}

	return dst, nil
}
