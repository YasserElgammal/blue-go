package router

import "strings"

type route struct {
	method   string
	segments []segment
	handler  HandlerFunc
	group    *Group
	order    int
}

type segment struct {
	value string
	param bool
}

func parseRoute(path string) (string, []segment) {
	path = cleanPath(path)
	parts := splitPath(path)
	segments := make([]segment, 0, len(parts))
	seenParams := make(map[string]struct{})
	for _, part := range parts {
		item := segment{value: part}
		if strings.HasPrefix(part, ":") {
			name := strings.TrimPrefix(part, ":")
			if name == "" {
				panic("blue: route parameter name cannot be empty")
			}
			if _, exists := seenParams[name]; exists {
				panic("blue: duplicate route parameter " + name)
			}
			seenParams[name] = struct{}{}
			item = segment{value: name, param: true}
		} else if strings.Contains(part, ":") {
			panic("blue: route parameters must occupy a complete path segment")
		}
		segments = append(segments, item)
	}
	return path, segments
}

func cleanPath(path string) string {
	if path == "" {
		return "/"
	}
	if !strings.HasPrefix(path, "/") {
		panic("blue: route path must begin with /")
	}
	if strings.ContainsAny(path, "?#") {
		panic("blue: route path cannot contain a query or fragment")
	}
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func splitPath(path string) []string {
	if path == "/" {
		return nil
	}
	return strings.Split(strings.Trim(path, "/"), "/")
}

func cleanRequestPath(path string) string {
	if path == "" {
		return "/"
	}
	if len(path) > 1 {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}

func (r *route) match(path string) (map[string]string, bool) {
	parts := splitPath(cleanRequestPath(path))
	if len(parts) != len(r.segments) {
		return nil, false
	}
	var params map[string]string
	for i, item := range r.segments {
		if item.param {
			if params == nil {
				params = make(map[string]string)
			}
			params[item.value] = parts[i]
			continue
		}
		if item.value != parts[i] {
			return nil, false
		}
	}
	return params, true
}

func sameShape(a, b []segment) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].param != b[i].param {
			return false
		}
		if !a[i].param && a[i].value != b[i].value {
			return false
		}
	}
	return true
}

func staticCount(segments []segment) int {
	count := 0
	for _, item := range segments {
		if !item.param {
			count++
		}
	}
	return count
}
