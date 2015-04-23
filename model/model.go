package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"reflect"
	"strings"
)

type Model interface {
	GetId() string
	RootURL() string
}

// Create expects a pointer some concrete type which implements Model (e.g., *Todo).
// It will send a POST request to the RESTful server. It expects a JSON containing the
// created object from the server if the request was successful, and will set the fields of
// model with the data in the response object. It will use the RootURL() method of
// the model to determine which url to send the POST request to.
func Create(model Model) error {
	fullURL := model.RootURL()
	encodedModelData, err := encodeModelFields(model)
	if err != nil {
		return err
	}
	return sendRequestAndUnmarshal("POST", fullURL, encodedModelData, model)
}

// Read will send a GET request to a RESTful server to get the model by the given id,
// then it will scan the results into model. It expects a json object which contains all
// the fields for the requested model. Read will use the RootURL() method of the model to
// figure out which url to send the GET request to. Typically the full url will look something
// like "http://hostname.com/todos/123"
func Read(id string, model Model) error {
	fullURL := model.RootURL() + "/" + id
	return sendRequestAndUnmarshal("GET", fullURL, "", model)
}

// ReadAll expects a pointer to a slice of poitners to some concrete type
// which implements Model (e.g., *[]*Todo). ReadAll will send a GET request to
// a RESTful server and scan the results into models. It expects a json array
// of json objects from the server, where each object represents a single Model
// of some concrete type. It will use the RootURL() method of the models to
// figure out which url to send the GET request to.
func ReadAll(models interface{}) error {
	rootURL, err := getURLFromModels(models)
	if err != nil {
		return err
	}
	return sendRequestAndUnmarshal("GET", rootURL, "", models)
}

// Update expects a pointer some concrete type which implements Model (e.g., *Todo), with a model.Id
// that matches a stored object on the server. It will send a PUT request to the RESTful server.
// It expects a JSON containing the updated object from the server if the request was successful,
// and will set the fields of model with the data in the response object.
// It will use the RootURL() method of the model to determine which url to send the PUT request to.
func Update(model Model) error {
	fullURL := model.RootURL() + "/" + model.GetId()
	encodedModelData, err := encodeModelFields(model)
	if err != nil {
		return err
	}
	return sendRequestAndUnmarshal("PUT", fullURL, encodedModelData, model)
}

// Delete expects a pointer some concrete type which implements Model (e.g., *Todo).
// It will send a DELETE request to a RESTful server. It expects an empty json
// object from the server if the request was successful, and will not attempt to do anything
// with the response. It will use the RootURL() and GetId() methods of the model to determine
// which url to send the DELETE request to. Typically, the full url will look something
// like "http://hostname.com/todos/123"
func Delete(model Model) error {
	fullURL := model.RootURL() + "/" + model.GetId()
	req, err := http.NewRequest("DELETE", fullURL, nil)
	if err != nil {
		return fmt.Errorf("Something went wrong building DELETE request to %s: %s", fullURL, err.Error())
	}
	if _, err := http.DefaultClient.Do(req); err != nil {
		return fmt.Errorf("Something went wrong with DELETE request to %s: %s", fullURL, err.Error())
	}
	return nil
}

// getURLFromModels returns the url that should be used for the type that corresponds
// to models. It does this by instantiating a new model of the correct type and then
// calling RootURL on it. models should be a pointer to a slice of models.
func getURLFromModels(models interface{}) (string, error) {
	// Check the type of models
	typ := reflect.TypeOf(models)
	switch {
	// Make sure its a pointer
	case typ.Kind() != reflect.Ptr:
		return "", fmt.Errorf("models must be a pointer to a slice of models. %T is not a pointer.", models)
	// Make sure its a pointer to a slice
	case typ.Elem().Kind() != reflect.Slice:
		return "", fmt.Errorf("models must be a pointer to a slice of models. %T is not a pointer to a slice", models)
	// Make sure the type of the elements of the slice implement Model
	case !typ.Elem().Elem().Implements(reflect.TypeOf([]Model{}).Elem()):
		return "", fmt.Errorf("models must be a pointer to a slice of models. The elem type %T does not implement model", typ.Elem().Elem())
	}
	// modelType is the type of the elements of models
	modelType := typ.Elem().Elem()
	// Ultimately, we need to be able to instantiate a new object of a type that
	// implements Model so that we can call RootURL on it. The trouble is that
	// reflect.New only works for things that are not pointers, and the type of
	// the elements of models could be pointers. To solve for this, we are going
	// to get the Elem of modelType if it is a pointer and keep track of the number
	// of times we get the Elem. So if modelType is *Todo, we'll call Elem once to
	// get the type Todo.
	numDeref := 0
	for modelType.Kind() == reflect.Ptr {
		modelType = modelType.Elem()
		numDeref += 1
	}
	// Now that we have the underlying type that is not a pointer, we can instantiate
	// a new object with reflect.New.
	newModelVal := reflect.New(modelType).Elem()
	// Now we need to iteratively get the address of the object we created exactly
	// numDeref times to get to a type that implements Model. Note that Addr is the
	// inverse of Elem.
	for i := 0; i < numDeref; i++ {
		newModelVal = newModelVal.Addr()
	}
	// Now we can use a type assertion to convert the object we instantiated to a Model
	newModel := newModelVal.Interface().(Model)
	// Finally, once we have a Model we can get what we wanted by calling RootURL
	return newModel.RootURL(), nil
}

