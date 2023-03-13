# syntax = docker/dockerfile-upstream:master-labs

#FROM golang:1.20 as go
#WORKDIR /src
#RUN git clone https://github.com/golang/go.git .
#RUN for i in 1.14 1.15 1.16 1.17 1.18 1.19 1.20; do git checkout "go$i"; cd src; ./make.bash; cd ..; cp bin/go "/go/bin/go$i"; cp bin/gofmt "/go/bin/gofmt$i"; done

FROM golang:1.20
WORKDIR /app

COPY go.* ./
RUN go mod download
COPY . ./
RUN go build -o /server cmd/gobinaries-api/main.go
EXPOSE 3000
CMD [ "/server" ]