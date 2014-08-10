package lisp

var scope *Scope

func init() {
	scope = NewScope()
	scope.AddEnv()
}

type ScopedVars interface {
	Get(key string) (Value, bool)
	Set(key string, val Value) Value
	Create(key string, val Value) Value
}

type Env map[string]Value

type Scope struct {
	parent *Scope
	envs   []*Env
}

func NewScope() *Scope {
	scope := &Scope{}
	scope.envs = make([]*Env, 0)
	return scope
}

func NewNestedScope(parent *Scope) *Scope {
	scope := &Scope{}
	scope.parent = parent
	scope.envs = make([]*Env, 0)
	return scope
}

func (s *Scope) Dup() *Scope {
	scope := &Scope{}
	scope.envs = make([]*Env, len(s.envs))
	copy(scope.envs, s.envs)
	return scope
}

func (s *Scope) Env() *Env {
	if len(s.envs) > 0 {
		return s.envs[len(s.envs)-1]
	}
	return nil
}

func (s *Scope) AddEnv() *Env {
	env := make(Env)
	s.envs = append(s.envs, &env)
	return &env
}

func (s *Scope) DropEnv() *Env {
	s.envs[len(s.envs)-1] = nil
	s.envs = s.envs[:len(s.envs)-1]
	return s.Env()
}

func (s *Scope) Create(key string, value Value) Value {
	env := *s.Env()
	env[key] = value
	return value
}

func (s *Scope) Set(key string, value Value) Value {
	t := s

	for t != nil {
		for i := len(s.envs) - 1; i >= 0; i-- {
			env := *s.envs[i]
			if _, ok := env[key]; ok {
				env[key] = value
				return value
			}
		}

		t = t.parent
	}

	return s.Create(key, value)
}

func (s *Scope) Get(key string) (val Value, ok bool) {
	t := s

	for t != nil {
		for i := len(s.envs) - 1; i >= 0; i-- {
			env := *s.envs[i]
			if val, ok = env[key]; ok {
				return
			}
		}

		t = t.parent
	}

	return
}
