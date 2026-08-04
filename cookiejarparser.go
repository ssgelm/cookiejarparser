package cookiejarparser

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
)

const httpOnlyPrefix = "#HttpOnly_"

// Option configures LoadCookieJarFile.
type Option func(*config)

type config struct {
	lenient              bool
	malformedLineHandler func(lineNum int, err error)
}

// WithLenient causes LoadCookieJarFile to skip malformed lines rather than
// returning an error for them.
//
// This covers lines that fail to parse.  If LoadCookieJarFile cannot read the
// file at all it still returns an error, since that leaves the cookie jar
// truncated at an arbitrary point.
func WithLenient() Option {
	return func(c *config) {
		c.lenient = true
	}
}

// WithMalformedLineHandler registers a function to call for each malformed line
// LoadCookieJarFile skips.  Lenient mode gives no other sign that it dropped a
// line, so without a handler you cannot tell what the file lost.  Strict mode
// never calls the handler, since it reports the bad line through its error.
func WithMalformedLineHandler(handler func(lineNum int, err error)) Option {
	return func(c *config) {
		c.malformedLineHandler = handler
	}
}

func parseCookieLine(cookieLine string, lineNum int) (*http.Cookie, error) {
	var err error
	cookieLineHttpOnly := false
	if strings.HasPrefix(cookieLine, httpOnlyPrefix) {
		cookieLineHttpOnly = true
		cookieLine = strings.TrimPrefix(cookieLine, httpOnlyPrefix)
	}

	if strings.HasPrefix(cookieLine, "#") || cookieLine == "" {
		return nil, nil
	}

	cookieFields := strings.Split(cookieLine, "\t")

	if len(cookieFields) < 6 || len(cookieFields) > 7 {
		return nil, fmt.Errorf("incorrect number of fields in line %d.  Expected 6 or 7, got %d.", lineNum, len(cookieFields))
	}

	for i, v := range cookieFields {
		cookieFields[i] = strings.TrimSpace(v)
	}

	cookie := &http.Cookie{
		Domain:   cookieFields[0],
		Path:     cookieFields[2],
		Name:     cookieFields[5],
		HttpOnly: cookieLineHttpOnly,
	}
	cookie.Secure, err = strconv.ParseBool(cookieFields[3])
	if err != nil {
		return nil, fmt.Errorf("could not parse secure field in line %d: %w", lineNum, err)
	}
	expiresInt, err := strconv.ParseInt(cookieFields[4], 10, 64)
	if err != nil {
		return nil, fmt.Errorf("could not parse expiration field in line %d: %w", lineNum, err)
	}
	if expiresInt > 0 {
		cookie.Expires = time.Unix(expiresInt, 0)
	}

	if len(cookieFields) == 7 {
		cookie.Value = cookieFields[6]
	}

	return cookie, nil
}

// LoadCookieJarFile takes a path to a curl (netscape) cookie jar file and crates a go http.CookieJar with the contents
//
// By default LoadCookieJarFile stops at the first malformed line and returns an
// error naming it.  Pass WithLenient to skip those lines instead, and
// WithMalformedLineHandler to see which lines it skipped.
func LoadCookieJarFile(path string, opts ...Option) (http.CookieJar, error) {
	cfg := &config{}
	for _, opt := range opts {
		opt(cfg)
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lineNum := 0
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lineNum++
		cookieLine := scanner.Text()
		cookie, err := parseCookieLine(cookieLine, lineNum)
		if err != nil {
			if !cfg.lenient {
				return nil, err
			}
			if cfg.malformedLineHandler != nil {
				cfg.malformedLineHandler(lineNum, err)
			}
			continue
		}
		if cookie == nil {
			continue
		}

		var cookieScheme string
		if cookie.Secure {
			cookieScheme = "https"
		} else {
			cookieScheme = "http"
		}
		cookieUrl := &url.URL{
			Scheme: cookieScheme,
			Host:   cookie.Domain,
		}

		cookies := jar.Cookies(cookieUrl)
		cookies = append(cookies, cookie)
		jar.SetCookies(cookieUrl, cookies)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return jar, nil
}
