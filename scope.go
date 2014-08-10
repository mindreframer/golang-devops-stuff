package tachyon

import (
	"encoding/json"
	"fmt"
)

type Value interface {
	Read() interface{}
}

type AnyValue struct {
	v interface{}
}

func (a AnyValue) Read() interface{} {
	return a.v
}

type AnyMap struct {
	m map[interface{}]interface{}
}

func (a AnyMap) Read() interface{} {
	return a.m
}

func (a AnyMap) Get(key string) (Value, bool) {
	if v, ok := a.m[key]; ok {
		return Any(v), true
	}

	return nil, false
}

type StrMap struct {
	m map[string]interface{}
}

func (a StrMap) Get(key string) (Value, bool) {
	if v, ok := a.m[key]; ok {
		return Any(v), true
	}

	return nil, false
}

func (a StrMap) Read() interface{} {
	return a.m
}

func (a AnyValue) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.v)
}

func (a AnyMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.m)
}

func (a StrMap) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.m)
}

func Any(v interface{}) Value {
	switch sv := v.(type) {
	case AnyValue:
		return sv
	case map[interface{}]interface{}:
		return AnyMap{sv}
	case map[string]interface{}:
		return StrMap{sv}
	default:
		return AnyValue{v}
	}
}

func (a AnyValue) GetYAML() (string, interface{}) {
	return "", a.v
}

func (a AnyValue) SetYAML(tag string, v interface{}) bool {
	a.v = v
	return true
}

type Map interface {
	Get(key string) (Value, bool)
}

type Scope interface {
	Get(key string) (Value, bool)
	Set(key string, val interface{})
}

type ScopeGetter interface {
	Get(key string) (Value, bool)
}

func SV(v interface{}, ok bool) interface{} {
	if !ok {
		return nil
	}

	return v
}

type NestedScope struct {
	Scope Scope
	Vars  Vars
}

func NewNestedScope(parent Scope) *NestedScope {
	return &NestedScope{parent, make(Vars)}
}

func SpliceOverrides(cur Scope, override *NestedScope) *NestedScope {
	ns := NewNestedScope(cur)

	for k, v := range override.Vars {
		ns.Set(k, v)
	}

	return ns
}

func (n *NestedScope) Get(key string) (v Value, ok bool) {
	v, ok = n.Vars[key]
	if !ok && n.Scope != nil {
		v, ok = n.Scope.Get(key)
	}

	return
}

func (n *NestedScope) Set(key string, v interface{}) {
	n.Vars[key] = Any(v)
}

func (n *NestedScope) Empty() bool {
	return len(n.Vars) == 0
}

func (n *NestedScope) Flatten() Scope {
	if len(n.Vars) == 0 && n.Scope != nil {
		return n.Scope
	}

	return n
}

func (n *NestedScope) addMapVars(mv map[interface{}]interface{}) error {
	for k, v := range mv {
		if sk, ok := k.(string); ok {
			if sv, ok := v.(string); ok {
				var err error

				v, err = ExpandVars(n, sv)
				if err != nil {
					return err
				}
			}

			n.Set(sk, v)
		}
	}

	return nil
}

func (n *NestedScope) addVars(vars interface{}) (err error) {
	switch mv := vars.(type) {
	case map[interface{}]interface{}:
		err = n.addMapVars(mv)
	case []interface{}:
		for _, i := range mv {
			err = n.addVars(i)
			if err != nil {
				return
			}
		}
	}

	return
}

func ImportVarsFile(s Scope, path string) error {
	var fv map[string]string

	err := yamlFile(path, &fv)

	if err != nil {
		return err
	}

	for k, v := range fv {
		s.Set(k, inferString(v))
	}

	return nil
}

func DisplayScope(s Scope) {
	if ns, ok := s.(*NestedScope); ok {
		DisplayScope(ns.Scope)

		for k, v := range ns.Vars {
			fmt.Printf("%s: %v\n", k, v)
		}
	}
}
