FUNCTION_NAME ?= commerce-orders

build:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o bootstrap ./handler

zip: build
	zip function.zip bootstrap

deploy: zip
	aws lambda update-function-code \
	  --function-name $(FUNCTION_NAME) \
	  --zip-file fileb://function.zip

clean:
	rm -f bootstrap function.zip

.PHONY: build zip deploy clean
