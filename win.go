package win

import (
	"encoding/json"
	"errors"
	"fmt"
)

var jsonNull = json.RawMessage("null")

type (
	HandlerFunc    func(ctx Context)
	MiddlewareFunc func(h HandlerFunc) HandlerFunc
)

type Request struct {
	Method  string                 `json:"method"`
	Params  *json.RawMessage       `json:"params,omitempty"`
	ID      int64                  `json:"id"`
	Headers map[string]interface{} `json:"headers"`
}

func (r Request) MarshalJSON() ([]byte, error) {
	r2 := struct {
		Method string           `json:"method"`
		Params *json.RawMessage `json:"params,omitempty"`
		ID     int64            `json:"id"`
	}{
		Method: r.Method,
		Params: r.Params,
		ID:     r.ID,
	}
	return json.Marshal(r2)
}

func (r *Request) UnmarshalJSON(data []byte) error {
	var r2 struct {
		Method string           `json:"method"`
		Params *json.RawMessage `json:"params,omitempty"`
		ID     int64            `json:"id"`
	}

	r2.Params = &json.RawMessage{}

	if err := json.Unmarshal(data, &r2); err != nil {
		return err
	}
	r.Method = r2.Method
	if r2.Params == nil {
		r.Params = &jsonNull
	} else if len(*r2.Params) == 0 {
		r.Params = nil
	} else {
		r.Params = r2.Params
	}
	r.ID = r2.ID
	return nil
}

func (r *Request) SetParams(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Params = (*json.RawMessage)(&b)
	return nil
}

type Response struct {
	Method  string                 `json:"method"`
	ID      int64                  `json:"id"`
	Result  *json.RawMessage       `json:"result,omitempty"`
	Error   *Error                 `json:"error,omitempty"`
	Headers map[string]interface{} `json:"headers"`
}

func (r Response) MarshalJSON() ([]byte, error) {
	if (r.Result == nil || len(*r.Result) == 0) && r.Error == nil {
		return nil, errors.New("can't marshal *win.Response (must have result or error)")
	}
	type tmpType Response
	b, err := json.Marshal(tmpType(r))
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (r *Response) UnmarshalJSON(data []byte) error {
	type tmpType Response

	*r = Response{Result: &json.RawMessage{}}

	if err := json.Unmarshal(data, (*tmpType)(r)); err != nil {
		return err
	}
	if r.Result == nil {
		r.Result = &jsonNull
	} else if len(*r.Result) == 0 {
		r.Result = nil
	}
	return nil
}

func (r *Response) SetResult(v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	r.Result = (*json.RawMessage)(&b)
	return nil
}

type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"msg"`
	Data    interface{} `json:"data"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("win: code %v message: %s", e.Code, e.Message)
}
