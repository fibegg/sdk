package fibe

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

var (
	contextType = reflect.TypeOf((*context.Context)(nil)).Elem()
	readerType  = reflect.TypeOf((*io.Reader)(nil)).Elem()
)

// TestServiceContractSmoke exercises every exported service method. The
// operation-specific golden tests assert exact wire details; this test catches
// uninitialized services, wrapper panics, and methods that cannot accept their
// declared argument types as the public surface grows.
func TestServiceContractSmoke(t *testing.T) {
	t.Parallel()

	fixture, err := os.CreateTemp(t.TempDir(), "fibe-contract-*.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixture.WriteString("fixture"); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Close(); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Disposition", `attachment; filename="fixture.txt"`)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"data":       []any{},
			"meta":       map[string]any{"page": 1, "per_page": 25, "total": 0},
			"status":     "success",
			"request_id": "contract-smoke",
			"result":     map[string]any{},
			"events":     []string{},
		})
	}))
	defer server.Close()

	client := NewClient(
		WithBaseURL(server.URL),
		WithDisableAutoConfig(),
		WithMaxRetries(0),
		WithTimeout(100*time.Millisecond),
	)
	clientValue := reflect.ValueOf(client).Elem()

	var called int
	for index := 0; index < clientValue.NumField(); index++ {
		service := clientValue.Field(index)
		field := clientValue.Type().Field(index)
		if !field.IsExported() || service.Kind() != reflect.Pointer || !strings.HasSuffix(service.Type().Elem().Name(), "Service") {
			continue
		}
		if service.IsNil() {
			t.Fatalf("service %s is nil", field.Name)
		}

		for methodIndex := 0; methodIndex < service.NumMethod(); methodIndex++ {
			method := service.Type().Method(methodIndex)
			t.Run(field.Name+"/"+method.Name, func(t *testing.T) {
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
				defer cancel()

				call := service.Method(methodIndex)
				arguments := make([]reflect.Value, call.Type().NumIn())
				for argumentIndex := range arguments {
					arguments[argumentIndex] = contractFixtureValue(call.Type().In(argumentIndex), ctx, fixture.Name(), 0)
				}

				defer func() {
					if recovered := recover(); recovered != nil {
						t.Fatalf("public service method panicked: %v", recovered)
					}
				}()
				results := call.Call(arguments)
				for _, result := range results {
					if result.IsValid() && result.CanInterface() && !isNilReflectValue(result) {
						if closer, ok := result.Interface().(io.Closer); ok {
							_ = closer.Close()
						}
					}
				}
				called++
			})
		}
	}

	if called != 297 {
		t.Fatalf("called %d exported service methods; update the contract count and REST manifest together", called)
	}
}

func contractFixtureValue(typ reflect.Type, ctx context.Context, filePath string, depth int) reflect.Value {
	if typ == contextType {
		return reflect.ValueOf(ctx)
	}
	if typ.Implements(readerType) {
		return reflect.ValueOf(strings.NewReader("fixture")).Convert(typ)
	}
	if depth > 4 {
		return reflect.Zero(typ)
	}

	if typ.Kind() == reflect.Pointer {
		if special := contractSpecialPointer(typ, filePath); special.IsValid() {
			return special
		}
		value := reflect.New(typ.Elem())
		value.Elem().Set(contractFixtureValue(typ.Elem(), ctx, filePath, depth+1))
		return value
	}

	switch typ.Kind() {
	case reflect.Interface:
		value := reflect.ValueOf(map[string]any{"fixture": true})
		if value.Type().AssignableTo(typ) {
			return value
		}
		return reflect.Zero(typ)
	case reflect.String:
		return reflect.ValueOf("fixture").Convert(typ)
	case reflect.Bool:
		return reflect.ValueOf(true).Convert(typ)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value := reflect.New(typ).Elem()
		value.SetInt(1)
		return value
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value := reflect.New(typ).Elem()
		value.SetUint(1)
		return value
	case reflect.Float32, reflect.Float64:
		value := reflect.New(typ).Elem()
		value.SetFloat(1)
		return value
	case reflect.Slice:
		if typ.Elem().Kind() == reflect.Uint8 {
			return reflect.ValueOf([]byte("fixture")).Convert(typ)
		}
		value := reflect.MakeSlice(typ, 1, 1)
		value.Index(0).Set(contractFixtureValue(typ.Elem(), ctx, filePath, depth+1))
		return value
	case reflect.Map:
		value := reflect.MakeMap(typ)
		key := contractFixtureValue(typ.Key(), ctx, filePath, depth+1)
		element := contractFixtureValue(typ.Elem(), ctx, filePath, depth+1)
		value.SetMapIndex(key, element)
		return value
	case reflect.Struct:
		if typ == reflect.TypeOf(time.Time{}) {
			return reflect.ValueOf(time.Unix(1, 0).UTC())
		}
		value := reflect.New(typ).Elem()
		for index := 0; index < value.NumField(); index++ {
			if value.Field(index).CanSet() {
				value.Field(index).Set(contractFixtureValue(value.Field(index).Type(), ctx, filePath, depth+1))
			}
		}
		return value
	case reflect.Func:
		return reflect.MakeFunc(typ, func([]reflect.Value) []reflect.Value {
			results := make([]reflect.Value, typ.NumOut())
			for index := range results {
				results[index] = reflect.Zero(typ.Out(index))
			}
			return results
		})
	default:
		return reflect.Zero(typ)
	}
}

func contractSpecialPointer(typ reflect.Type, filePath string) reflect.Value {
	switch typ.Elem().Name() {
	case "AgentCreateParams":
		return reflect.ValueOf(&AgentCreateParams{Name: "fixture", Provider: ProviderOpenCode})
	case "AgentPokeCreateParams":
		return reflect.ValueOf(&AgentPokeCreateParams{Schedule: "0 * * * *", Prompt: "fixture"})
	case "GreenfieldCreateParams":
		return reflect.ValueOf(&GreenfieldCreateParams{Name: "fixture"})
	case "LaunchParams":
		return reflect.ValueOf(&LaunchParams{Name: "fixture", ComposeYAML: "services: {}"})
	case "MarqueeCreateParams":
		return reflect.ValueOf(&MarqueeCreateParams{Name: "fixture", Host: "127.0.0.1", Port: 22, User: "fixture", SSHPrivateKey: "fixture"})
	case "PlaygroundActionParams":
		return reflect.ValueOf(&PlaygroundActionParams{ActionType: PlaygroundActionStop})
	case "PlaygroundCreateParams":
		return reflect.ValueOf(&PlaygroundCreateParams{Name: "fixture", PlayspecID: 1})
	case "PlayspecCreateParams":
		return reflect.ValueOf(&PlayspecCreateParams{Name: "fixture", BaseComposeYAML: "services: {}"})
	case "SecretCreateParams":
		return reflect.ValueOf(&SecretCreateParams{Key: "FIXTURE", Value: "fixture"})
	case "UploadImageParams":
		value := reflect.New(typ.Elem())
		field := value.Elem().FieldByName("FilePath")
		if field.IsValid() && field.CanSet() {
			field.SetString(filePath)
		}
		return value
	case "WebhookEndpointCreateParams":
		return reflect.ValueOf(&WebhookEndpointCreateParams{URL: "https://example.com/hook", Events: []string{"fixture"}})
	default:
		return reflect.Value{}
	}
}

func isNilReflectValue(value reflect.Value) bool {
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
