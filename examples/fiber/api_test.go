package main

import (
	"io"
	"net/http"
	"testing"

	"github.com/go-spectest/spectest"
	"github.com/go-spectest/spectest/jsonpath"
	"github.com/gofiber/fiber/v2"
)

func TestGetUserCookieMatching(t *testing.T) {
	spectest.New().
		HandlerFunc(FiberToHandlerFunc(newApp())).
		Get("/user/1234").
		Expect(t).
		Cookies(spectest.NewCookie("CookieForAndy").Value("Andy")).
		Status(http.StatusOK).
		End()
}

func TestGetUserSuccess(t *testing.T) {
	spectest.New().
		HandlerFunc(FiberToHandlerFunc(newApp())).
		Get("/user/1234").
		Expect(t).
		Body(`{"id": "1234", "name": "Andy"}`).
		Status(http.StatusOK).
		End()
}

func TestGetUserSuccessJSONPath(t *testing.T) {
	spectest.New().
		HandlerFunc(FiberToHandlerFunc(newApp())).
		Get("/user/1234").
		Expect(t).
		Assert(jsonpath.Equal(`$.id`, "1234")).
		Status(http.StatusOK).
		End()
}

func TestGetUserNotFound(t *testing.T) {
	spectest.New().
		HandlerFunc(FiberToHandlerFunc(newApp())).
		Get("/user/1515").
		Expect(t).
		Status(http.StatusNotFound).
		End()
}

func FiberToHandlerFunc(app *fiber.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp, err := app.Test(r)
		if err != nil {
			panic(err)
		}
		defer resp.Body.Close()

		// copy headers
		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)

		// copy body
		if _, err := io.Copy(w, resp.Body); err != nil {
			panic(err)
		}
	}
}
