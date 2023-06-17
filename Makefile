.PHONY: offer
offer:
	go run offer/offer.go

.PHONY: answer
answer:
	go run answer/answer.go

.PHONY: build
build:
	CGO_ENABLED=0 go build -o offer/offer offer/offer.go
	CGO_ENABLED=0 go build -o answer/answer answer/answer.go


# .PHONY: docker-offer
# docker-offer:
#   cd offer && docker build  -t offer:latest .
#
# .PHONY: docker-answer
# docker-answer:
#   cd answer && docker build  -t answer:latest .

.PHONY: docker-run
docker-run:
	docker run -it -d -v /home/harshit/go/src/github.com/hvaghani221/webrtc/volume:/volume answer:latest /answer -rootPath "/volume/"

	docker run -it -d -v /home/harshit/go/src/github.com/hvaghani221/webrtc/volume:/volume offer:latest /offer -rootPath "/volume/"

