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

// LoadCookieJarFile takes a path to a curl (netscape) cookie jar file and crates a go http.CookieJar with the contents
func LoadCookieJarFile(path string) (http.CookieJar, error) {
	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	line_num := 1
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cookieLineStr := scanner.Text()

		cookieLineHttpOnly := false
		if strings.HasPrefix(cookieLineStr, httpOnlyPrefix) {
			cookieLineHttpOnly = true
			cookieLineStr = strings.TrimPrefix(cookieLineStr, httpOnlyPrefix)
		}

		if strings.HasPrefix(cookieLineStr, "#") || cookieLineStr == "" {
			continue
		}

		cookieFields := strings.Split(cookieLineStr, "\t")

		if len(cookieFields) < 6 || len(cookieFields) > 7 {
			return nil, fmt.Errorf("incorrect number of fields in line %d.  Expected 6 or 7, got %d.", line_num, len(cookieFields))
		}

		for i, v := range cookieFields {
			cookieFields[i] = strings.TrimSpace(v)
		}

		cookie := http.Cookie{
			Domain:   cookieFields[0],
			Path:     cookieFields[2],
			Name:     cookieFields[5],
			HttpOnly: cookieLineHttpOnly,
		}
		cookie.Secure, err = strconv.ParseBool(cookieFields[3])
		if err != nil {
			return nil, err
		}
		expiresInt, err := strconv.ParseInt(cookieFields[4], 10, 64)
		if err != nil {
			return nil, err
		}
		cookie.Expires = time.Unix(expiresInt, 0)

		if len(cookieFields) == 7 {
			cookie.Value = cookieFields[6]
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

		jar.SetCookies(cookieUrl, []*http.Cookie{&cookie})

		line_num++
	}

	return jar, nil
}
