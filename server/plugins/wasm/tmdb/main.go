//go:build wasip1

package main

import (
	"fmt"
	"net/url"

	"github.com/extism/go-pdk"
)

func newTMDB() tmdb {
	key, _ := pdk.GetConfig("apikey")
	base, ok := pdk.GetConfig("base_url")
	if !ok {
		base = "https://api.themoviedb.org"
	}
	return tmdb{get: func(path string, query url.Values) ([]byte, error) {
		if query == nil {
			query = url.Values{}
		}
		if len(key) == 32 {
			query.Set("api_key", key)
		}
		req := pdk.NewHTTPRequest(pdk.MethodGet, base+path+"?"+query.Encode())
		if len(key) != 32 {
			req.SetHeader("Authorization", "Bearer "+key)
		}
		res := req.Send()
		switch res.Status() {
		case 200:
			return res.Body(), nil
		case 404:
			return nil, fmt.Errorf("%w: %s", errNotFound, path)
		}
		return nil, fmt.Errorf("tmdb: status %d for %s", res.Status(), path)
	}}
}

func answer(v any, err error) int32 {
	if err == nil {
		err = pdk.OutputJSON(v)
	}
	if err != nil {
		pdk.SetError(err)
		return 1
	}
	return 0
}

//go:wasmexport info
func exportInfo() int32 { return answer(describe(), nil) }

//go:wasmexport fetch
func exportFetch() int32 {
	var in struct{ Arrangement, Ref string }
	if err := pdk.InputJSON(&in); err != nil {
		return answer(nil, err)
	}
	return answer(newTMDB().fetch(in.Arrangement, in.Ref))
}

//go:wasmexport search
func exportSearch() int32 {
	var in struct{ Query string }
	if err := pdk.InputJSON(&in); err != nil {
		return answer(nil, err)
	}
	return answer(newTMDB().search(in.Query))
}

func main() {}
