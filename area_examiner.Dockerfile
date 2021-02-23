FROM golang:latest
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go build -o hlcup2021 area_examiner/main.go
CMD ["/app/hlcup2021"]
