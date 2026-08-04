# cookiejarparser

cookiejarparser is a Go library that parses a curl (netscape) cookiejar file into a Go http.CookieJar.

## Usage

Assuming you have a netscape/curl style cookie jar made with something like:
```
$ curl -c cookies.txt -v https://github.com
```
That cookiejar can be used when making a web request using the following code:
```golang
package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/ssgelm/cookiejarparser"
)

func main() {
	cookies, err := cookiejarparser.LoadCookieJarFile("cookies.txt")
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Jar: cookies,
	}
	resp, err := client.Get("https://github.com")
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	respData, err := ioutil.ReadAll(resp.Body)
	fmt.Println(string(respData))
}
```

### Malformed lines

By default `LoadCookieJarFile` stops at the first line it cannot parse and
returns an error naming that line:

```
incorrect number of fields in line 4.  Expected 6 or 7, got 4.
```

To skip malformed lines instead and keep whatever else the file contains, pass
`WithLenient`:

```golang
cookies, err := cookiejarparser.LoadCookieJarFile("cookies.txt",
	cookiejarparser.WithLenient())
```

Lenient mode drops those lines without telling you.  To see them, add
`WithMalformedLineHandler`:

```golang
cookies, err := cookiejarparser.LoadCookieJarFile("cookies.txt",
	cookiejarparser.WithLenient(),
	cookiejarparser.WithMalformedLineHandler(func(lineNum int, err error) {
		log.Printf("skipping line %d: %v", lineNum, err)
	}))
```

Lenient mode covers lines that fail to parse.  If `LoadCookieJarFile` cannot
read the file at all it still returns an error, since that leaves the cookie jar
truncated at an arbitrary point.

## License

MIT