// sendRequestAndUnmarshal constructs a request with the given method, url, and
// data. If data is an empty string, it will construct a request without any
// data in the body. If data is a non-empty string, it will send it as the body
// of the request and set the Content-Type header to
// application/x-www-form-urlencoded. Then sendRequestAndUnmarshal sends the
// request using http.DefaultClient and marshals the response into v using the json
// package.
// TODO: do something if the response status code is non-200.
func sendRequestAndUnmarshal(method string, url string, data string, v interface{}) error {
	// Build the request
	req, err := http.NewRequest(method, url, strings.NewReader(data))
	if err != nil {
		return fmt.Errorf("Something went wrong building %s request to %s: %s", method, url, err.Error())
	}
	// Set the Content-Type header only if data was provided
	if data != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	// Send the request using the default client
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("Something went wrong with %s request to %s: %s", req.Method, req.URL.String(), err.Error())
	}
	// Unmarshal the response into v
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return fmt.Errorf("Couldn't read response to %s: %s", res.Request.URL.String(), err.Error())
	}
	return json.Unmarshal(body, v)
}

// encodeModelFields returns the fields of model represented as a url-encoded string.
// Suitable for POST requests with a content type of application/x-www-form-urlencoded.
// It returns an error if model is a nil pointer or if it is not a struct or a pointer
// to a struct. Any fields that are nil will not be added to the url-encoded string.
func encodeModelFields(model Model) (string, error) {
	modelVal := reflect.ValueOf(model)
	// dereference the pointer until we reach the underlying struct value.
	for modelVal.Kind() == reflect.Ptr {
		if modelVal.IsNil() {
			return "", errors.New("Error encoding model as url-encoded data: model was a nil pointer.")
		}
		modelVal = modelVal.Elem()
	}
	// Make sure the type of model after dereferencing is a struct.
	if modelVal.Kind() != reflect.Struct {
		return "", fmt.Errorf("Error encoding model as url-encoded data: model must be a struct or a pointer to a struct.")
	}
	encodedFields := []string{}
	for i := 0; i < modelVal.Type().NumField(); i++ {
		field := modelVal.Type().Field(i)
		fieldValue := modelVal.FieldByName(field.Name)
		encodedField, err := encodeField(field, fieldValue)
		if err != nil {
			if _, ok := err.(nilFieldError); ok {
				// If there was a nil field, continue without adding the field
				// to the encoded data.
				continue
			}
			// We should return any other kind of error
			return "", err
		}
		encodedFields = append(encodedFields, field.Name+"="+encodedField)
	}
	return strings.Join(encodedFields, "&"), nil
}

type nilFieldError struct{}

func (nilFieldError) Error() string {
	return "field was nil"
}

// encodeField converts a field with the given value to a string. It returns an error
// if field has a type which is unsupported. It returns a special error (nilFieldError)
// if a field has a value of nil. The supported types are int and its variants (int64,
// int32, etc.), uint and its variants (uint64, uint32, etc.), bool, string, and []byte.
func encodeField(field reflect.StructField, value reflect.Value) (string, error) {
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			// Skip nil fields
			return "", nilFieldError{}
		}
		value = value.Elem()
	}
	switch v := value.Interface().(type) {
	case int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8, bool:
		return fmt.Sprint(v), nil
	case string:
		return url.QueryEscape(v), nil
	case []byte:
		return url.QueryEscape(string(v)), nil
	default:
		return "", fmt.Errorf("Error encoding model as url-encoded data: Don't know how to convert %v of type %T to a string.", v, v)
	}
}
