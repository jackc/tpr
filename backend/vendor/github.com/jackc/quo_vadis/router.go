// Quo Vadis is a really simple and fast HTTP router
package quo_vadis

import (
	"io"
	"net/http"
	"strings"
)

type endpoint struct {
	handler    http.Handler
	parameters []string
}

type node struct {
	methodEndpoints map[string]*endpoint
	staticBranches  map[string]*node
	parameterBranch *node
}

type Router struct {
	root                    *node
	NotFoundHandler         http.Handler // Handler to call if no path matches request
	MethodNotAllowedHandler http.Handler // Handler to call if the path does not respond to the request method
}

// ServeHTTP makes Router implement standard http.Handler
//
// ServeHTTP will rewrite the req.URL.RawQuery to include any path arguments so
// they can be treated as traditional query parameters.
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	segments := segmentizePath(req.URL.Path)
	if node, arguments, ok := r.root.findNode(segments, []string{}); ok {
		if endpoint, present := node.methodEndpoints[req.Method]; present {
			addRouteArgumentsToRequest(endpoint.parameters, arguments, req)
			endpoint.handler.ServeHTTP(w, req)
			return
		} else if len(node.methodEndpoints) > 0 {
			var allowedMethods []string
			for m, _ := range node.methodEndpoints {
				allowedMethods = append(allowedMethods, m)
			}
			w.Header()["Allow"] = []string{strings.Join(allowedMethods, ", ")}
			r.MethodNotAllowedHandler.ServeHTTP(w, req)
			return
		}
	}

	r.NotFoundHandler.ServeHTTP(w, req)
}

// AddRoute adds a route for the given HTTP method, path, and handler. path can
// contain parameterized segments. The parameters will be added to the query
// string for requests will routing.
//
// r.AddRoute("GET", "/people", peopleIndexHandler) --> Only matched /people
// r.AddRoute("GET", "/people/:id", peopleIndexHandler) --> matches /people/*
func (r *Router) AddRoute(method string, path string, handler http.Handler) {
	segments := segmentizePath(path)
	parameters := extractParameterNames(segments)
	endpoint := &endpoint{handler: handler, parameters: parameters}
	r.root.addRouteFromSegments(method, segments, endpoint)
}

// Get is a shortcut for AddRoute("GET", path, handler)
func (r *Router) Get(path string, handler http.Handler) {
	r.AddRoute("GET", path, handler)
}

// Post is a shortcut for AddRoute("POST", path, handler)
func (r *Router) Post(path string, handler http.Handler) {
	r.AddRoute("POST", path, handler)
}

// Put is a shortcut for AddRoute("PUT", path, handler)
func (r *Router) Put(path string, handler http.Handler) {
	r.AddRoute("PUT", path, handler)
}

// Patch is a shortcut for AddRoute("PATCH", path, handler)
func (r *Router) Patch(path string, handler http.Handler) {
	r.AddRoute("PATCH", path, handler)
}

// Delete is a shortcut for AddRoute("DELETE", path, handler)
func (r *Router) Delete(path string, handler http.Handler) {
	r.AddRoute("DELETE", path, handler)
}

func (n *node) addRouteFromSegments(method string, segments []string, endpoint *endpoint) {
	if len(segments) > 0 {
		head, tail := segments[0], segments[1:]
		var subnode *node
		if strings.HasPrefix(head, ":") {
			if n.parameterBranch == nil {
				n.parameterBranch = newNode()
			}
			subnode = n.parameterBranch
		} else {
			if _, present := n.staticBranches[head]; !present {
				n.staticBranches[head] = newNode()
			}
			subnode = n.staticBranches[head]
		}
		subnode.addRouteFromSegments(method, tail, endpoint)
	} else {
		n.methodEndpoints[method] = endpoint
	}
}

func (n *node) findNode(segments, pathArguments []string) (*node, []string, bool) {
	if len(segments) > 0 {
		head, tail := segments[0], segments[1:]
		if subnode, present := n.staticBranches[head]; present {
			return subnode.findNode(tail, pathArguments)
		} else if n.parameterBranch != nil {
			pathArguments = append(pathArguments, head)
			return n.parameterBranch.findNode(tail, pathArguments)
		} else {
			return nil, nil, false
		}
	}
	return n, pathArguments, true
}

func segmentizePath(path string) (segments []string) {
	for _, s := range strings.Split(path, "/") {
		if len(s) != 0 {
			segments = append(segments, s)
		}
	}
	return
}

func extractParameterNames(segments []string) (parameters []string) {
	for _, s := range segments {
		if strings.HasPrefix(s, ":") {
			parameters = append(parameters, s[1:])
		}
	}
	return
}

func addRouteArgumentsToRequest(names, values []string, req *http.Request) {
	query := req.URL.Query()
	for i := 0; i < len(names); i++ {
		query.Set(names[i], values[i])
	}
	req.URL.RawQuery = query.Encode()
}

func newNode() (n *node) {
	n = new(node)
	n.methodEndpoints = make(map[string]*endpoint)
	n.staticBranches = make(map[string]*node)
	return
}

func NewRouter() (r *Router) {
	r = new(Router)
	r.root = newNode()
	r.NotFoundHandler = http.HandlerFunc(notFoundHandler)
	r.MethodNotAllowedHandler = http.HandlerFunc(methodNotAllowedHandler)
	return
}

func notFoundHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(404)
	io.WriteString(w, "404 Not Found")
}

func methodNotAllowedHandler(w http.ResponseWriter, req *http.Request) {
	w.WriteHeader(405)
	io.WriteString(w, "405 Method Not Allowed")
}
