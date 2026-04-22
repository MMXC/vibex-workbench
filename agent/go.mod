module vibex/agent

go 1.22.2

require github.com/openai/openai-go/v3 v3.8.0

replace vibex/generators/memlace => ../generators/memlace

require github.com/joho/godotenv v1.5.1

require (
	github.com/tidwall/gjson v1.14.4 // indirect
	github.com/tidwall/match v1.1.1 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	vibex/generators/memlace v0.0.0-00010101000000-000000000000 // indirect
)
