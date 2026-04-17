package main

var providerMap = map[string]string{
	"google":    "Google",
	"facebook":  "Facebook",
	"microsoft": "Microsoft",
	"linkedin":  "LinkedIn",
}

func mapProvider(provider string) (string, bool) {
	name, ok := providerMap[provider]
	return name, ok
}
