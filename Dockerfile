FROM golang:latest


RUN mkdir /app
ADD index.go /app
WORKDIR /app

RUN go get github.com/globalsign/mgo
RUN go get github.com/gorilla/mux
RUN go build index.go

CMD ["./index"]
