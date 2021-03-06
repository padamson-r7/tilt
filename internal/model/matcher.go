package model

import (
	"path/filepath"

	"github.com/gobwas/glob"
	"github.com/pkg/errors"
	"github.com/windmilleng/tilt/internal/ospath"
)

type PathMatcher interface {
	Matches(f string, isDir bool) (bool, error)
}

// A Matcher that matches nothing.
type emptyMatcher struct{}

func (m emptyMatcher) Matches(f string, isDir bool) (bool, error) {
	return false, nil
}

var EmptyMatcher PathMatcher = emptyMatcher{}

// A matcher that matches exactly against a set of files.
type fileMatcher struct {
	paths map[string]bool
}

func (m fileMatcher) Matches(f string, isDir bool) (bool, error) {
	return m.paths[f], nil
}

// NewSimpleFileMatcher returns a matcher for the given paths; any relative paths
// are converted to absolute (relative to cwd).
func NewSimpleFileMatcher(paths ...string) (fileMatcher, error) {
	pathMap := make(map[string]bool, len(paths))
	for _, path := range paths {
		// Get the absolute path of the path, because PathMatchers expect to always
		// work with absolute paths.
		path, err := filepath.Abs(path)
		if err != nil {
			return fileMatcher{}, errors.Wrap(err, "NewSimplePathMatcher")
		}
		pathMap[path] = true
	}
	return fileMatcher{paths: pathMap}, nil
}

// This matcher will match a path if it is:
// A. an exact match for one of matcher.paths, or
// B. the child of a path in matcher.paths
// e.g. if paths = {"foo.bar", "baz/"}, will match both
// A. "foo.bar" (exact match), and
// B. "baz/qux" (child of one of the paths)
type fileOrChildMatcher struct {
	paths map[string]bool
}

func (m fileOrChildMatcher) Matches(f string, isDir bool) (bool, error) {
	// (A) Exact match
	if m.paths[f] {
		return true, nil
	}

	// (B) f is child of any of m.paths
	for path := range m.paths {
		if ospath.IsChild(path, f) {
			return true, nil
		}
	}

	return false, nil

}

// NewRelativeFileOrChildMatcher returns a matcher for the given paths (with any
// relative paths converted to absolute, relative to the given baseDir).
func NewRelativeFileOrChildMatcher(baseDir string, paths ...string) fileOrChildMatcher {
	pathMap := make(map[string]bool, len(paths))
	for _, path := range paths {
		if !filepath.IsAbs(path) {
			path = filepath.Join(baseDir, path)
		}
		pathMap[path] = true
	}
	return fileOrChildMatcher{paths: pathMap}
}

// A PathSet stores one or more filepaths, along with the directory that any
// relative paths are relative to
// NOTE(maia): in its current usage (for LiveUpdate.Run.Triggers, LiveUpdate.FallBackOnFiles())
// this isn't strictly necessary, could just as easily convert paths to Abs when specified in
// the Tiltfile--but leaving this code in place for now because it was already written and
// may help with complicated future cases (glob support, etc.)
type PathSet struct {
	Paths         []string
	BaseDirectory string
}

func NewPathSet(paths []string, baseDir string) PathSet {
	return PathSet{
		Paths:         paths,
		BaseDirectory: baseDir,
	}
}

func (ps PathSet) Empty() bool { return len(ps.Paths) == 0 }

// AnyMatch returns true if any of the given filepaths match any paths contained in the pathset
// (along with the first path that matched).
func (ps PathSet) AnyMatch(paths []string) (bool, string, error) {
	matcher := NewRelativeFileOrChildMatcher(ps.BaseDirectory, ps.Paths...)

	for _, path := range paths {
		match, err := matcher.Matches(path, false)
		if err != nil {
			return false, "", err
		}
		if match {
			return true, path, nil
		}
	}
	return false, "", nil
}

type globMatcher struct {
	globs []glob.Glob
}

func (gm globMatcher) Matches(f string, isDir bool) (bool, error) {
	for _, g := range gm.globs {
		if g.Match(f) {
			return true, nil
		}
	}

	return false, nil
}

func NewGlobMatcher(globs ...string) PathMatcher {
	ret := globMatcher{}
	for _, g := range globs {
		ret.globs = append(ret.globs, glob.MustCompile(g))
	}

	return ret
}

type PatternMatcher interface {
	PathMatcher

	// Express this PathMatcher as a sequence of filepath.Match
	// patterns. These patterns are widely useful in Docker-land because
	// they're suitable in .dockerignore or Dockerfile ADD statements
	// https://docs.docker.com/engine/reference/builder/#add
	AsMatchPatterns() []string
}

type CompositePathMatcher struct {
	Matchers []PathMatcher
}

func NewCompositeMatcher(matchers []PathMatcher) PathMatcher {
	if len(matchers) == 0 {
		return EmptyMatcher
	}
	cMatcher := CompositePathMatcher{Matchers: matchers}
	pMatchers := make([]PatternMatcher, len(matchers))
	for i, m := range matchers {
		pm, ok := m.(CompositePatternMatcher)
		if !ok {
			return cMatcher
		}
		pMatchers[i] = pm
	}
	return CompositePatternMatcher{
		CompositePathMatcher: cMatcher,
		Matchers:             pMatchers,
	}
}

func (c CompositePathMatcher) Matches(f string, isDir bool) (bool, error) {
	for _, t := range c.Matchers {
		ret, err := t.Matches(f, isDir)
		if err != nil {
			return false, err
		}
		if ret {
			return true, nil
		}
	}
	return false, nil
}

type CompositePatternMatcher struct {
	CompositePathMatcher
	Matchers []PatternMatcher
}

func (c CompositePatternMatcher) AsMatchPatterns() []string {
	result := []string{}
	for _, m := range c.Matchers {
		result = append(result, m.AsMatchPatterns()...)
	}
	return result
}

var _ PathMatcher = CompositePathMatcher{}
var _ PatternMatcher = CompositePatternMatcher{}
