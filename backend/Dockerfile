FROM golang:1.25-alpine
WORKDIR /src
COPY . .
RUN go build -o /out/api .
EXPOSE 8080
CMD ["/out/api"]
