FROM golang:latest
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go build -o hlcup2021 ./experiments/treasure_depth_ordering
CMD ["/app/hlcup2021"]
