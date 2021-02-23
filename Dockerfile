FROM golang:latest
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go build -o hlcup2021 .
CMD ["/app/hlcup2021"]
