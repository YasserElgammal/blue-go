package router

import "strings"

type route struct {
	method   string
	segments []segment
	handler  HandlerFunc
	group    *Group
}

type segment struct {
	value    string
	param    bool
	wildcard bool // greedy catch-all; must be the last segment
}

func parseRoute(path string) (string, []segment) {
	path = cleanPath(path)
	parts := splitPath(path)
	segments := make([]segment, 0, len(parts))
	seenParams := make(map[string]struct{})
	for i, part := range parts {
		item := segment{value: part}
		switch {
		case strings.HasPrefix(part, "*"):
			name := strings.TrimPrefix(part, "*")
			if name == "" {
				panic("blue: wildcard parameter name cannot be empty")
			}
			if i != len(parts)-1 {
				panic("blue: wildcard segment must be the last segment in the route")
			}
			if _, exists := seenParams[name]; exists {
				panic("blue: duplicate route parameter " + name)
			}
			seenParams[name] = struct{}{}
			item = segment{value: name, wildcard: true}
		case strings.HasPrefix(part, ":"):
			name := strings.TrimPrefix(part, ":")
			if name == "" {
				panic("blue: route parameter name cannot be empty")
			}
			if _, exists := seenParams[name]; exists {
				panic("blue: duplicate route parameter " + name)
			}
			seenParams[name] = struct{}{}
			item = segment{value: name, param: true}
		case strings.ContainsAny(part, ":*"):
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
	return trimTrailingSlashes(path)
}

func splitPath(path string) []string {
	if path == "/" {
		return nil
	}
	return strings.Split(strings.TrimPrefix(path, "/"), "/")
}

func cleanRequestPath(path string) string {
	if path == "" {
		return "/"
	}
	return trimTrailingSlashes(path)
}

func trimTrailingSlashes(path string) string {
	path = strings.TrimRight(path, "/")
	if path == "" {
		return "/"
	}
	return path
}

func (r *route) match(parts []string) (map[string]string, bool) {

	// Wildcard route: the last segment greedily captures everything from its
	// position onward, joined with "/".
	if len(r.segments) > 0 && r.segments[len(r.segments)-1].wildcard {
		prefix := r.segments[:len(r.segments)-1]
		if len(parts) < len(prefix) {
			return nil, false
		}
		var params map[string]string
		for i, item := range prefix {
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
		wildcardName := r.segments[len(r.segments)-1].value
		if params == nil {
			params = make(map[string]string)
		}
		params[wildcardName] = strings.Join(parts[len(prefix):], "/")
		return params, true
	}

	// Normal (static / :param) route.
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
	aWild := len(a) > 0 && a[len(a)-1].wildcard
	bWild := len(b) > 0 && b[len(b)-1].wildcard
	// A wildcard route and a non-wildcard route are never ambiguous.
	if aWild != bWild {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		aParam := a[i].param || a[i].wildcard
		bParam := b[i].param || b[i].wildcard
		if aParam != bParam {
			return false
		}
		if !aParam && a[i].value != b[i].value {
			return false
		}
	}
	return true
}

// moreSpecific reports whether a should take precedence over b. Segment
// specificity is compared from left to right: static, parameter, wildcard.
func moreSpecific(a, b []segment) bool {
	for i := 0; i < len(a) && i < len(b); i++ {
		aScore := segmentSpecificity(a[i])
		bScore := segmentSpecificity(b[i])
		if aScore != bScore {
			return aScore > bScore
		}
	}
	return false
}

func segmentSpecificity(item segment) int {
	switch {
	case item.wildcard:
		return 0
	case item.param:
		return 1
	default:
		return 2
	}
}
