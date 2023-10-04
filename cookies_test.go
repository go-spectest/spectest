package spectest

import (
	"net/http"
	"testing"
	"time"
)

func TestApiTestCookiesExpectedCookie(t *testing.T) {
	expiry, _ := time.Parse("1/2/2006 15:04:05", "03/01/2017 12:00:00")

	cookie := NewCookie("Tom").
		Value("LovesBeers").
		Path("/at-the-lyric").
		Domain("in.london").
		Expires(expiry).
		MaxAge(10).
		Secure(true).
		HTTPOnly(false)

	ten := 10
	boolt := true
	boolf := false

	assert.Equal(t, Cookie{
		name:     toString("Tom"),
		value:    toString("LovesBeers"),
		path:     toString("/at-the-lyric"),
		domain:   toString("in.london"),
		expires:  &expiry,
		maxAge:   &ten,
		secure:   &boolt,
		httpOnly: &boolf,
	}, *cookie)
}

func TestApiTestCookiesToHttpCookie(t *testing.T) {
	expiry, _ := time.Parse("1/2/2006 15:04:05", "03/01/2017 12:00:00")

	httpCookie := NewCookie("Tom").
		Value("LovesBeers").
		Path("/at-the-lyric").
		Domain("in.london").
		Expires(expiry).
		MaxAge(10).
		Secure(true).
		HTTPOnly(false).
		ToHTTPCookie()

	assert.Equal(t, http.Cookie{
		Name:     "Tom",
		Value:    "LovesBeers",
		Path:     "/at-the-lyric",
		Domain:   "in.london",
		Expires:  expiry,
		MaxAge:   10,
		Secure:   true,
		HttpOnly: false,
	}, *httpCookie)
}

func TestApiTestCookiesFromHttpCookie(t *testing.T) {
	expiry, _ := time.Parse("1/2/2006 15:04:05", "03/01/2017 12:00:00")

	cookie := NewCookie("Tom").
		Value("LovesBeers").
		Path("/at-the-lyric").
		Domain("in.london").
		Expires(expiry).
		MaxAge(10).
		Secure(true).
		HTTPOnly(false)

	result := FromHTTPCookie(cookie.ToHTTPCookie())

	assert.Equal(t, cookie, result)
}

func TestApiTestCookiesToHttpCookiePartiallyCreated(t *testing.T) {
	expiry, _ := time.Parse("1/2/2006 15:04:05", "03/01/2017 12:00:00")

	httpCookie := NewCookie("Tom").
		Value("LovesBeers").
		Expires(expiry).
		ToHTTPCookie()

	assert.Equal(t, http.Cookie{
		Name:     "Tom",
		Value:    "LovesBeers",
		Expires:  expiry,
		Secure:   false,
		HttpOnly: false,
	}, *httpCookie)
}

func TestCompareCookies(t *testing.T) {
	tests := []struct {
		name       string
		expected   *Cookie
		actual     http.Cookie
		mismatches []string
	}{
		{
			name:       "mismatches value",
			expected:   NewCookie("C").Value("A"),
			actual:     http.Cookie{Name: "C", Value: "V"},
			mismatches: []string{"Mismatched field Value. Expected A but received V"},
		},
		{
			name:       "mismatches domain",
			expected:   NewCookie("C").Value("A").Domain("b.com"),
			actual:     http.Cookie{Name: "C", Value: "A", Domain: "a.com"},
			mismatches: []string{"Mismatched field Domain. Expected b.com but received a.com"},
		},
		{
			name:       "mismatches path",
			expected:   NewCookie("C").Value("A").Path("/"),
			actual:     http.Cookie{Name: "C", Value: "A", Path: "/path"},
			mismatches: []string{"Mismatched field Path. Expected / but received /path"},
		},
		{
			name:       "mismatches expires",
			expected:   NewCookie("C").Value("A").Expires(time.Unix(0, 0).UTC()),
			actual:     http.Cookie{Name: "C", Value: "A", Expires: time.Unix(1, 0).UTC()},
			mismatches: []string{"Mismatched field Expires. Expected 1970-01-01 00:00:00 +0000 UTC but received 1970-01-01 00:00:01 +0000 UTC"},
		},
		{
			name:       "mismatches max age",
			expected:   NewCookie("C").Value("A").MaxAge(0),
			actual:     http.Cookie{Name: "C", Value: "A", MaxAge: 1},
			mismatches: []string{"Mismatched field MaxAge. Expected 0 but received 1"},
		},
		{
			name:       "mismatches max secure",
			expected:   NewCookie("C").Value("A").Secure(true),
			actual:     http.Cookie{Name: "C", Value: "A", Secure: false},
			mismatches: []string{"Mismatched field Secure. Expected true but received false"},
		},
		{
			name:       "mismatches http only",
			expected:   NewCookie("C").Value("A").HTTPOnly(true),
			actual:     http.Cookie{Name: "C", Value: "A", HttpOnly: false},
			mismatches: []string{"Mismatched field HttpOnly. Expected true but received false"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			found, mismatches := compareCookies(test.expected, &test.actual)

			assert.True(t, found)
			assert.Equal(t, test.mismatches, mismatches)
		})
	}
}

func toString(str string) *string {
	return &str
}
