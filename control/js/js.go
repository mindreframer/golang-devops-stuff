// This package implements vulcan controller and is based on
// Robert Krimen's Otto javascript magnificent interpreter.
package js

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/vulcan/client"
	. "github.com/mailgun/vulcan/command"
	"github.com/mailgun/vulcan/discovery"
	"github.com/mailgun/vulcan/netutils"
	"github.com/robertkrimen/otto"
	"net/http"
	"runtime/debug"
	"sync"
)

type JsController struct {
	// Discovery service interface, Zookeeper or Etcd can hide behind
	// the simple interface.
	DiscoveryService discovery.Service
	// Code getter is responsible for fetching the request from file,
	// hardcoded string or discovery service.
	CodeGetter CodeGetter
	// Client allows controlller to issue concurrent get requests
	// within the javascript handler.
	Client client.Client

	pool     *ContextPool
	lasthash string
}

type JsContext struct {
	otto        *otto.Otto
	handle      otto.Value
	handleError otto.Value
	pool        *ContextPool
}

// TODO(pquerna): Remove once Go 1.3 is out <https://code.google.com/p/go/issues/detail?id=4720>
type ContextPool struct {
	list []*JsContext
	ctrl *JsController
	mu   sync.Mutex
}

func NewContextPool(ctrl *JsController) *ContextPool {
	return &ContextPool{ctrl: ctrl, list: make([]*JsContext, 0, 64)}
}

func (p *ContextPool) New() (*JsContext, error) {
	return p.ctrl.getContext()
}

func (p *ContextPool) Get() (*JsContext, error) {
	p.mu.Lock()
	var x *JsContext
	var err error
	if n := len(p.list); n > 0 {
		x = p.list[n-1]
		p.list[n-1] = nil // Just to be safe
		p.list = p.list[:n-1]
	}
	p.mu.Unlock()

	if x == nil {
		x, err = p.New()
		if x != nil {
			x.pool = p
		}
	}

	return x, err
}

func (p *ContextPool) Put(jsc *JsContext) {
	if jsc == nil {
		return
	}
	p.mu.Lock()
	p.list = append(p.list, jsc)
	p.mu.Unlock()
}

func NewJsContext() *JsContext {
	return &JsContext{otto: otto.New()}
}

func (jsc *JsContext) Release() {
	if jsc.pool != nil {
		jsc.pool.Put(jsc)
	}
}

func (ctrl *JsController) getContext() (*JsContext, error) {
	code, err := ctrl.CodeGetter.GetCode()

	if err != nil {
		return nil, err
	}

	jsc := NewJsContext()
	ctrl.registerBuiltins(jsc.otto)

	_, err = jsc.otto.Run(code)
	if err != nil {
		return nil, err
	}

	handler, err := jsc.otto.Get("handle")
	if err != nil {
		return nil, err
	}

	jsc.handle = handler

	handler, err = jsc.otto.Get("handleError")
	if err != nil {
		return nil, err
	}

	jsc.handleError = handler

	return jsc, nil
}

const SKIP_CONTEXT_CACHE = false

func (ctrl *JsController) getContextFromCache() (*JsContext, error) {
	pool := ctrl.pool

	if SKIP_CONTEXT_CACHE {
		jsc, err := ctrl.getContext()
		return jsc, err
	}

	hash, err := ctrl.CodeGetter.GetHash()
	if err != nil {
		return nil, err
	}

	if hash != ctrl.lasthash || ctrl.pool == nil {
		pool = NewContextPool(ctrl)
		ctrl.pool = pool
		ctrl.lasthash = hash
	}

	return pool.Get()
}

func (ctrl *JsController) GetInstructions(req *http.Request) (interface{}, error) {
	var instr interface{}
	err := fmt.Errorf("Internal system error")
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf("Recovered: %v %s", r, debug.Stack())
			err = fmt.Errorf("Internal js error")
			instr = nil
		}
	}()

	jsc, err := ctrl.getContextFromCache()
	if err != nil {
		return nil, err
	}
	defer jsc.Release()

	jsRequest, err := requestToJs(req)
	if err != nil {
		return nil, err
	}
	jsObj, err := jsc.otto.ToValue(jsRequest)
	if err != nil {
		return nil, err
	}
	instr, err = ctrl.callHandler(jsc.handle, jsObj)
	if err != nil {
		return nil, err
	}
	return NewCommandFromObj(instr)
}

func (ctrl *JsController) ConvertError(req *http.Request, inError error) (response *netutils.HttpError, err error) {
	response = netutils.NewHttpError(http.StatusInternalServerError)
	err = fmt.Errorf("Internal error")
	defer func() {
		if r := recover(); r != nil {
			glog.Errorf("Recovered: %v %s", r, debug.Stack())
		}
	}()

	jsc, err := ctrl.getContextFromCache()
	if err != nil {
		return nil, err
	}
	defer jsc.Release()

	if jsc.handleError.IsUndefined() {
		glog.Infof("Missing error handler: %s", err)
		converted, err := errorFromJs(errorToJs(inError))
		if err != nil {
			glog.Errorf("Failed to convert error: %s", err)
			return nil, err
		}
		return converted, nil
	}
	obj := errorToJs(inError)
	jsObj, err := jsc.otto.ToValue(obj)
	if err != nil {
		glog.Errorf("Error: %s", err)
		return nil, err
	}
	jsRequest, err := requestToJs(req)
	if err != nil {
		return nil, err
	}
	jsRequestValue, err := jsc.otto.ToValue(jsRequest)
	if err != nil {
		return nil, err
	}
	out, err := ctrl.callHandler(jsc.handleError, jsRequestValue, jsObj)
	if err != nil {
		glog.Errorf("Error: %s", err)
		return nil, err
	}
	converted, err := errorFromJs(out)
	if err != nil {
		glog.Errorf("Failed to convert error: %s", err)
		return nil, err
	}
	return converted, nil
}

