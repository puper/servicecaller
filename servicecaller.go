package servicecaller

import (
	"context"
	"encoding/json"
	"errors"
	"go/token"
	"reflect"
	"strings"
)

func New() *ServiceCaller {
	return &ServiceCaller{
		reflectServiceMap: make(map[string]*service),
		serviceMap:        make(map[string]any),
	}
}

type ServiceCaller struct {
	reflectServiceMap map[string]*service
	serviceMap        map[string]any
}

func (me *ServiceCaller) Register(name string, rcvr any) {
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	s.name = name
	s.method = suitableMethods(s.typ)
	me.reflectServiceMap[name] = s
	me.serviceMap[name] = rcvr
}

func (me *ServiceCaller) Get(name string) any {
	return me.serviceMap[name]
}

func (me *ServiceCaller) Call(ctx context.Context, serviceMethod string, args json.RawMessage) (any, error) {
	dot := strings.LastIndex(serviceMethod, ".")
	if dot < 0 {
		return nil, errors.New("service/method request ill-formed: " + serviceMethod)
	}
	serviceName := serviceMethod[:dot]
	methodName := serviceMethod[dot+1:]
	s, ok := me.reflectServiceMap[serviceName]
	if !ok {
		return nil, errors.New("service not found")
	}
	mtype, ok := s.method[methodName]
	if !ok {
		return nil, errors.New("method not found")
	}
	var argv, replyv reflect.Value
	argIsValue := false
	if mtype.ArgType.Kind() == reflect.Pointer {
		argv = reflect.New(mtype.ArgType.Elem())
	} else {
		argv = reflect.New(mtype.ArgType)
		argIsValue = true
	}
	if err := json.Unmarshal(args, argv.Interface()); err != nil {
		return nil, err
	}
	if argIsValue {
		argv = argv.Elem()
	}

	replyv = reflect.New(mtype.ReplyType.Elem())

	switch mtype.ReplyType.Elem().Kind() {
	case reflect.Map:
		replyv.Elem().Set(reflect.MakeMap(mtype.ReplyType.Elem()))
	case reflect.Slice:
		replyv.Elem().Set(reflect.MakeSlice(mtype.ReplyType.Elem(), 0, 0))
	}
	function := mtype.method.Func
	returnValues := function.Call([]reflect.Value{s.rcvr, reflect.ValueOf(ctx), argv, replyv})
	errInter := returnValues[0].Interface()
	if errInter == nil {
		return reflect.Indirect(replyv).Interface(), nil
	}
	return nil, errInter.(error)
}

func suitableMethods(typ reflect.Type) map[string]*methodType {
	methods := make(map[string]*methodType)
	for m := 0; m < typ.NumMethod(); m++ {
		method := typ.Method(m)
		mtype := method.Type
		mname := method.Name
		if !method.IsExported() {
			continue
		}
		if mtype.NumIn() != 4 {
			continue
		}
		ctxType := mtype.In(1)
		if ctxType != typeOfContext {
			continue
		}
		argType := mtype.In(2)
		if !isExportedOrBuiltinType(argType) {
			continue
		}
		replyType := mtype.In(3)
		if replyType.Kind() != reflect.Pointer {
			continue
		}
		if !isExportedOrBuiltinType(replyType) {
			continue
		}
		if mtype.NumOut() != 1 {
			continue
		}
		if returnType := mtype.Out(0); returnType != typeOfError {
			continue
		}
		methods[mname] = &methodType{method: method, ArgType: argType, ReplyType: replyType}
	}
	return methods
}

type service struct {
	name   string
	rcvr   reflect.Value
	typ    reflect.Type
	method map[string]*methodType
}

var typeOfError = reflect.TypeFor[error]()

var typeOfContext = reflect.TypeFor[context.Context]()

type methodType struct {
	method    reflect.Method
	ArgType   reflect.Type
	ReplyType reflect.Type
}

func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return token.IsExported(t.Name()) || t.PkgPath() == ""
}
