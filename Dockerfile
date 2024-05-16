# imagen para construir
FROM golang AS build
# PRIMERA OPTIMIZACIÓN: 
WORKDIR /usr/src/app
# - go.mod y go.sum solo cambian cuando agregamos librerías.
COPY go.mod ./
# - descargamos dependencias, si estas no cambian en la siguiente construcción esta capa está cacheada.
RUN go mod download && go mod verify
# CONSTRUIMOS APP
# - IMPORTANTE: agregar .dockerignore para copiar solo lo necesario.
COPY . .

# compilamos y generamos binario
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /app/webapp-lnx main.go



# --------------------------------------------
# MONTAMOS CONTENEDOR
# --------------------------------------------

# Imagen base
FROM scratch

# Carpeta de trabajo.
WORKDIR /app

# Copiamos binario
COPY --from=build /app/webapp-lnx .

EXPOSE 3100
# Iniciamos app.
CMD ["/app/webapp-lnx"]
