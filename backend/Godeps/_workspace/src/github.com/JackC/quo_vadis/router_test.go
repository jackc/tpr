package quo_vadis

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

var benchmarkRouter *Router

func getBenchmarkRouter() *Router {
	if benchmarkRouter != nil {
		return benchmarkRouter
	}

	benchmarkRouter := NewRouter()
	handler := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	benchmarkRouter.AddRoute("GET", "/", handler)
	benchmarkRouter.AddRoute("GET", "/foo", handler)
	benchmarkRouter.AddRoute("GET", "/foo/bar", handler)
	benchmarkRouter.AddRoute("GET", "/foo/baz", handler)
	benchmarkRouter.AddRoute("GET", "/foo/bar/baz/quz", handler)
	benchmarkRouter.AddRoute("GET", "/people", handler)
	benchmarkRouter.AddRoute("GET", "/people/search", handler)
	benchmarkRouter.AddRoute("GET", "/people/:id", handler)
	benchmarkRouter.AddRoute("GET", "/users", handler)
	benchmarkRouter.AddRoute("GET", "/users/:id", handler)
	benchmarkRouter.AddRoute("GET", "/widgets", handler)
	benchmarkRouter.AddRoute("GET", "/widgets/important", handler)

	return benchmarkRouter
}

func TestSegmentizePath(t *testing.T) {
	test := func(path string, expected []string) {
		actual := segmentizePath(path)
		if len(actual) != len(expected) {
			t.Errorf("Expected \"%v\" to be segmented into %v, but it actually was %v", path, expected, actual)
			return
		}

		for i := 0; i < len(actual); i++ {
			if actual[i] != expected[i] {
				t.Errorf("Expected \"%v\" to be segmented into %v, but it actually was %v", path, expected, actual)
				return
			}
		}
	}

	test("/", []string{})
	test("/foo", []string{"foo"})
	test("/foo/", []string{"foo"})
	test("/foo/bar", []string{"foo", "bar"})
	test("/foo/bar/", []string{"foo", "bar"})
	test("/foo/bar/baz", []string{"foo", "bar", "baz"})
}

func TestExtractParameterNames(t *testing.T) {
	test := func(segments []string, expected []string) {
		actual := extractParameterNames(segments)
		if len(actual) != len(expected) {
			t.Errorf("Expected \"%v\" to have %v parameters, but it actually had %v", segments, expected, actual)
			return
		}

		for i := 0; i < len(actual); i++ {
			if actual[i] != expected[i] {
				t.Errorf("Expected \"%v\" to have %v parameters, but it actually had %v", segments, expected, actual)
				return
			}
		}
	}

	test([]string{}, []string{})
	test([]string{"foo"}, []string{})
	test([]string{"foo", ":id"}, []string{"id"})
	test([]string{"foo", ":id", "edit"}, []string{"id"})
}

func stubHandler(responseBody string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, responseBody)
		for key, values := range r.URL.Query() {
			fmt.Fprintf(w, " %s: %s", key, values[0])
		}
	})
}

func testRequest(t *testing.T, router *Router, method string, path string, expectedCode int, expectedBody string) *httptest.ResponseRecorder {
	response := httptest.NewRecorder()
	request, err := http.NewRequest(method, "http://example.com"+path, nil)
	if err != nil {
		t.Errorf("Unable to create test %s request for %s", method, path)
	}

	router.ServeHTTP(response, request)
	if response.Code != expectedCode {
		t.Errorf("%s %s: expected HTTP code %d, received %d", method, path, expectedCode, response.Code)
	}
	if response.Body.String() != expectedBody {
		t.Errorf("%s %s: expected HTTP response body \"%s\", received \"%s\"", method, path, expectedBody, response.Body.String())
	}

	return response
}

