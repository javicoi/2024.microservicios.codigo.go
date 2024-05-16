# imagen para construir
FROM golang

# carpeta de trabajo por defecto (cuando referenciamos "./") 
WORKDIR /usr/src/app

COPY . .
RUN go mod download && go mod verify

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/webapp-lnx main.go

CMD ["/app/webapp-lnx"]