func (ctrl *JsController) callHandler(handler otto.Value, params ...interface{}) (interface{}, error) {
	if !handler.IsFunction() {
		return nil, fmt.Errorf("Result should be a function, got %v", handler)
	}
	out, err := handler.Call(handler, params...)
	if err != nil {
		glog.Errorf("Call resulted in failure %#v", err)
		return nil, err
	}

	obj, err := out.Export()
	if err != nil {
		glog.Errorf("Failed to extract response %#v", err)
		return nil, err
	}
	return obj, nil
}

func (ctrl *JsController) registerBuiltins(o *otto.Otto) {
	ctrl.addDiscoveryService(o)
	ctrl.addGetter(o)
	ctrl.addLoggers(o)
}

func (ctrl *JsController) addDiscoveryService(o *otto.Otto) {
	o.Set("discover", func(call otto.FunctionCall) otto.Value {
		if len(call.ArgumentList) == 0 {
			glog.Errorf("DISCOVER: Missing arguments")
			return otto.NullValue()
		}

		url, _ := call.Argument(0).ToString()
		upstreams, err := ctrl.DiscoveryService.Get(url)
		if err != nil {
			glog.Errorf("Failed to discover upstreams: %v", err)
			return otto.NullValue()
		}

		glog.Infof("Discovered upstreams: %v", upstreams)

		result, err := o.ToValue(upstreams)
		if err != nil {
			glog.Errorf("Failed to convert: %v", err)
			return otto.NullValue()
		}

		return result
	})
}

func (ctrl *JsController) addLoggers(o *otto.Otto) {
	o.Set("info", func(call otto.FunctionCall) otto.Value {
		return log("info", call)
	})
	o.Set("error", func(call otto.FunctionCall) otto.Value {
		return log("error", call)
	})

}

func (ctrl *JsController) addGetter(o *otto.Otto) {
	o.Set("get", func(call otto.FunctionCall) otto.Value {
		if len(call.ArgumentList) <= 0 {
			glog.Errorf("GET: Missing arguments")
			return newError(o, fmt.Errorf("GET: missing arguments"))
		}

		// Convert first argument, expect either string with url or list of strings
		upstreamsI, err := call.Argument(0).Export()
		if err != nil {
			glog.Errorf("GET: Failed to export first argument: %s", err)
			return newError(o, err)
		}
		upstreams, err := toStringArray(upstreamsI)
		if err != nil {
			glog.Errorf("GET: Failed to convert upstreams: %s", err)
			return newError(o, err)
		}

		// Second argument may be absent
		var query client.MultiDict
		if len(call.ArgumentList) > 1 {
			queryI, err := call.Argument(1).Export()
			if err != nil {
				glog.Errorf("GET: Failed to export first argument: %s", err)
				return newError(o, err)
			}
			dict, err := toMultiDict(queryI)
			if err != nil {
				glog.Errorf("GET: Failed: %s", err)
				return newError(o, err)
			}
			query = dict
		}

		// Third argument is optional username/password object
		var auth *netutils.BasicAuth
		if len(call.ArgumentList) > 2 {
			queryI, err := call.Argument(2).Export()
			if err != nil {
				glog.Errorf("GET: Failed: %s", err)
				return newError(o, err)
			}
			creds, err := toBasicAuth(queryI)
			if err != nil {
				glog.Errorf("GET: Failed: %s", err)
				return newError(o, err)
			}
			auth = creds
		}
		writer := NewResponseWriter()
		err = ctrl.Client.Get(writer, upstreams, query, auth)
		if err != nil {
			glog.Errorf("GET: Failed: %s", err)
			return newError(o, err)
		}
		reply := writer.ToReply()
		converted, err := o.ToValue(reply)
		if err != nil {
			glog.Errorf("GET: Failed: %s", err)
			return newError(o, err)
		}
		return converted
	})
}

func log(severity string, call otto.FunctionCall) otto.Value {
	var logger func(string, ...interface{})
	if severity == "info" {
		logger = glog.Infof
	} else if severity == "error" {
		logger = glog.Errorf
	} else {
		glog.Errorf("Unsupported severity: %s", severity)
		return otto.NullValue()
	}

	if len(call.ArgumentList) <= 0 {
		glog.Errorf("Missing arguments")
		return otto.NullValue()
	}
	formatI, err := call.Argument(0).Export()
	if err != nil {
		glog.Errorf("Fail: %s", err)
		return otto.NullValue()
	}
	formatString, err := toString(formatI)
	if err != nil {
		return otto.NullValue()
	}

	arguments := make([]interface{}, len(call.ArgumentList)-1)
	for i, val := range call.ArgumentList {
		if i == 0 {
			continue
		}
		obj, err := val.Export()
		if err != nil {
			glog.Errorf("Failed to convert argument: %v", err)
			return otto.NullValue()
		}
		arguments[i-1] = obj
	}
	logger(formatString, arguments...)
	return otto.NullValue()
}
