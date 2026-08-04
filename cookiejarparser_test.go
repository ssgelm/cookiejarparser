package cookiejarparser

import (
	"bufio"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/net/publicsuffix"
)

func TestParseCookieLine(t *testing.T) {
	// normal
	cookie, err := parseCookieLine("example.com	FALSE	/	FALSE	0	test_cookie	1", 1)
	sampleCookie := &http.Cookie{
		Domain:   "example.com",
		Path:     "/",
		Name:     "test_cookie",
		Value:    "1",
		HttpOnly: false,
		Secure:   false,
	}

	if !reflect.DeepEqual(cookie, sampleCookie) || err != nil {
		c1, _ := json.Marshal(cookie)
		c2, _ := json.Marshal(sampleCookie)

		t.Errorf("Parsing normal cookie failed.  Expected:\n  cookie: %s err: nil,\ngot:\n  cookie: %s err: %s", c2, c1, err)
	}
	// httponly
	cookieHttp, err := parseCookieLine("#HttpOnly_example.com	FALSE	/	FALSE	0	test_cookie_httponly	1", 1)
	sampleCookieHttp := &http.Cookie{
		Domain:   "example.com",
		Path:     "/",
		Name:     "test_cookie_httponly",
		Value:    "1",
		HttpOnly: true,
		Secure:   false,
	}

	if !reflect.DeepEqual(cookieHttp, sampleCookieHttp) || err != nil {
		c1, _ := json.Marshal(cookieHttp)
		c2, _ := json.Marshal(sampleCookieHttp)

		t.Errorf("Parsing httpOnly cookie failed.  Expected:\n  cookie: %s err: nil,\ngot:\n  cookie: %s err: %s", c2, c1, err)
	}

	// comment
	cookieComment, err := parseCookieLine("# This is a comment", 1)
	if cookieComment != nil || err != nil {
		t.Errorf("Parsing comment failed.  Expected cookie: nil err: nil, got cookie: %s err: %s", cookie, err)
	}

	cookieBlank, err := parseCookieLine("", 1)
	if cookieBlank != nil || err != nil {
		t.Errorf("Parsing blank line failed.  Expected cookie: nil err: nil, got cookie: %s err: %s", cookie, err)
	}

	// malformed lines
	malformed := []struct {
		name string
		line string
	}{
		{"too few fields", "example.com	FALSE	/	FALSE"},
		{"too many fields", "example.com	FALSE	/	FALSE	0	test_cookie	1	extra"},
		{"unparsable secure field", "example.com	FALSE	/	NOTABOOL	0	test_cookie	1"},
		{"unparsable expiration field", "example.com	FALSE	/	FALSE	NOTANINT	test_cookie	1"},
	}
	for _, m := range malformed {
		cookieMalformed, err := parseCookieLine(m.line, 42)
		if err == nil {
			t.Errorf("Parsing %s did not fail.  Expected an error, got cookie: %s err: nil", m.name, cookieMalformed)
			continue
		}
		if !strings.Contains(err.Error(), "42") {
			t.Errorf("Error for %s does not reference the line number.  Expected an error mentioning line 42, got: %s", m.name, err)
		}
	}
}

var exampleURL = &url.URL{
	Scheme: "http",
	Host:   "example.com",
}

func TestLoadCookieJarFile(t *testing.T) {
	sampleCookies := []*http.Cookie{
		{
			Domain:   "example.com",
			Path:     "/",
			Name:     "test_cookie",
			Value:    "1",
			HttpOnly: false,
			Secure:   false,
		},
		{
			Domain:   "example.com",
			Path:     "/",
			Name:     "test_cookie_httponly",
			Value:    "1",
			HttpOnly: true,
			Secure:   false,
		},
	}
	sampleCookieJar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		t.Fatalf("Could not create sample cookie jar: %s", err)
	}
	sampleCookieJar.SetCookies(exampleURL, sampleCookies)

	cookieJar, err := LoadCookieJarFile("data/cookies.txt")

	c1, _ := json.Marshal(cookieJar.Cookies(exampleURL))
	c2, _ := json.Marshal(sampleCookieJar.Cookies(exampleURL))

	if !reflect.DeepEqual(c1, c2) || err != nil {
		t.Errorf("Cookie jar creation failed.  Expected:\n  cookieJar: %s err: nil,\ngot:\n  cookieJar: %s err: %s", c2, c1, err)
	}
}

func TestLoadCookieJarFileMalformed(t *testing.T) {
	// The malformed line is line 4, after a comment and a blank line, so the
	// reported number is right only if the parser counts the lines it skips.
	cookieJar, err := LoadCookieJarFile("data/cookies_malformed.txt")
	if err == nil {
		t.Fatalf("Loading a malformed cookie jar did not fail.  Expected an error, got cookieJar: %v err: nil", cookieJar)
	}
	if cookieJar != nil {
		t.Errorf("Loading a malformed cookie jar returned a jar alongside the error.  Expected cookieJar: nil, got: %v", cookieJar)
	}
	if !strings.Contains(err.Error(), "line 4") {
		t.Errorf("Error does not reference the correct line.  Expected an error mentioning line 4, got: %s", err)
	}
}

func TestLoadCookieJarFileScanError(t *testing.T) {
	// A line longer than bufio.Scanner's token limit stops the scan early.  That
	// must surface as an error rather than a cookie jar missing half its file.
	path := filepath.Join(t.TempDir(), "cookies.txt")
	longValue := strings.Repeat("a", bufio.MaxScanTokenSize+1)
	contents := "example.com\tFALSE\t/\tFALSE\t0\ttest_cookie\t" + longValue + "\n"
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("Could not write test cookie file: %s", err)
	}

	cookieJar, err := LoadCookieJarFile(path)
	if err == nil {
		t.Fatalf("Loading a cookie jar with an over-long line did not fail.  Expected an error, got cookieJar: %v err: nil", cookieJar)
	}
	if !errors.Is(err, bufio.ErrTooLong) {
		t.Errorf("Unexpected error loading a cookie jar with an over-long line.  Expected %s, got: %s", bufio.ErrTooLong, err)
	}
}