func TestRouter(t *testing.T) {
	r := NewRouter()
	r.AddRoute("GET", "/", stubHandler("root"))
	r.AddRoute("GET", "/widget", stubHandler("widgetIndex"))
	r.AddRoute("POST", "/widget", stubHandler("widgetCreate"))
	r.AddRoute("GET", "/widget/:id", stubHandler("widgetShow"))
	r.AddRoute("GET", "/widget/:id/edit", stubHandler("widgetEdit"))

	testRequest(t, r, "GET", "/", 200, "root")
	testRequest(t, r, "GET", "/widget", 200, "widgetIndex")
	testRequest(t, r, "POST", "/widget", 200, "widgetCreate")
	testRequest(t, r, "GET", "/widget/1", 200, "widgetShow id: 1")
	testRequest(t, r, "GET", "/widget/1/edit", 200, "widgetEdit id: 1")

	testRequest(t, r, "GET", "/missing", 404, "404 Not Found")
	testRequest(t, r, "GET", "/widget/1/missing", 404, "404 Not Found")

	r.NotFoundHandler = stubHandler("Custom Not Found")
	testRequest(t, r, "GET", "/missing", 200, "Custom Not Found")
}

func TestRouterMethodNotAllowed(t *testing.T) {
	r := NewRouter()
	r.AddRoute("GET", "/", stubHandler("root"))
	r.AddRoute("POST", "/", stubHandler("root"))
	r.AddRoute("GET", "/foo/bar", stubHandler("foobar"))

	response := testRequest(t, r, "BADMETHOD", "/", 405, "405 Method Not Allowed")
	if len(response.HeaderMap["Allow"]) == 0 {
		t.Fatal("Expected Allow header, but it was not set")
	}
	if response.HeaderMap["Allow"][0] != "GET, POST" {
		t.Errorf(`Expected Allow header to be "GET, POST" but it was %v`, response.HeaderMap["Allow"][0])
	}

	testRequest(t, r, "GET", "/foo", 404, "404 Not Found")
	testRequest(t, r, "BADMETHOD", "/foo", 404, "404 Not Found")

	r.MethodNotAllowedHandler = stubHandler("Custom Method Not Allowed")
	testRequest(t, r, "BADMETHOD", "/", 200, "Custom Method Not Allowed")
}

func getBench(b *testing.B, handler http.Handler, path string, expectedCode int) {
	response := httptest.NewRecorder()
	request, err := http.NewRequest("GET", "http://example.com"+path, nil)
	if err != nil {
		b.Fatalf("Unable to create test GET request for %v", path)
	}

	handler.ServeHTTP(response, request)
	if response.Code != expectedCode {
		b.Fatalf("GET %v: expected HTTP code %v, received %v", path, expectedCode, response.Code)
	}
}

func BenchmarkRoutedRequest(b *testing.B) {
	router := getBenchmarkRouter()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		getBench(b, router, "/widgets/important", 200)
	}
}

func BenchmarkFindNodeRoot(b *testing.B) {
	router := getBenchmarkRouter()

	for i := 0; i < b.N; i++ {
		router.root.findNode(segmentizePath("/"), []string{})
	}
}

func BenchmarkFindNodeSegment1(b *testing.B) {
	router := getBenchmarkRouter()

	for i := 0; i < b.N; i++ {
		router.root.findNode(segmentizePath("/foo"), []string{})
	}
}

func BenchmarkFindNodeSegment2(b *testing.B) {
	router := getBenchmarkRouter()

	for i := 0; i < b.N; i++ {
		router.root.findNode(segmentizePath("/people/search"), []string{})
	}
}

func BenchmarkFindNodeSegment2Placeholder(b *testing.B) {
	router := getBenchmarkRouter()

	for i := 0; i < b.N; i++ {
		router.root.findNode(segmentizePath("/people/1"), []string{})
	}
}

func BenchmarkFindNodeSegment4(b *testing.B) {
	router := getBenchmarkRouter()

	for i := 0; i < b.N; i++ {
		router.root.findNode(segmentizePath("/foo/bar/baz/quz"), []string{})
	}
}
