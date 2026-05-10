module vibex/agent

go 1.22.2

require github.com/openai/openai-go/v3 v3.8.0

replace vibex/generators/memlace => ../generators/memlace

require (
	github.com/chromedp/chromedp v0.10.0
	github.com/gorilla/websocket v1.5.3
	github.com/joho/godotenv v1.5.1
	vibex/generators/memlace v0.0.0-00010101000000-000000000000
)

require (
	github.com/chromedp/cdproto v0.0.0-20240801214329-3f85d328b335 // indirect
	github.com/chromedp/sysutil v1.0.0 // indirect
	github.com/gobwas/httphead v0.1.0 // indirect
	github.com/gobwas/pool v0.2.1 // indirect
	github.com/gobwas/ws v1.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	golang.org/x/sys v0.29.0 // indirect
)
