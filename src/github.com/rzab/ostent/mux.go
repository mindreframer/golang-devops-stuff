package ostent
import (
	"net/http"

	"github.com/rcrowley/go-tigertonic"
)

type ServeMux interface {
	Handle(string, string, http.Handler)
}

type TrieServeMux struct {
	*tigertonic.TrieServeMux
	constructor func(http.Handler) http.Handler
}

func NewMux(constructor func(http.Handler) http.Handler) *TrieServeMux {
	return &TrieServeMux{
		TrieServeMux: tigertonic.NewTrieServeMux(),
		constructor:  constructor,
	}
}

// catch tigertonic error handlers, override
func (mux *TrieServeMux) handlerFunc(handler http.Handler) http.HandlerFunc {
	NA := tigertonic.MethodNotAllowedHandler{}
	NF := tigertonic.NotFoundHandler{}
	if handler == NF {
		return http.NotFound
	}
	if handler == NA {
		return func(w http.ResponseWriter, _ *http.Request) {
			http.Error(w, statusLine(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		}
	}
	return nil
}

func (mux *TrieServeMux) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	handler, _ := mux.TrieServeMux.Handler(r)
	if h := mux.handlerFunc(handler); h != nil {
		handler = mux.constructor(h)
	}
	handler.ServeHTTP(w, r)
}

func (mux *TrieServeMux) Handle(method, pattern string, handler http.Handler) {
	mux.TrieServeMux.Handle(method, pattern, handler)
}
