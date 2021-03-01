FROM golang:latest
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN make build
CMD ["/app/hlcup2021"]
